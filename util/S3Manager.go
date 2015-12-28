package util

import (
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"io"
	"log"
	"sort"
	"sync"
)

const dataType string = "image/jpeg"
const s3RootUrl = "https://s3.amazonaws.com"

type S3Manager struct {
	Bucket              string
	Region              string
	ImagePath           string
	ThumbPath           string
	MediumPath          string
	mutex               *sync.Mutex
	existingFiles       []string
	NbConcurrentUploads int
	svc                 *s3.S3
	queue               chan (bool)
}

func (manager *S3Manager) Connect() error {

	manager.svc = s3.New(&aws.Config{
		Region: aws.String(manager.Region),
	})

	manager.queue = make(chan (bool), manager.NbConcurrentUploads)
	manager.mutex = &sync.Mutex{}

	params := &s3.HeadBucketInput{
		Bucket: aws.String(manager.Bucket), // Required
	}
	_, err := manager.svc.HeadBucket(params)
	return err
}

func (manager S3Manager) UploadImage(rs io.ReadSeeker, fileName string) (url string, err error) {
	return manager.upload(rs, fileName, manager.ImagePath, "image")
}

func (manager S3Manager) UploadThumb(rs io.ReadSeeker, fileName string) (url string, err error) {
	return manager.upload(rs, fileName, manager.ThumbPath, "thumb")
}

func (manager S3Manager) UploadMedium(rs io.ReadSeeker, fileName string) (url string, err error) {
	return manager.upload(rs, fileName, manager.MediumPath, "medium")
}

func (manager S3Manager) upload(rs io.ReadSeeker, fileName string, path string, imageType string) (url string, err error) {
	defer func() { <-manager.queue }()
	manager.queue <- true

	log.Printf("Uploading %s %s", imageType, fileName)

	filePath := path + fileName

	if manager.svc == nil {
		return "", errors.New("S3Manager not initialized, Connect should be called first")
	}

	params := &s3.PutObjectInput{
		Bucket:      aws.String(manager.Bucket), // Required
		Key:         aws.String(filePath),       // Required
		Body:        rs,
		ContentType: aws.String(dataType),
	}

	resp, err := manager.svc.PutObject(params)

	log.Printf("%s %s successfully uploaded", imageType, fileName)

	return resp.String(), err
}

func (manager *S3Manager) ExistsImage(fileName string) (exists bool, err error) {
	return manager.exists(fileName, manager.ImagePath)
}

func (manager *S3Manager) ExistsThumb(fileName string) (exists bool, err error) {
	return manager.exists(fileName, manager.ThumbPath)
}

func (manager *S3Manager) ExistsMedium(fileName string) (exists bool, err error) {
	return manager.exists(fileName, manager.MediumPath)
}

func (manager *S3Manager) exists(fileName string, path string) (exists bool, err error) {
	initError := manager.initExistingFiles()
	if initError != nil {
		return false, initError
	}

	filePath := path + fileName
	i := sort.Search(len(manager.existingFiles), func(i int) bool { return manager.existingFiles[i] >= filePath })
	if i < len(manager.existingFiles) && manager.existingFiles[i] == filePath {
		return true, nil
	}

	return false, nil
}

func (manager *S3Manager) initExistingFiles() error {
	defer manager.mutex.Unlock()
	manager.mutex.Lock()

	if len(manager.existingFiles) == 0 {
		log.Printf("Retrieving all images from S3")
		params := &s3.ListObjectsInput{
			Bucket:  aws.String(manager.Bucket), // Required
			MaxKeys: aws.Int64(1000),
		}
		err := manager.svc.ListObjectsPages(params, func(p *s3.ListObjectsOutput, lastPage bool) bool {
			for _, object := range p.Contents {
				manager.existingFiles = append(manager.existingFiles, *object.Key)
			}
			return true
		})
		if err != nil {
			return err
		}
		sort.Strings(manager.existingFiles)
		log.Printf("%d images retrieved from S3", len(manager.existingFiles))
	}
	return nil
}

func (manager S3Manager) BucketURL() string {
	return manager.svc.Endpoint + "/" + manager.Bucket + "/"
}
