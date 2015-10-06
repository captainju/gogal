package util

import (
	"encoding/json"
	"strconv"
	"time"
)

type Photo struct {
	DateTime      time.Time
	AlbumDateTime time.Time
	Filename      string
	ThumbUrl      string `json:",omitempty"`
	MediumUrl     string `json:",omitempty"`
}

type ByDateTime []Photo

func (a ByDateTime) Len() int           { return len(a) }
func (a ByDateTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByDateTime) Less(i, j int) bool { return a[i].DateTime.Unix() > a[j].DateTime.Unix() }

func (p Photo) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		DateTime      string
		AlbumDateTime string
		Filename      string
		ThumbUrl      string
		MediumUrl     string
	}{
		DateTime:      strconv.FormatInt(p.DateTime.Unix(), 10),
		AlbumDateTime: strconv.FormatInt(p.AlbumDateTime.Unix(), 10),
		Filename:      p.Filename,
		ThumbUrl:      p.ThumbUrl,
		MediumUrl:     p.MediumUrl,
	})
}
