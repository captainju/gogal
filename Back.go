package main
import (
	"os"
	"fmt"
	"io/ioutil"
	"log"
	"github.com/rwcarlsen/goexif/exif"
	"image/jpeg"
	"github.com/nfnt/resize"
	"sync"
)

var (
	thumbFolderPath = "/images/thumb/"
	imageFolderPath = "/images/"
	nbWorkers = make(chan bool, 4)
	wg sync.WaitGroup
)

func main() {

	var res []os.FileInfo
	var err error
	res, err = ioutil.ReadDir(imageFolderPath)
	if err != nil {
		panic(err)
	}
	for _, fileInfo := range res {
		if !fileInfo.IsDir() {
			f, err := os.Open(imageFolderPath+fileInfo.Name())
			if err != nil {
				log.Println(err)
				continue
			}

			x, err := exif.Decode(f)
			if err != nil {
				log.Println(fileInfo.Name(), err)
				continue
			}
			tm, _ := x.DateTime()
			if false {
				fmt.Println("Taken: ", tm)
			}

			wg.Add(1)
			go GenerateThumb(fileInfo.Name())

		}
	}
	 wg.Wait()
}

func GenerateThumb(sourceFilename string) {
	defer wg.Done()
	nbWorkers <- true
	defer func() { <-nbWorkers }()
	srcFile, err := os.Open(imageFolderPath+sourceFilename)
	if err != nil {
		log.Println(err)
		return
	}

	// decode jpeg into image.Image
	img, err := jpeg.Decode(srcFile)
	if err != nil {
		log.Println(err)
		return
	}
	srcFile.Close()

	// resize to width 1000 using Lanczos resampling
	// and preserve aspect ratio
	m := resize.Resize(0, 162, img, resize.Lanczos3)

	out, err := os.Create(thumbFolderPath+sourceFilename)
	if err != nil {
		log.Println(err)
		return
	}
	defer out.Close()

	// write new image to file
	jpeg.Encode(out, m, nil)
	fmt.Println("Generated ", sourceFilename)

}
