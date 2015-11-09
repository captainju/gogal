package util

import (
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"io"
	"log"
)

const dataType string = "image/jpeg"
const s3RootUrl = "https://s3.amazonaws.com"

type S3Manager struct {
	Bucket              string
	Region              string
	ImagePath           string
	ThumbPath           string
	MediumPath          string
	NbConcurrentUploads int
	svc                 *s3.S3
	queue               chan (bool)
}

func (manager *S3Manager) Connect() error {

	manager.svc = s3.New(&aws.Config{
		Region: aws.String(manager.Region),
	})

	manager.queue = make(chan (bool), manager.NbConcurrentUploads)

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

func (manager S3Manager) ExistsImage(fileName string) (exists bool) {
	return manager.exists(fileName, manager.ImagePath)
}

func (manager S3Manager) ExistsThumb(fileName string) (exists bool) {
	return manager.exists(fileName, manager.ThumbPath)
}

func (manager S3Manager) ExistsMedium(fileName string) (exists bool) {
	return manager.exists(fileName, manager.MediumPath)
}

func (manager S3Manager) exists(fileName string, path string) (exists bool) {
	filePath := path + fileName

	params := &s3.HeadObjectInput{
		Bucket: aws.String(manager.Bucket), // Required
		Key:    aws.String(filePath),       // Required
	}
	_, err := manager.svc.HeadObject(params)

	if err != nil {
		return false
	}

	return true
}

func (manager S3Manager) BucketURL() string {
	return manager.svc.Endpoint + "/" + manager.Bucket + "/"
}
