package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/blackjack/webcam"
	"github.com/stianeikeland/go-rpio"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

func main() {
	os.Setenv("DISPLAY", ":0")

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

	font, err := ttf.OpenFont("Raleway-Black.ttf", 600)
	if err != nil {
		log.Fatalf("failed to read font: %v", err)
	}
	texes := make([]*sdl.Texture, 3)
	for i := 0; i < 3; i++ {
		surf, err := font.RenderUTF8Blended(strconv.Itoa(i+1), sdl.Color{R: 255, G: 255, B: 255, A: 255})
		if err != nil {
			log.Fatalf("failed to render text: %v", err)
		}
		texes[i], err = renderer.CreateTextureFromSurface(surf)
		if err != nil {
			log.Fatalf("failed to create texture from surface: %v", err)
		}
		surf.Free()
		texes[i].SetBlendMode(sdl.BLENDMODE_BLEND)
	}

	var buttonPressed time.Time
	buttonPressed = time.Now() // TODO TMP

	for framecount := 0; ; framecount++ {
		t := time.Now()
		if button.EdgeDetected() { // cleeeeeck
			fmt.Println("CLEEEEEECK")
			buttonPressed = time.Now()
		}
		renderer.Clear()
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

		if !buttonPressed.IsZero() {
			if time.Since(buttonPressed) > time.Millisecond*4750 {
				buttonPressed = time.Time{}
				buttonPressed = time.Now() // TODO TMP
				shutter.High()
			} else if time.Since(buttonPressed) > time.Millisecond*4500 {
				renderer.SetDrawColor(255, 255, 255, 255)
				renderer.Clear()
				renderer.SetDrawColor(0, 0, 0, 255)
				shutter.Low()
			} else if time.Since(buttonPressed) > time.Millisecond*3000 {
				focus.Low()
				_, _, texWidth, texHeight, _ := texes[0].Query()
				renderer.Copy(texes[0],
					&sdl.Rect{X: 0, Y: 0, W: texWidth, H: texHeight},
					&sdl.Rect{X: (900 - texWidth) / 2, Y: (1600 - texHeight) / 2, W: texWidth, H: texHeight})
			} else if time.Since(buttonPressed) > time.Millisecond*1500 {
				focus.High()
				_, _, texWidth, texHeight, _ := texes[1].Query()
				renderer.Copy(texes[1],
					&sdl.Rect{X: 0, Y: 0, W: texWidth, H: texHeight},
					&sdl.Rect{X: (900 - texWidth) / 2, Y: (1600 - texHeight) / 2, W: texWidth, H: texHeight})
			} else {
				focus.Low()
				_, _, texWidth, texHeight, _ := texes[2].Query()
				renderer.Copy(texes[2],
					&sdl.Rect{X: 0, Y: 0, W: texWidth, H: texHeight},
					&sdl.Rect{X: (900 - texWidth) / 2, Y: (1600 - texHeight) / 2, W: texWidth, H: texHeight})
			}
		}
		renderer.Present()
		fmt.Println(time.Since(t))
	}
}
