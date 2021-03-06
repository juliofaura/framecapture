// Example program that uses blakjack/webcam library
// for working with V4L2 devices.
// The application reads frames from device and writes them to stdout
// If your device supports motion formats (e.g. H264 or MJPEG) you can
// use it's output as a video stream.
// Example usage: go run stdout_streamer.go | vlc -
package main

import (
	"fmt"
	"github.com/blackjack/webcam"
	"image"
	"image/jpeg"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"time"
)

var timelag time.Duration

// const videoCam = "/dev/video0"

func readChoice(s string) int {
	var i int
	for true {
		print(s)
		_, err := fmt.Scanf("%d\n", &i)
		if err != nil || i < 1 {
			println("Invalid input. Try again")
		} else {
			break
		}
	}
	return i
}

type FrameSizes []webcam.FrameSize

func (slice FrameSizes) Len() int {
	return len(slice)
}

//For sorting purposes
func (slice FrameSizes) Less(i, j int) bool {
	ls := slice[i].MaxWidth * slice[i].MaxHeight
	rs := slice[j].MaxWidth * slice[j].MaxHeight
	return ls < rs
}

//For sorting purposes
func (slice FrameSizes) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

func main() {

	files, err := ioutil.ReadDir("/dev")
	if err != nil {
		log.Fatal(err)
	}
	var videoCams []string
	for _, f := range files {
		if len(f.Name()) > 5 && f.Name()[:5] == "video" {
			videoCams = append(videoCams, "/dev/"+f.Name())
		}
	}
	for i, v := range videoCams {
		fmt.Fprintf(os.Stderr, "[%d] %s\n", i+1, v)
	}
	videoCam := videoCams[readChoice(fmt.Sprintf("Choose camera [1-%d]: ", len(videoCams)))-1]

	cam, err := webcam.Open(videoCam)
	if err != nil {
		panic(err.Error())
	}

	var format webcam.PixelFormat
	var format_chosen string
	var subSampleRatio image.YCbCrSubsampleRatio
	var subsampleratio_chosen string

	for fd, f := range cam.GetSupportedFormats() {
		if f[0:4] == "YUYV" {
			format = fd
			format_chosen = f
			subsampleratio_chosen = format_chosen[len(format_chosen)-5:]
			switch subsampleratio_chosen {
			case "4:1:0":
				subSampleRatio = image.YCbCrSubsampleRatio410
			case "4:1:1":
				subSampleRatio = image.YCbCrSubsampleRatio411
			case "4:2:0":
				subSampleRatio = image.YCbCrSubsampleRatio420
			case "4:2:2":
				subSampleRatio = image.YCbCrSubsampleRatio422
			case "4:4:0":
				subSampleRatio = image.YCbCrSubsampleRatio440
			case "4:4:4":
				subSampleRatio = image.YCbCrSubsampleRatio444
			default:
				panic("Unknown subsample ratio: " + subsampleratio_chosen)
			}
			break
		}
	}
	if format_chosen == "" {
		panic("No YUYV format found")
	}

	fmt.Fprintf(os.Stderr, "Supported frame sizes for format %s\n", format_chosen)
	frames := FrameSizes(cam.GetSupportedFrameSizes(format))
	sort.Sort(frames)

	for i, value := range frames {
		fmt.Fprintf(os.Stderr, "[%d] %s\n", i+1, value.GetString())
	}
	choice := readChoice(fmt.Sprintf("Choose format [1-%d]: ", len(frames)))
	size := frames[choice-1]

	f, w, h, err := cam.SetImageFormat(format, uint32(size.MaxWidth), uint32(size.MaxHeight))

	if err != nil {
		panic(err.Error())
	} else {
		fmt.Fprintf(os.Stderr, "Resulting image format: %v (%s) (%dx%d)\n", f, format_chosen, w, h)
	}

	println("time lag (in seconds):")
	timelag = time.Duration(readChoice("")) * time.Second

	println("Press Enter to start capturing frames")
	fmt.Scanf("\n")
	cam.Close()

	timeout := uint32(5) //5 seconds
	frameNr := 0
	initial_time := time.Now()

	for {
		cam, err := webcam.Open(videoCam)
		if err != nil {
			panic(err.Error())
		}
		_, _, _, err = cam.SetImageFormat(format, uint32(size.MaxWidth), uint32(size.MaxHeight))
		err = cam.StartStreaming()
		if err != nil {
			panic(err.Error())
		}
		fmt.Println("Streaming started, waiting to stabilize ...")
		time.Sleep(time.Second)
		err = cam.WaitForFrame(timeout)

		switch err.(type) {
		case nil:
		case *webcam.Timeout:
			fmt.Fprint(os.Stderr, err.Error())
			continue
		default:
			panic(err.Error())
		}
		fmt.Println("Waited for frame, all good, now to read frame ...")

		frame, err := cam.ReadFrame()
		if err != nil || len(frame) == 0 {
			log.Panic("Error reading frame after WaitForFrame")
		}
		fmt.Println("Frame read")
		for {
			lastframe := frame
			frame, err = cam.ReadFrame()
			if err != nil || len(frame) == 0 {
				fmt.Println("Read new frame, len is", len(frame), "and err is", err)
				frame = lastframe
				break
			}
		}
		//fmt.Printf("Received frame, format is %v, width is %v, heigth is %v\n", format_chosen, size.MaxWidth, size.MaxHeight)
		log.Println("Photo taken, frameNr is", frameNr)
		now := time.Now()
		file, err := os.Create(fmt.Sprintf("Frame-%08d--%04d-%02d-%02d-%02d-%02d-%02d-%02d.jpg",
			frameNr, now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond()))
		if err != nil {
			panic(err.Error())
		}
		if format_chosen[0:4] == "YUYV" {
			YUVimage := image.NewYCbCr(image.Rect(0, 0, int(size.MaxWidth), int(size.MaxHeight)), subSampleRatio)
			fmt.Println("Now iterating", len(YUVimage.Cb), "elements, subsample ratio is", subSampleRatio)
			fmt.Printf("Sizes are: %v (Y), %v (Cb), %v (Cr), and %v (frame)\n",
				len(YUVimage.Y), len(YUVimage.Cb), len(YUVimage.Cr), len(frame))
			for i := range YUVimage.Cb {
				YUVimage.Y[2*i] = frame[4*i]
				YUVimage.Y[2*i+1] = frame[4*i+2]
				YUVimage.Cb[i] = frame[4*i+1]
				YUVimage.Cr[i] = frame[4*i+3]
			}
			fmt.Printf("Offsets of 0,0 are %v (Y) and %v (C), and strides are %v (Y) and %v (C)\n",
				YUVimage.YOffset(0, 0), YUVimage.YStride, YUVimage.COffset(0, 0), YUVimage.CStride)
			jpeg.Encode(file, YUVimage, &jpeg.Options{Quality: 100})
		} else {
			_, err = file.Write(frame)
			if err != nil {
				panic(err.Error())
			}
		}
		err = file.Close()
		if err != nil {
			panic(err.Error())
		}

		cam.Close()
		fmt.Println("Camera closed\n")
		frameNr++
		for time.Since(initial_time) < timelag*time.Duration(frameNr) {
			time.Sleep(100 * time.Millisecond)
		}
	}
}
