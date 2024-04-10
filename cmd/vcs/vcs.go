package vcs

import (
	"fmt"
	"github.com/adrg/sysfont"
	"github.com/c2h5oh/datasize"
	"github.com/fogleman/gg"
	"govcs/backends"
	"image"
	"strconv"
	"strings"
	"sync"
)

type Config struct {
	Pics            int
	Columns         int
	Rows            int
	ImagePadding    int
	FontSize        int
	ThumbHeight     int
	ThumbWidth      int
	FontFamily      string
	Verbose         bool
	Quiet           bool
	Format          string
	BgColor         string
	FontColor       string
	FFmpeg          string
	FFprobe         string
	Overwrite       bool
	IgnoreExtension bool
	JpegQuality     int
}

type FrameInfo struct {
	Timestamp string
	Image     *image.Image
	Error     error
}

func CreateVCS(filename string, config Config) error {
	if !config.Quiet {
		fmt.Printf("processing %s\n", filename)
	}

	// Resolv the processor backend
	var provider backends.MediaProcessor
	provider = backends.Ffmpeg{FFmpeg: config.FFmpeg, FFprobe: config.FFprobe}

	info, err := provider.GetInfo(filename)
	if err != nil {
		return err
	}

	// Calculate from media
	config.ThumbWidth = int(float64(info.Width) / float64(info.Height) * float64(config.ThumbHeight))
	config.Rows = config.Pics / config.Columns
	if config.Pics%config.Columns > 0 {
		config.Rows += 1
	}

	frames, err := generateFrames(config, info, provider)
	if err != nil {
		return err
	}

	err = drawImage(config, info, frames)
	if err != nil {
		return err
	}

	return nil
}

func generateFrames(config Config, info backends.MediaInfo, provider backends.MediaProcessor) ([]FrameInfo, error) {
	frameInfos := make([]FrameInfo, config.Pics)
	stepDuration := info.Duration / float64(config.Pics+2)

	var wg sync.WaitGroup
	wg.Add(config.Pics)

	for i := 0; i < config.Pics; i++ {
		frameIdx := i // May be fixed in go 1.22
		go func() {
			defer wg.Done()

			timestamp := stepDuration * float64(frameIdx+1)

			if config.Verbose {
				fmt.Printf("extracting frame %v/%v (%s)\n", frameIdx+1, config.Pics, formatTimestamp(int(timestamp)))
			}
			img, err := provider.ExtractFrame(info.Filename, timestamp, config.ThumbWidth, config.ThumbHeight)

			if err != nil {
				fmt.Printf("error in frame %v: %s", frameIdx, err)
				frameInfos[frameIdx] = FrameInfo{
					Error: err,
				}
				return
			}

			frameInfos[frameIdx] = FrameInfo{
				Timestamp: formatTimestamp(int(timestamp)),
				Image:     &img,
			}
		}()
	}
	wg.Wait()

	// Check if any errors occurred generating the frames
	for _, frameInfo := range frameInfos {
		if frameInfo.Error != nil {
			return nil, frameInfo.Error
		}
	}

	return frameInfos, nil
}

func drawImage(config Config, info backends.MediaInfo, frameInfos []FrameInfo) error {
	// load font
	if config.Verbose {
		fmt.Printf("composing image\n")
	}
	fontFinder := sysfont.NewFinder(nil)
	font := fontFinder.Match(config.FontFamily)

	height := config.ImagePadding                                      // Initial border
	height += config.FontSize + 4                                      // First row
	height += config.FontSize + 4                                      // second row
	height += config.FontSize + 4                                      // third row
	height += config.Rows * (config.ThumbHeight + config.ImagePadding) // for each row thumb height + padding

	width := config.ImagePadding                        // Padding left
	width += config.ImagePadding                        // Padding right
	width += config.Columns * config.ThumbWidth         // space for all images
	width += (config.Columns - 1) * config.ImagePadding // space for padding between images

	dc := gg.NewContext(width, height)
	err := dc.LoadFontFace(font.Filename, float64(config.FontSize))
	if err != nil {
		return err
	}
	dc.SetHexColor(config.BgColor)
	dc.Clear()
	dc.SetHexColor(config.FontColor)

	dc.Push()
	dc.Translate(float64(config.ImagePadding), float64(config.ImagePadding))
	dc.DrawStringAnchored(fmt.Sprintf("Filename: %s", info.Filename), 0, 0, 0, 1)
	dc.DrawStringAnchored(fmt.Sprintf("Dimensions: %vx%v", info.Width, info.Height), float64(width-config.ImagePadding*2), 0, 1, 1)
	dc.Translate(0, float64(config.FontSize+4))

	dc.DrawStringAnchored(fmt.Sprintf("Duration: %v", formatTimestamp(int(info.Duration))), 0, 0, 0, 1)
	dc.DrawStringAnchored(fmt.Sprintf("Format: %s", info.Format), float64(width-config.ImagePadding*2), 0, 1, 1)
	dc.Translate(0, float64(config.FontSize+4))

	dc.DrawStringAnchored(fmt.Sprintf("File size: %v", humanSize(info.Filesize)), 0, 0, 0, 1)
	dc.DrawStringAnchored(fmt.Sprintf("Fps: %s", info.Framerate), float64(width-config.ImagePadding*2), 0, 1, 1)
	dc.Translate(0, float64(config.FontSize+4))

	dc.Push()

	err = dc.LoadFontFace(font.Filename, 10)
	if err != nil {
		return err
	}

	for i, pic := range frameInfos {
		xOffset := float64((config.ThumbWidth + config.ImagePadding) * (i % config.Columns))
		yOffset := float64((config.ThumbHeight + config.ImagePadding) * (i / config.Columns))
		dc.DrawImage(*pic.Image, int(xOffset), int(yOffset))

		// Draw background for timestamp
		stringWidth, stringHeight := dc.MeasureString(pic.Timestamp)

		dc.Push()
		// Translate into the lower right corner
		dc.Translate(xOffset+float64(config.ThumbWidth), yOffset+float64(config.ThumbHeight))

		// Translate to top left of string with padding
		dc.Translate(-float64(config.ImagePadding)-stringWidth, -float64(config.ImagePadding)-stringHeight)

		dc.SetRGBA(0, 0, 0, .7)
		dc.DrawRectangle(-2, -2, stringWidth+6, stringHeight+6)
		dc.Fill()

		// Draw the timestamp
		dc.SetRGB(1, 1, 1)
		dc.DrawStringAnchored(pic.Timestamp, 0, 0, 0, 1)
		dc.Pop()
	}

	if config.Format == "jpg" {
		err = gg.SaveJPG(fmt.Sprintf("%s.jpg", info.Filename), dc.Image(), config.JpegQuality)
	} else if config.Format == "png" {
		err = gg.SavePNG(fmt.Sprintf("%s.png", info.Filename), dc.Image())
	}

	if err != nil {
		return err
	}
	return nil
}

func humanSize(bytes int) string {
	return datasize.ByteSize.HR(datasize.ByteSize(bytes))
}

func formatTimestamp(seconds int) string {
	result := strings.Builder{}
	minutes := seconds / 60
	seconds = seconds % 60
	if minutes > 60 {
		hours := minutes / 60
		result.WriteString(strconv.Itoa(hours))
		result.WriteString(":")
		minutes = minutes % 60
	}
	result.WriteString(strconv.Itoa(minutes))
	result.WriteString(":")
	result.WriteString(fmt.Sprintf("%02d", seconds))

	return result.String()
}
