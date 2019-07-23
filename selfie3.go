package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/blackjack/webcam"
	"github.com/veandco/go-sdl2/sdl"
)

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

	cam, err := webcam.Open("/dev/video0")
	if err != nil {
		panic(err.Error())
	}
	defer cam.Close()

	f, cw, ch, err := cam.SetImageFormat(1448695129, uint32(320), uint32(240))
	fmt.Println(cw, ch)
	if err != nil {
		panic(err.Error())
	}
	if f != 1448695129 {
		panic("UNSUPPORTED CAM FORMAT")
	}
	if err = cam.StartStreaming(); err != nil {
		panic(err.Error())
	}
	cam.SetBufferCount(1)

	tex, err := renderer.CreateTexture(sdl.PIXELFORMAT_YUY2, sdl.TEXTUREACCESS_STREAMING, int32(cw), int32(ch))
	if err != nil {
		log.Fatalf("error creating texture: %v", err)
	}
	defer tex.Destroy()

	for {
		t := time.Now()
		pixels, _, err := tex.Lock(&sdl.Rect{X: 0, Y: 0, W: int32(cw), H: int32(ch)})
		for {
			if frame, frameIndex, _ := cam.GetFrame(); frame != nil && len(frame) != 0 {
				if err != nil {
					log.Fatalf("error locking texture: %v", err)
				}
				copy(pixels, frame)
				cam.ReleaseFrame(frameIndex)
			} else {
				break
			}
		}
		tex.Unlock()
		renderer.Copy(tex, &sdl.Rect{X: 0, Y: 0, W: int32(cw), H: int32(ch)}, &sdl.Rect{X: 0, Y: 0, W: 900, H: 675})
		renderer.Present()
		fmt.Println(time.Since(t))
	}
}

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
