package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/blackjack/webcam"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

func main() {
	os.Setenv("DISPLAY", ":0")

	/*
		if err := rpio.Open(); err != nil {
			log.Fatalf("error opening rpio: %v", err)
		}
		defer rpio.Close()
		button := rpio.Pin(14)
		button.Input()
		button.PullUp()
		button.Detect(rpio.FallEdge)
		focus := rpio.Pin(3)
		focus.Output()
		focus.High()
		shutter := rpio.Pin(5)
		shutter.Output()
		shutter.High()
	*/

	err := sdl.Init(sdl.INIT_EVERYTHING)
	if err != nil {
		log.Fatalf("failed to initialize sdl: %v", err)
	}
	defer sdl.Quit()
	sdl.DisableScreenSaver()

	err = ttf.Init()
	if err != nil {
		log.Fatalf("failed to initialize ttf: %v", err)
	}
	defer ttf.Quit()

	window, err := sdl.CreateWindow("SELFIES", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		900, 1600, sdl.WINDOW_SHOWN|sdl.WINDOW_FULLSCREEN|sdl.WINDOW_BORDERLESS)
	if err != nil {
		log.Fatalf("failed to create renderer: %v", err)
	}
	defer window.Destroy()

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		log.Fatalf("error creating renderer: %v", err)
	}
	defer renderer.Destroy()
	renderer.Clear()

	cam, err := webcam.Open("/dev/video0")
	if err != nil {
		log.Fatalf("failed to open video device: %v", err)
	}
	defer cam.Close()

	f, cw, ch, err := cam.SetImageFormat(1448695129, uint32(320), uint32(240))
	fmt.Println(cw, ch)
	if err != nil {
		log.Fatalf("failed to set video format: %v", err)
	}
	if f != 1448695129 {
		panic("UNSUPPORTED CAM FORMAT")
	}
	if err = cam.StartStreaming(); err != nil {
		log.Fatalf("failed to begin video streaming: %v", err)
	}
	cam.SetBufferCount(1)

	tex, err := renderer.CreateTexture(sdl.PIXELFORMAT_YUY2, sdl.TEXTUREACCESS_STREAMING, int32(cw), int32(ch))
	if err != nil {
		log.Fatalf("error creating texture: %v", err)
	}
	defer tex.Destroy()
	snaps := make([]*sdl.Texture, 4)
	for i := range snaps {
		snaps[i], err = renderer.CreateTexture(sdl.PIXELFORMAT_YUY2, sdl.TEXTUREACCESS_STREAMING, int32(cw), int32(ch))
		if err != nil {
			log.Fatalf("error creating texture: %v", err)
		}
		defer snaps[i].Destroy()
	}

	font, err := ttf.OpenFont("Raleway-Black.ttf", 800)
	if err != nil {
		log.Fatalf("failed to read font: %v", err)
	}
	surf3, err := font.RenderUTF8Blended("3", sdl.Color{R: 255, G: 255, B: 255, A: 255})
	if err != nil {
		log.Fatalf("failed to render text: %v", err)
	}
	tex3, err := renderer.CreateTextureFromSurface(surf3)
	if err != nil {
		log.Fatalf("failed to create texture from surface: %v", err)
	}
	tex3.SetBlendMode(sdl.BLENDMODE_BLEND)
	_, _, texWidth, texHeight, _ := tex3.Query()

	for framecount := 0; ; framecount++ {
		t := time.Now()
		renderer.Clear()
		/*
			if button.EdgeDetected() { // cleeeeeck
				fmt.Println("CLEEEEEECK")
			}
		*/
		for {
			if frame, _ := cam.ReadFrame(); frame != nil && len(frame) != 0 {
				if framecount%50 == 0 {
					x := snaps[3]
					copy(snaps[1:4], snaps[0:3])
					snaps[0] = x
					snaps[0].Update(&sdl.Rect{X: 0, Y: 0, W: int32(cw), H: int32(ch)}, frame, 2*int(cw))
				}
				tex.Update(&sdl.Rect{X: 0, Y: 0, W: int32(cw), H: int32(ch)}, frame, 2*int(cw))
			} else {
				break
			}
		}
		renderer.Copy(tex, &sdl.Rect{X: 1, Y: 14, W: 318, H: 212}, &sdl.Rect{X: 0, Y: 0, W: 900, H: 600})
		renderer.Copy(snaps[0], &sdl.Rect{X: 1, Y: 14, W: 318, H: 212}, &sdl.Rect{X: 0, Y: 800, W: 430, H: 287})
		renderer.Copy(snaps[1], &sdl.Rect{X: 1, Y: 14, W: 318, H: 212}, &sdl.Rect{X: 470, Y: 800, W: 430, H: 287})
		renderer.Copy(snaps[2], &sdl.Rect{X: 1, Y: 14, W: 318, H: 212}, &sdl.Rect{X: 0, Y: 1237, W: 430, H: 287})
		renderer.Copy(snaps[3], &sdl.Rect{X: 1, Y: 14, W: 318, H: 212}, &sdl.Rect{X: 470, Y: 1237, W: 430, H: 287})
		renderer.Copy(tex3, &sdl.Rect{X: 0, Y: 0, W: texWidth, H: texHeight}, &sdl.Rect{X: (900 - texWidth) / 2, Y: (1600 - texHeight) / 2, W: texWidth, H: texHeight})

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
