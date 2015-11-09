package util

import (
	"testing"
)

const filename string = "/tmp/testjsonFilePhotoStore.json"

func photoFixture(filename string) Photo {
	return Photo{Filename: filename, AlbumDateTime: 1, DateTime: 2}
}

func TestAdd(t *testing.T) {
	photo1 := photoFixture("filename")
	photo2 := photoFixture("filename")

	jsonFilePhotoStore := JsonFilePhotoStore{FileName: filename}
	err := jsonFilePhotoStore.Add(photo1)
	if err != nil {
		t.Error(err)
	}
	err = jsonFilePhotoStore.Add(photo2)
	if err == nil {
		t.Error("Filename already in, should not be added")
	}
}

func TestGet(t *testing.T) {
	photo1 := photoFixture("filename1")
	photo2 := photoFixture("filename2")
	jsonFilePhotoStore := JsonFilePhotoStore{FileName: filename}
	jsonFilePhotoStore.Add(photo1)
	jsonFilePhotoStore.Add(photo2)

	photo, err := jsonFilePhotoStore.Get(photo1.Filename)
	if err != nil {
		t.Error(err)
	}
	if photo != photo1 {
		t.Error("Not the same photos")
	}
}

func TestRemove(t *testing.T) {
	photo1 := photoFixture("filename1")
	photo2 := photoFixture("filename2")
	jsonFilePhotoStore := JsonFilePhotoStore{FileName: filename}
	jsonFilePhotoStore.Add(photo1)
	jsonFilePhotoStore.Add(photo2)

	err := jsonFilePhotoStore.Remove(photo1)
	if err != nil {
		t.Error(err)
	}

	_, err = jsonFilePhotoStore.Get(photo1.Filename)
	if err == nil {
		t.Error("Photo1 should be removed")
	}

	photo, err2 := jsonFilePhotoStore.Get(photo2.Filename)
	if err2 != nil || photo != photo2 {
		t.Error("Photo2 should remain in collection")
	}

}

func TestStoreToFileAndRestore(t *testing.T) {

	photo1 := photoFixture("filename1")
	photo2 := photoFixture("filename2")

	//Touch
	jsonFilePhotoStore := JsonFilePhotoStore{FileName: filename}
	err := jsonFilePhotoStore.Touch()
	if err != nil {
		t.Error(err)
	}

	//store
	jsonFilePhotoStore = JsonFilePhotoStore{FileName: filename}
	jsonFilePhotoStore.Add(photo1)
	jsonFilePhotoStore.Add(photo2)
	err = jsonFilePhotoStore.StoreToFile()
	if err != nil {
		t.Error(err)
	}

	//load
	jsonFilePhotoStore = JsonFilePhotoStore{FileName: filename}
	err = jsonFilePhotoStore.LoadFromFile()
	if err != nil {
		t.Error(err)
	}
	allPhotos := jsonFilePhotoStore.GetAll()
	if len(allPhotos) != 2 {
		t.Error("not enough photos retreived")
	}
	if allPhotos[0] != photo1 || allPhotos[1] != photo2 {
		t.Error("not the same photos retreived")
	}
}
