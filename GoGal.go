package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/captainju/gogal/util"
	"github.com/joho/godotenv"
	"github.com/nfnt/resize"
	"github.com/rwcarlsen/goexif/exif"
	"image/jpeg"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"
)

var (
	wg                    sync.WaitGroup
	mongoPhotoStore       util.MongoPhotoStore
	s3Manager             util.S3Manager
	cloudFrontManager     util.CloudFrontManager
	imageSourceFolderPath string
)

func main() {
	log.Println("Initializing...")
	loadEnvVars()
	initS3Manager()
	initMongoPhotoStore()
	initCloudFrontManager()
	log.Println("Init ok")

	back := flag.Bool("back", false, "detect, resize and upload pictures")
	eraseDB := flag.Bool("erasedb", false, "if running in back mode, replace data in DB")
	flag.Parse()

	if *back {
		runAsBack(*eraseDB)
	} else {
		runAsFront()
	}
}

func runAsFront() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))
	serveSingle("/", "static/main.html")
	http.HandleFunc("/albums.json", albumsHandler)
	http.HandleFunc("/images.json", imagesHandler)

	log.Println(http.ListenAndServe(":8080", nil))
}

func albumsHandler(w http.ResponseWriter, r *http.Request) {
	albums := []string{}
PhtotSreamLoop:
	for photo := range mongoPhotoStore.PhotoStream() {
		timestamp := strconv.FormatInt(photo.AlbumDateTime.Unix(), 10)
		for _, b := range albums {
			if b == timestamp {
				continue PhtotSreamLoop
			}
		}
		albums = append(albums, timestamp)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(albums)))
	slcB, _ := json.Marshal(albums)
	w.Header().Set("Content-Type", "application/javascript")
	fmt.Fprintf(w, string(slcB))
}

func imagesHandler(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()

	photos := []util.Photo{}

	for photo := range mongoPhotoStore.PhotoStream() {
		photoAlbumTimestamp := strconv.FormatInt(photo.AlbumDateTime.Unix(), 10)
		for _, albumTimestamp := range r.Form["albums"] {
			if albumTimestamp == photoAlbumTimestamp {
				photo.ThumbUrl = cloudFrontManager.SignUrl(cloudFrontManager.BaseUrl + "/" + s3Manager.ThumbPath + photo.Filename)
				photo.MediumUrl = cloudFrontManager.SignUrl(cloudFrontManager.BaseUrl + "/" + s3Manager.MediumPath + photo.Filename)
				photos = append(photos, photo)
			}
		}
	}

	sort.Sort(util.ByDateTime(photos))
	slcB, _ := json.Marshal(photos)
	w.Header().Set("Content-Type", "application/javascript")
	fmt.Fprintf(w, string(slcB))
}

func serveSingle(pattern string, filename string) {
	http.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filename)
	})
}

func runAsBack(eraseDb bool) {
	var res []os.FileInfo
	var err error
	res, err = ioutil.ReadDir(imageSourceFolderPath)
	if err != nil {
		panic(err)
	}

	if eraseDb {
		mongoPhotoStore.EraseDB()
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

func initCloudFrontManager() {
	cloudFrontManager = util.CloudFrontManager{
		BaseUrl:        os.Getenv("CLOUDFRONT_BASE_URL"),
		PrivateKeyFile: os.Getenv("CLOUDFRONT_PRIVATE_KEY_FILE"),
		KeyId:          os.Getenv("CLOUDFRONT_KEY_ID"),
		Expiration:     1,
	}
	if cloudFrontManager.BaseUrl == "" || cloudFrontManager.PrivateKeyFile == "" || cloudFrontManager.KeyId == "" {
		panic("Error CloudFront : not configured")
	}
	err := cloudFrontManager.Init()
	if err != nil {
		panic("Error CloudFront : " + err.Error())
	}
	log.Println("CloudFront ok")
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
