package backends

import "image"

type MediaInfo struct {
	Filename  string
	Duration  float64
	Framerate string
	Width     int
	Height    int
	Format    string
	Filesize  int
}

type MediaProcessor interface {
	GetInfo(filename string) (MediaInfo, error)
	ExtractFrame(filename string, timestamp float64, thumbWidth int, thumbHeight int) (image.Image, error)
}
