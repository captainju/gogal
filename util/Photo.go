package util

type Photo struct {
	DateTime      int
	AlbumDateTime int
	Filename      string
	ThumbUrl      string `json:",omitempty"`
	MediumUrl     string `json:",omitempty"`
}

type ByDateTime []Photo

func (a ByDateTime) Len() int           { return len(a) }
func (a ByDateTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByDateTime) Less(i, j int) bool { return a[i].DateTime > a[j].DateTime }
