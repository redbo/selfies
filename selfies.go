package selfies

import (
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"time"

	"github.com/blackjack/webcam"
	"github.com/jacobsa/go-serial/serial"
	"github.com/nfnt/resize"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

func Selfies() {
	os.Setenv("DISPLAY", ":0")

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
	sdl.WarpMouseGlobal(900, 1600)
	sdl.ShowCursor(sdl.DISABLE)

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		log.Fatalf("error creating renderer: %v", err)
	}
	defer renderer.Destroy()
	renderer.Clear()

	initCam := func(width, height uint32) (*webcam.Webcam, error) {
		cam, err := webcam.Open("/dev/video0")
		if err != nil {
			return nil, err
		}
		if f, cw, ch, err := cam.SetImageFormat(1448695129, width, height); err != nil {
			cam.Close()
			return nil, err
		} else if f != 1448695129 || cw != width || ch != height {
			cam.Close()
			return nil, fmt.Errorf("Unknown pixel format %d (%d/%d)", f, cw, ch)
		}
		if err = cam.SetBufferCount(1); err != nil {
			cam.Close()
			return nil, err
		}
		return cam, cam.StartStreaming()
	}

	cam, err := initCam(1280, 720)
	if err != nil {
		log.Fatalf("failed to initialize webcam: %v", err)
	}
	defer cam.Close()

	tex, err := renderer.CreateTexture(sdl.PIXELFORMAT_YUY2, sdl.TEXTUREACCESS_STREAMING, int32(1280), int32(720))
	if err != nil {
		log.Fatalf("error creating texture: %v", err)
	}
	defer tex.Destroy()
	snaps := make([]*sdl.Texture, 4)
	snapfiles := make([]string, 4)
	for i := range snaps {
		if snaps[i], err = renderer.CreateTexture(sdl.PIXELFORMAT_ABGR8888, sdl.TEXTUREACCESS_STREAMING, 430, 287); err != nil {
			log.Fatalf("error creating texture: %v", err)
		}
		defer snaps[i].Destroy()
	}
	usr, err := user.Current()
	if err != nil {
		log.Fatalf("failed to get home dir: %v", err)
	}
	font, err := makeFont(600)
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

	font, err = makeFont(30)
	if err != nil {
		log.Fatalf("failed to read font: %v", err)
	}
	surf, err := font.RenderUTF8Blended("Print", sdl.Color{R: 255, G: 255, B: 0, A: 255})
	if err != nil {
		log.Fatalf("failed to render text: %v", err)
	}
	printtexWidth := surf.W
	printtexHeight := surf.H
	printtex, err := renderer.CreateTextureFromSurface(surf)
	if err != nil {
		log.Fatalf("failed to create texture from surface: %v", err)
	}
	surf.Free()
	printtex.SetBlendMode(sdl.BLENDMODE_BLEND)

	surf, err = font.RenderUTF8Blended("Printing", sdl.Color{R: 255, G: 0, B: 0, A: 255})
	if err != nil {
		log.Fatalf("failed to render text: %v", err)
	}
	printingtexWidth := surf.W
	printingtexHeight := surf.H
	printingtex, err := renderer.CreateTextureFromSurface(surf)
	if err != nil {
		log.Fatalf("failed to create texture from surface: %v", err)
	}
	surf.Free()
	printingtex.SetBlendMode(sdl.BLENDMODE_BLEND)

	arduino, err := serial.Open(serial.OpenOptions{
		PortName:        "/dev/ttyUSB0",
		BaudRate:        9600,
		DataBits:        8,
		StopBits:        1,
		MinimumReadSize: 1,
	})
	if err != nil {
		log.Fatalf("serial.Open: %v", err)
	}
	arduino.Write([]byte{'R', '\r', '\n'})
	var buttonPress = make(chan byte)
	defer arduino.Close()
	go func() {
		start := time.Now()
		b := make([]byte, 1)
		for true {
			arduino.Read(b)
			if b[0] > ' ' && time.Since(start) > 10*time.Second {
				buttonPress <- b[0]
			}
		}
	}()

	buttonPressed := time.Time{}
	printCooldown := time.Time{}
	var lights, focus bool
	var frame []byte
	var printnotify = make(chan bool)
	var printing bool

	for framecount := 0; ; framecount++ {
		select {
		case button := <-buttonPress:
			if button == '2' {
				arduino.Write([]byte{'R', '\r', '\n'})
				buttonPressed = time.Now()
			} else if button == '3' && snapfiles[0] != "" && time.Since(printCooldown) > time.Second*30 {
				printCooldown = time.Now()
				go func() {
					printnotify <- true
					exec.Command("/usr/bin/obexftp", "--nopath", "--noconn", "--uuid", "none",
						"--bluetooth", "C4:30:18:19:C6:3D", "--channel", "4", "-p", snapfiles[0]).Run()
					printnotify <- false
				}()
			}
		case printing = <-printnotify:
		default:
		}
		renderer.Clear()
		for {
			if f, _ := cam.ReadFrame(); f != nil && len(f) != 0 {
				frame = f
				tex.Update(&sdl.Rect{X: 0, Y: 0, W: int32(1280), H: int32(720)}, frame, 2*int(1280))
			} else {
				break
			}
		}
		if snapfiles[0] == "" {
		} else if printing {
			renderer.SetDrawColor(uint8(rand.Int()%255), uint8(rand.Int()%255), uint8(rand.Int()%255), 255)
			renderer.FillRect(&sdl.Rect{X: 0, Y: 800, W: 430, H: 287})
			renderer.Copy(printingtex,
				&sdl.Rect{X: 0, Y: 0, W: printingtexWidth, H: printingtexHeight},
				&sdl.Rect{X: (430 - printingtexWidth) / 2, Y: (800 - printingtexHeight), W: printingtexWidth, H: printingtexHeight})
			renderer.Copy(snaps[0], &sdl.Rect{X: 2, Y: 2, W: 430, H: 287}, &sdl.Rect{X: 10, Y: 810, W: 410, H: 267})
		} else {
			renderer.SetDrawColor(255, 255, 0, 255)
			renderer.FillRect(&sdl.Rect{X: 0, Y: 800, W: 430, H: 287})
			renderer.Copy(printtex,
				&sdl.Rect{X: 0, Y: 0, W: printtexWidth, H: printtexHeight},
				&sdl.Rect{X: (430 - printtexWidth) / 2, Y: (800 - printtexHeight), W: printtexWidth, H: printtexHeight})
			renderer.Copy(snaps[0], &sdl.Rect{X: 2, Y: 2, W: 430, H: 287}, &sdl.Rect{X: 2, Y: 802, W: 426, H: 283})
		}
		renderer.SetDrawColor(0, 0, 0, 255)
		renderer.Copy(tex, &sdl.Rect{X: 0, Y: 0, W: 1280, H: 720}, &sdl.Rect{X: -90, Y: 0, W: 1080, H: 600})
		renderer.Copy(snaps[1], &sdl.Rect{X: 0, Y: 0, W: 430, H: 287}, &sdl.Rect{X: 470, Y: 800, W: 430, H: 287})
		renderer.Copy(snaps[2], &sdl.Rect{X: 0, Y: 0, W: 430, H: 287}, &sdl.Rect{X: 0, Y: 1237, W: 430, H: 287})
		renderer.Copy(snaps[3], &sdl.Rect{X: 0, Y: 0, W: 430, H: 287}, &sdl.Rect{X: 470, Y: 1237, W: 430, H: 287})

		if !buttonPressed.IsZero() {
			if !focus && time.Since(buttonPressed) > time.Millisecond*4000 {
				focus = true
				arduino.Write([]byte{'A', '\r', '\n'})
			}
			if !lights && time.Since(buttonPressed) > time.Millisecond*3500 {
				lights = true
				arduino.Write([]byte{'B', '\r', '\n'})
			}

			if time.Since(buttonPressed) > time.Millisecond*4500 {
				if frame != nil && len(frame) != 0 {
					// flash a white screen
					renderer.SetDrawColor(255, 255, 255, 255)
					renderer.Clear()
					renderer.Present()
					renderer.SetDrawColor(0, 0, 0, 255)
					img := image.NewYCbCr(image.Rect(0, 0, 1280, 720), image.YCbCrSubsampleRatio422)
					for i := 0; i < int(1280*720); i++ {
						img.Y[i] = frame[i*2]
						if i%2 == 0 {
							img.Cb[i/2] = frame[i*2+1]
						} else {
							img.Cr[i/2] = frame[i*2+1]
						}
					}
					cropped := image.NewRGBA(image.Rect(0, 0, 1080, 720))
					draw.Draw(cropped, cropped.Bounds(), img, image.Point{80, 0}, draw.Over)
					filename := filepath.Join(usr.HomeDir, "selfies", "snaps", fmt.Sprintf("%d.jpg", time.Now().Unix()))
					if fp, err := os.Create(filename); err == nil {
						jpeg.Encode(fp, cropped, &jpeg.Options{Quality: 90})
						fp.Close()
					}
					snap := image.NewRGBA(image.Rect(0, 0, 430, 287))
					draw.Draw(snap, snap.Bounds(), resize.Resize(430, 287, cropped, resize.Bicubic), image.ZP, draw.Over)
					snaps[0], snaps[1], snaps[2], snaps[3] = snaps[3], snaps[0], snaps[1], snaps[2]
					snaps[0].Update(&sdl.Rect{X: 0, Y: 0, W: 430, H: 287}, snap.Pix, snap.Stride)
					snapfiles[0], snapfiles[1], snapfiles[2], snapfiles[3] = filename, snapfiles[0], snapfiles[1], snapfiles[2]
				} else {
					fmt.Println("BAD FRAME")
				}
				arduino.Write([]byte{'C', '\r', '\n'})
				time.Sleep(time.Millisecond * 200)
				arduino.Write([]byte{'R', '\r', '\n'})
				lights, focus, buttonPressed = false, false, time.Time{}
			} else if time.Since(buttonPressed) > time.Millisecond*3000 {
				_, _, texWidth, texHeight, _ := texes[0].Query()
				renderer.Copy(texes[0],
					&sdl.Rect{X: 0, Y: 0, W: texWidth, H: texHeight},
					&sdl.Rect{X: (900 - texWidth) / 2, Y: (1600 - texHeight) / 2, W: texWidth, H: texHeight})
			} else if time.Since(buttonPressed) > time.Millisecond*1500 {
				_, _, texWidth, texHeight, _ := texes[1].Query()
				renderer.Copy(texes[1],
					&sdl.Rect{X: 0, Y: 0, W: texWidth, H: texHeight},
					&sdl.Rect{X: (900 - texWidth) / 2, Y: (1600 - texHeight) / 2, W: texWidth, H: texHeight})
			} else {
				_, _, texWidth, texHeight, _ := texes[2].Query()
				renderer.Copy(texes[2],
					&sdl.Rect{X: 0, Y: 0, W: texWidth, H: texHeight},
					&sdl.Rect{X: (900 - texWidth) / 2, Y: (1600 - texHeight) / 2, W: texWidth, H: texHeight})
			}
		}
		renderer.Present()
	}
}
