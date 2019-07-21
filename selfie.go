package main

import (
	"fmt"
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
		Bounds:      pixel.R(0, 0, 900, 1600),
		Undecorated: true,
		AlwaysOnTop: true,
		VSync:       false,
		Monitor:     pixelgl.Monitors()[0],
	})
	if err != nil {
		panic(err)
	}

	win.SetCursorVisible(false)
	win.Clear(colornames.Greenyellow)
	fmt.Println("AM I DOING IT")

	w := 320
	h := 240
	pic := pixel.MakePictureData(pixel.Rect{pixel.Vec{0, 0}, pixel.Vec{320, 240}})
	sprite := pixel.NewSprite(pic, pic.Bounds())

	for !win.Closed() {
		t := time.Now()
		select {
		case frame := <-framec:
			fmt.Println("UPDATE")
			for i := 0; i < int(w*h); i += 2 {
				y1 := float64(frame[i*2])
				y2 := float64(frame[i*2+2])
				cb := float64(frame[i*2+1])
				cr := float64(frame[i*2+3])
				pic.Pix[(w*h-1)-i].R = uint8(1.164381*y1 + 1.5960195*cr + -222.921612)
				pic.Pix[(w*h-1)-i].G = uint8(1.164381*y1 + -0.3917565*cb + -0.8129655*cr + 135.575391)
				pic.Pix[(w*h-1)-i].B = uint8(1.164381*y1 + 2.0172285*cb + -276.836313)
				pic.Pix[(w*h-1)-i].A = uint8(255)
				pic.Pix[(w*h-1)-(i+1)].R = uint8(1.164381*y2 + 1.5960195*cr + -222.921612)
				pic.Pix[(w*h-1)-(i+1)].G = uint8(1.164381*y2 + -0.3917565*cb + -0.8129655*cr + 135.575391)
				pic.Pix[(w*h-1)-(i+1)].B = uint8(1.164381*y2 + 2.0172285*cb + -276.836313)
				pic.Pix[(w*h-1)-(i+1)].A = uint8(255)
			}
			// sprite.Set(pic, pic.Bounds())
			sprite = pixel.NewSprite(pic, pic.Bounds())
		}
		m := pixel.IM.Moved(win.Bounds().Center()).Scaled(win.Bounds().Center(), 2.8125)
		sprite.Draw(win, m)
		win.Update()
		fmt.Println(time.Since(t))
	}
}

func cam() {
	cam, err := webcam.Open("/dev/video0")
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
