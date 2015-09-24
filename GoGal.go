package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/captainju/gogal/util"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/nfnt/resize"
	"github.com/rwcarlsen/goexif/exif"
	"image/jpeg"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	wg                    sync.WaitGroup
	mongoPhotoStore       util.MongoPhotoStore
	s3Manager             util.S3Manager
	imageSourceFolderPath string
)

func main() {
	log.Println("Initializing...")
	loadEnvVars()
	initS3Manager()
	initMongoPhotoStore()
	log.Println("Init ok")

	back := flag.Bool("back", false, "detect, resize and upload pictures")
	flag.Parse()

	if *back {
		runAsBack()
	} else {
		runAsFront()
	}
}

func runAsFront() {

	/*for photo := range mongoPhotoStore.PhotoStream() {
		log.Println(photo.Filename)
	}*/

	r := mux.NewRouter()
	r.HandleFunc("/", HomeHandler)
	http.Handle("/", r)
	log.Println(http.ListenAndServe(":8080", nil))

}

func HomeHandler(w http.ResponseWriter, r *http.Request) {

	fmt.Fprintf(w, "woot !")

}

func runAsBack() {
	var res []os.FileInfo
	var err error
	res, err = ioutil.ReadDir(imageSourceFolderPath)
	if err != nil {
		panic(err)
	}
	for _, fileInfo := range res {
		if !fileInfo.IsDir() {
			wg.Add(1)
			go handleFile(fileInfo.Name())
		}
	}
	wg.Wait()
}

func loadEnvVars() {
	godotenv.Load()
	imageSourceFolderPath = os.Getenv("IMAGE_SOURCE_FOLDER_PATH")
	if imageSourceFolderPath == "" {
		panic("image source folder path not configured")
	}
	_, err := os.Open(imageSourceFolderPath)
	if err != nil {
		panic("can't access image source folder path : " + err.Error())
	}
	log.Println("image folder ok")
}

func initS3Manager() {
	s3Manager = util.S3Manager{
		Bucket:              os.Getenv("S3_BUCKET"),
		Region:              os.Getenv("S3_REGION"),
		ImagePath:           os.Getenv("S3_IMAGE_FOLDER_PATH"),
		ThumbPath:           os.Getenv("S3_THUMB_FOLDER_PATH"),
		MediumPath:          os.Getenv("S3_MEDIUM_FOLDER_PATH"),
		NbConcurrentUploads: 2,
	}
	err := s3Manager.Connect()
	if err != nil {
		panic("Error S3 : " + err.Error())
	}
	log.Println("S3 ok")
}

func initMongoPhotoStore() {
	mongoPhotoStore = util.MongoPhotoStore{
		Url:            os.Getenv("MONGODB_URL"),
		DbName:         os.Getenv("MONGODB_DB_NAME"),
		CollectionName: os.Getenv("MONGODB_COLLECTION_NAME"),
	}
	err := mongoPhotoStore.Ping()
	if err != nil {
		panic("Error mongodb : " + err.Error())
	}
	log.Println("MongoDB ok")
}

func handleFile(sourceFilename string) {
	defer wg.Done()

	photo, err := mongoPhotoStore.LookupPhoto(sourceFilename)
	if err != nil {
		photo, err = createPhoto(sourceFilename)
		if err != nil {
			log.Printf("Can't create photo from %s : %s\n", sourceFilename, err.Error())
			return
		}
		err = mongoPhotoStore.StorePhoto(photo)
		if err != nil {
			log.Printf("Can't store photo from %s : %s\n", sourceFilename, err.Error())
			return
		}
	}

	if !s3Manager.ExistsImage(sourceFilename) {
		f, err := os.Open(imageSourceFolderPath + sourceFilename)
		if err != nil {
			log.Println(err)
		} else {
			log.Printf("Uploading %s", sourceFilename)
			s3Manager.UploadImage(f, sourceFilename)
			log.Printf("%s successfully uploaded", sourceFilename)
		}
	}

	if !s3Manager.ExistsThumb(sourceFilename) {
		f, err := os.Open(imageSourceFolderPath + sourceFilename)
		if err != nil {
			log.Println(err)
		} else {
			log.Printf("Resizing %s", sourceFilename)
			buf := bytes.NewBuffer(make([]byte, 0))
			resizeImg(f, buf, 0, 162)
			log.Printf("%s successfully resized", sourceFilename)
			log.Printf("Uploading thumb %s", sourceFilename)
			r := bytes.NewReader(buf.Bytes())
			s3Manager.UploadThumb(r, sourceFilename)
			log.Printf("%s thumb successfully uploaded", sourceFilename)
		}
	}

	if !s3Manager.ExistsMedium(sourceFilename) {
		f, err := os.Open(imageSourceFolderPath + sourceFilename)
		if err != nil {
			log.Println(err)
		} else {
			log.Printf("Resizing %s", sourceFilename)
			buf := bytes.NewBuffer(make([]byte, 0))
			resizeImg(f, buf, 0, 768)
			log.Printf("%s successfully resized", sourceFilename)
			log.Printf("Uploading medium %s", sourceFilename)
			r := bytes.NewReader(buf.Bytes())
			s3Manager.UploadMedium(r, sourceFilename)
			log.Printf("%s medium successfully uploaded", sourceFilename)
		}
	}
}

func createPhoto(sourceFilename string) (util.Photo, error) {
	photo := util.Photo{}

	f, err := os.Open(imageSourceFolderPath + sourceFilename)
	if err != nil {
		log.Println(err)
		return photo, err
	}

	x, err := exif.Decode(f)
	if err != nil {
		log.Println(sourceFilename, err)
		return photo, err
	}
	tm, _ := x.DateTime()
	if false {
		log.Println("Taken: ", tm)
	}

	albumTime := time.Date(tm.Year(), tm.Month(), tm.Day(), 0, 0, 0, 0, time.UTC)
	photo.AlbumDateTime = albumTime
	photo.DateTime = tm
	photo.Filename = sourceFilename
	return photo, nil
}

func resizeImg(r io.Reader, w io.Writer, width uint, height uint) {
	// decode jpeg into image.Image
	img, err := jpeg.Decode(r)
	if err != nil {
		log.Println(err)
		return
	}

	// resize using Lanczos resampling
	// and preserve aspect ratio
	m := resize.Resize(width, height, img, resize.Lanczos3)

	// write new image to file
	jpeg.Encode(w, m, nil)
}
