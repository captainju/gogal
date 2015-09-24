package util

import "time"

type Photo struct {
	DateTime      time.Time
	AlbumDateTime time.Time
	Filename      string
}
