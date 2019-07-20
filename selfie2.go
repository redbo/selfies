package main

import (
	"fmt"
	"image"
	"log"
	"os"
	"time"

	"github.com/blackjack/webcam"
	"github.com/hajimehoshi/ebiten"
)

var framec = make(chan []byte)
var eimg *ebiten.Image

func update(screen *ebiten.Image) error {
	w := 320
	h := 240
	select {
	case frame := <-framec:
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
		eimg, err := ebiten.NewImageFromImage(img, ebiten.FilterDefault)
		if err != nil {
			return err
		}
		return screen.DrawImage(eimg, &ebiten.DrawImageOptions{})
	}
	return nil
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
			case <-time.After(time.Second / 10):
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

	if err := ebiten.Run(update, 1600, 900, 2, "Your game's title"); err != nil {
		log.Fatal(err)
	}
}
