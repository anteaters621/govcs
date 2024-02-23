package backends

import (
	"fmt"
	"github.com/fogleman/gg"
	"image"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type Ffmpeg struct {
	FFmpeg  string
	FFprobe string
}

func (f Ffmpeg) GetInfo(filename string) (MediaInfo, error) {
	result := MediaInfo{
		Filename: filename,
	}

	cmd := exec.Command(f.FFprobe, "-hide_banner", "-select_streams", "v:0", "-show_streams", "-show_format", filename)
	//stderr, err := cmd.StderrPipe()
	//if err != nil {
	//	return result, err
	//}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return result, err
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("error extracting info\n")
		return result, err
	}

	//slurp, _ := io.ReadAll(stderr)
	//fmt.Printf("%s\n", slurp)

	output, _ := io.ReadAll(stdout)
	for _, line := range strings.Split(string(output), "\n") {
		prefix, suffix, hasSuffix := strings.Cut(line, "=")
		if !hasSuffix {
			continue
		}
		if prefix == "codec_long_name" {
			result.Format = suffix

		} else if prefix == "width" {
			result.Width, err = strconv.Atoi(suffix)
			if err != nil {
				fmt.Printf("failed to parse width: %s\n", line)
				return result, err
			}

		} else if prefix == "height" {
			result.Height, err = strconv.Atoi(suffix)
			if err != nil {
				fmt.Printf("failed to parse height: %s\n", line)
				return result, err
			}

		} else if prefix == "avg_frame_rate" {
			// cut of '/1' when the framerate is 60/1 or sth.
			result.Framerate = strings.TrimSuffix(suffix, "/1")

		} else if prefix == "size" {
			result.Filesize, err = strconv.Atoi(suffix)
			if err != nil {
				fmt.Printf("failed to parse size: %s\n", err)
				return result, err
			}

		} else if prefix == "duration" {
			if suffix == "N/A" {
				continue
			}
			result.Duration, err = strconv.ParseFloat(suffix, 64)
			if err != nil {
				fmt.Printf("failed to parse duration: %s\n", err)
				return result, err
			}
		}
	}
	if err := cmd.Wait(); err != nil {
		return result, err
	}

	return result, nil
}

func (f Ffmpeg) ExtractFrame(filename string, timestamp float64, thumbWidth int, thumbHeight int) (image.Image, error) {
	temp, err := os.CreateTemp("", "screenshot-*.png")
	if err != nil {
		fmt.Printf("error extracting frame\n")
		return nil, err
	}
	defer os.Remove(temp.Name())

	// Create still frame with ffmpeg
	cmd := exec.Command(f.FFmpeg,
		"-ss", fmt.Sprintf("%.2f", timestamp),
		"-i", filename,
		"-s", fmt.Sprintf("%vx%v", thumbWidth, thumbHeight),
		"-frames:v", "1",
		"-y",
		temp.Name())
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	// load file into image
	pic, _ := gg.LoadPNG(temp.Name())
	return pic, nil
}
