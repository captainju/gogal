package util

import (
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"io"
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

func (uploader *S3Manager) Connect() error {

	uploader.svc = s3.New(&aws.Config{
		Region: aws.String(uploader.Region),
	})

	uploader.queue = make(chan (bool), uploader.NbConcurrentUploads)

	params := &s3.HeadBucketInput{
		Bucket: aws.String(uploader.Bucket), // Required
	}
	_, err := uploader.svc.HeadBucket(params)
	return err
}

func (uploader S3Manager) UploadImage(rs io.ReadSeeker, fileName string) (url string, err error) {
	return uploader.upload(rs, fileName, uploader.ImagePath)
}

func (uploader S3Manager) UploadThumb(rs io.ReadSeeker, fileName string) (url string, err error) {
	return uploader.upload(rs, fileName, uploader.ThumbPath)
}

func (uploader S3Manager) UploadMedium(rs io.ReadSeeker, fileName string) (url string, err error) {
	return uploader.upload(rs, fileName, uploader.MediumPath)
}

func (uploader S3Manager) upload(rs io.ReadSeeker, fileName string, path string) (url string, err error) {
	defer func() { <-uploader.queue }()
	uploader.queue <- true

	filePath := path + fileName

	if uploader.svc == nil {
		return "", errors.New("S3Manager not initialized, Connect should be called first")
	}

	params := &s3.PutObjectInput{
		Bucket:      aws.String(uploader.Bucket), // Required
		Key:         aws.String(filePath),        // Required
		Body:        rs,
		ContentType: aws.String(dataType),
	}

	resp, err := uploader.svc.PutObject(params)

	return resp.String(), err
}

func (uploader S3Manager) ExistsImage(fileName string) (exists bool) {
	return uploader.exists(fileName, uploader.ImagePath)
}

func (uploader S3Manager) ExistsThumb(fileName string) (exists bool) {
	return uploader.exists(fileName, uploader.ThumbPath)
}

func (uploader S3Manager) ExistsMedium(fileName string) (exists bool) {
	return uploader.exists(fileName, uploader.MediumPath)
}

func (uploader S3Manager) exists(fileName string, path string) (exists bool) {
	filePath := path + fileName

	params := &s3.HeadObjectInput{
		Bucket: aws.String(uploader.Bucket), // Required
		Key:    aws.String(filePath),        // Required
	}
	_, err := uploader.svc.HeadObject(params)

	if err != nil {
		return false
	}

	return true
}
