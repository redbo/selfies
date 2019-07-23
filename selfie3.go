package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/blackjack/webcam"
	"github.com/veandco/go-sdl2/sdl"
)

var framec = make(chan []byte)

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
	cam.SetBufferCount(1)
	err = cam.StartStreaming()
	if err != nil {
		panic(err.Error())
	}
	for {
		frame, err := cam.ReadFrame()
		if frame != nil && len(frame) != 0 {
			/*
				wait := time.NewTimer(time.Second / 4)
				select {
				case framec <- frame:
				case <-wait.C:
				}
				wait.Stop()
			*/

			/*
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
				fp, err := os.Create("img.jpg")
				if err != nil {
					panic(err)
				}
				defer fp.Close()
				jpeg.Encode(fp, img, &jpeg.Options{Quality: 95})
				fp.Close()
			*/
		}
	}
}

func main() {
	os.Setenv("DISPLAY", ":0")

	// try to initialize everything
	err := sdl.Init(sdl.INIT_EVERYTHING)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize sdl: %s\n", err)
		os.Exit(1)
	}
	defer sdl.Quit()

	// try to create a window
	window, err := sdl.CreateWindow("SELFIES", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		900, 1600, sdl.WINDOW_SHOWN|sdl.WINDOW_FULLSCREEN|sdl.WINDOW_BORDERLESS)
	if err != nil {
		fmt.Fprint(os.Stderr, "Failed to create renderer: %s\n", err)
		os.Exit(2)
	}
	defer window.Destroy()
	sdl.DisableScreenSaver()

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		log.Fatalf("error creating renderer: %v", err)
	}
	defer renderer.Destroy()
	renderer.Clear()

	tex, err := renderer.CreateTexture(sdl.PIXELFORMAT_YUY2, sdl.TEXTUREACCESS_STREAMING, 320, 240)
	if err != nil {
		log.Fatalf("error creating texture: %v", err)
	}
	defer tex.Destroy()

	go cam()

	for {
		t := time.Now()
		select {
		case frame := <-framec:
			pixels, _, err := tex.Lock(&sdl.Rect{X: 0, Y: 0, W: 320, H: 240})
			if err != nil {
				log.Fatalf("error locking texture: %v", err)
			}
			copy(pixels, frame)
			tex.Unlock()
		}
		renderer.Copy(tex, &sdl.Rect{X: 0, Y: 0, W: 320, H: 240}, &sdl.Rect{X: 0, Y: 0, W: 900, H: 675})
		renderer.Present()
		fmt.Println(time.Since(t))
	}
}
