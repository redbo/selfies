package main

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/blackjack/webcam"
	"github.com/nfnt/resize"
	"github.com/stianeikeland/go-rpio"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

var re = regexp.MustCompile(`<td align="center">(\w+.JPG)</td></tr>`)

var ic = make(chan *image.RGBA)

func fetchCamera() {
	client := http.Client{Transport: &http.Transport{Dial: &net.Dialer{Timeout: 2 * time.Second, KeepAlive: time.Second}.Dial}}

	for {
		time.Sleep(time.Second * 2)
		resp, err := client.Get("http://192.168.4.1/photo")
		if err != nil {
			continue
		}
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			continue
		}
		resp.Body.Close()
		fmt.Println("Got index body")

		for _, res := range re.FindAllSubmatch(data, len(data)) {
			filename := string(res[1])
			if _, err := os.Stat(filepath.Join("snaps", filename)); err == nil {
				continue
			}
			fmt.Println("Downloading frame")
			resp, err := client.Get("http://192.168.4.1/download?fname=" + filename + "&fdir=100OLYMP&folderFlag=0")
			if err != nil {
				continue
			}
			data, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				continue
			}
			resp.Body.Close()
			img, err := jpeg.Decode(bytes.NewBuffer(data))
			if err != nil {
				continue
			}
			rgbimg := image.NewRGBA(image.Rect(0, 0, 300, 200))
			draw.Draw(rgbimg, rgbimg.Bounds(), resize.Resize(300, 200, img, resize.Bilinear), image.ZP, draw.Over)
			fmt.Println("Sending frame down")
			ic <- rgbimg
			ioutil.WriteFile(filepath.Join("snaps", filename), data, os.ModePerm)
		}
	}
}

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

	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		log.Fatalf("failed to initialize sdl: %v", err)
	}
	defer sdl.Quit()
	sdl.DisableScreenSaver()

	if err := ttf.Init(); err != nil {
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
		if snaps[i], err = renderer.CreateTexture(sdl.PIXELFORMAT_RGBA8888, sdl.TEXTUREACCESS_STREAMING, 300, 200); err != nil {
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
		if texes[i], err = renderer.CreateTextureFromSurface(surf); err != nil {
			log.Fatalf("failed to create texture from surface: %v", err)
		}
		surf.Free()
		texes[i].SetBlendMode(sdl.BLENDMODE_BLEND)
	}

	var buttonPressed time.Time
	buttonPressed = time.Now() // TODO TMP

	go fetchCamera()

	fmt.Println("STARTING FRAMES")
	for framecount := 0; ; framecount++ {
		// t := time.Now()
		if button.EdgeDetected() { // cleeeeeck
			fmt.Println("CLEEEEEECK")
			buttonPressed = time.Now()
		}
		renderer.Clear()
		for {
			if frame, _ := cam.ReadFrame(); frame != nil && len(frame) != 0 {
				if framecount%50 == 0 {
				}
				tex.Update(&sdl.Rect{X: 0, Y: 0, W: int32(cw), H: int32(ch)}, frame, 2*int(cw))
			} else {
				break
			}
		}
		select {
		case snap := <-ic:
			snaps[0], snaps[1], snaps[2], snaps[3] = snaps[3], snaps[0], snaps[1], snaps[2]
			snaps[0].Update(&sdl.Rect{X: 0, Y: 0, W: int32(cw), H: int32(ch)}, snap.Pix, snap.Stride)
		default:
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
		//fmt.Println(time.Since(t))
	}
}
