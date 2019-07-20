package main

import (
	"fmt"
	"image"
	"os"
	"time"

	"github.com/blackjack/webcam"
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"golang.org/x/image/colornames"
)

var framec = make(chan []byte)

func run() {
	win, err := pixelgl.NewWindow(pixelgl.WindowConfig{
		Bounds:      pixel.R(0, 0, 1600, 900),
		Undecorated: true,
		AlwaysOnTop: true,
		VSync:       false,
	})
	if err != nil {
		panic(err)
	}

	win.SetCursorVisible(false)
	win.Clear(colornames.Greenyellow)
	fmt.Println("AM I DOING IT")
	var sprite *pixel.Sprite = nil

	for !win.Closed() {
		select {
		case frame := <-framec:
			w := 320
			h := 240
			img := &image.YCbCr{
				Y:              make([]byte, int(w*h)),
				Cb:             make([]byte, int(w*h/2)),
				Cr:             make([]byte, int(w*h/2)),
				YStride:        int(w),
				CStride:        int(w / 2),
				SubsampleRatio: image.YCbCrSubsampleRatio422,
				Rect:           image.Rectangle{Min: image.Point{0, 0}, Max: image.Point{int(w), int(h)}},
			}
			for i := 0; i < int(w*h); i++ {
				img.Y[i] = frame[i*2]
				if i%2 == 0 {
					img.Cb[i/2] = frame[i*2+1]
				} else {
					img.Cr[i/2] = frame[i*2+1]
				}
			}
			pic := pixel.PictureDataFromImage(img)
			sprite = pixel.NewSprite(pic, pic.Bounds())
		}
		if sprite != nil {
			sprite.Draw(win, pixel.IM.Moved(win.Bounds().Center()))
		}
		fmt.Println("FRAME")
		t := time.Now()
		win.Update()
		fmt.Println(time.Since(t))
	}
}

func cam() {
	cam, err := webcam.Open("/dev/video0") // Open webcam
	if err != nil {
		panic(err.Error())
	}
	defer cam.Close()
	f, w, h, err := cam.SetImageFormat(1448695129, uint32(320), uint32(240))
	fmt.Println(w, h)
	if err != nil {
		panic(err.Error())
	}
	if f != 1448695129 {
		panic("UNSUPPORTED FORMAT")
	}
	err = cam.StartStreaming()
	if err != nil {
		panic(err.Error())
	}
	for {
		err = cam.WaitForFrame(10)

		switch err.(type) {
		case nil:
		case *webcam.Timeout:
			fmt.Fprint(os.Stderr, err.Error())
			continue
		default:
			panic(err.Error())
		}

		frame, err := cam.ReadFrame()
		if err == nil && len(frame) != 0 {
			select {
			case framec <- frame:
			case <-time.After(time.Second / 2):
			}
			// Process frame

			// 	fp, err := os.Create("img.jpg")
			// 	if err != nil {
			// 		panic(err)
			// 	}
			// 	defer fp.Close()
			// 	jpeg.Encode(fp, img, &jpeg.Options{Quality: 95})
			// 	fp.Close()
		} else if err != nil {
			panic(err.Error())
		}
	}
}

func main() {
	go cam()
	pixelgl.Run(run)
}
