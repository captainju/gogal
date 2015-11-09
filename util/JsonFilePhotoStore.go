package util

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"sync"
)

type JsonFilePhotoStore struct {
	FileName string
	photos   []Photo
	mutex    sync.Mutex
}

func (jfps *JsonFilePhotoStore) LoadFromFile() error {
	jfps.mutex.Lock()
	defer jfps.mutex.Unlock()
	contentBytes, err := ioutil.ReadFile(jfps.FileName)
	if err != nil {
		return err
	}
	return json.Unmarshal(contentBytes, &jfps.photos)
}

func (jfps *JsonFilePhotoStore) StoreToFile() error {
	jfps.mutex.Lock()
	defer jfps.mutex.Unlock()
	bytes, err := json.Marshal(jfps.photos)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(jfps.FileName, bytes, os.FileMode(0644))
}

func (jfps *JsonFilePhotoStore) Touch() error {
	jfps.mutex.Lock()
	defer jfps.mutex.Unlock()
	if jfps.FileName == "" {
		return errors.New("The filename is empty")
	}
	_, err := os.Open(jfps.FileName)
	if err != nil {
		//try to create it
		return ioutil.WriteFile(jfps.FileName, []byte("[]"), os.FileMode(0644))
	}
	return nil
}

func (jfps *JsonFilePhotoStore) GetAll() []Photo {
	return jfps.photos
}

func (jfps *JsonFilePhotoStore) Get(fileName string) (Photo, error) {
	for _, photo := range jfps.photos {
		if photo.Filename == fileName {
			return photo, nil
		}
	}
	return Photo{}, errors.New("No photo found for filename " + fileName)
}

func (jfps *JsonFilePhotoStore) Add(photo Photo) error {
	jfps.mutex.Lock()
	defer jfps.mutex.Unlock()
	if _, err := jfps.Get(photo.Filename); err == nil {
		return errors.New("Filename already exists")
	}
	jfps.photos = append(jfps.photos, photo)
	return nil
}

func (jfps *JsonFilePhotoStore) Remove(photoToRemove Photo) error {
	jfps.mutex.Lock()
	defer jfps.mutex.Unlock()
	for i := len(jfps.photos) - 1; i >= 0; i-- {
		photo := jfps.photos[i]
		// Condition to decide if current element has to be deleted:
		if photo == photoToRemove {
			jfps.photos = append(jfps.photos[:i], jfps.photos[i+1:]...)
			return nil
		}
	}
	return errors.New("Photo not in list")
}
