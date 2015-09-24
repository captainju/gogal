package util

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const filenameProperty string = "filename"

type MongoPhotoStore struct {
	Url            string
	DbName         string
	CollectionName string
	collection     *mgo.Collection
	session        *mgo.Session
}

func (mpr *MongoPhotoStore) Ping() error {
	session, err := mgo.Dial(mpr.Url)
	if err != nil {
		return err
	}
	session.Close()
	return nil
}

func (mpr *MongoPhotoStore) getConnection() *mgo.Collection {
	if mpr.collection == nil {
		session, mgoerr := mgo.Dial(mpr.Url)
		if mgoerr != nil {
			panic(mgoerr)
		}
		mpr.session = session
		mpr.collection = session.DB(mpr.DbName).C(mpr.CollectionName)
	}
	return mpr.collection
}

func (mpr *MongoPhotoStore) closeConnection() {
	if mpr.session != nil {
		mpr.session.Close()
		mpr.session = nil
		mpr.collection = nil
	}
}

func (mpr *MongoPhotoStore) StorePhoto(photo Photo) error {
	result := Photo{}
	con := mpr.getConnection()
	err := con.Find(bson.M{filenameProperty: photo.Filename}).One(&result)
	if err != nil {
		return con.Insert(photo)
	}
	return nil
}

func (mpr *MongoPhotoStore) LookupPhoto(fileName string) (Photo, error) {
	result := Photo{}
	con := mpr.getConnection()
	err := con.Find(bson.M{filenameProperty: fileName}).One(&result)
	return result, err
}

func (mpr *MongoPhotoStore) PhotoStream() chan Photo {

	photoStream := make(chan Photo, 1)

	go func() {
		p := &Photo{}
		iter := mpr.getConnection().Find(nil).Iter()
		for iter.Next(&p) {
			photoStream <- *p
		}
		close(photoStream)
		if err := iter.Close(); err != nil {
			panic(err)
		}
	}()

	return photoStream
}
