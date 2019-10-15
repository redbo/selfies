package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/blackjack/webcam"
	"github.com/jacobsa/go-serial/serial"
	"github.com/nfnt/resize"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

var re = regexp.MustCompile(`<td align="center">(\w+.JPG)</td></tr>`)

func main() {
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

	cam, err := initCam(320, 240)
	if err != nil {
		log.Fatalf("failed to initialize webcam: %v", err)
	}
	defer cam.Close()

	tex, err := renderer.CreateTexture(sdl.PIXELFORMAT_YUY2, sdl.TEXTUREACCESS_STREAMING, int32(320), int32(240))
	if err != nil {
		log.Fatalf("error creating texture: %v", err)
	}
	defer tex.Destroy()
	snaps := make([]*sdl.Texture, 4)
	snapfiles := make([]string, 4)
	for i := range snaps {
		if snaps[i], err = renderer.CreateTexture(sdl.PIXELFORMAT_ABGR8888, sdl.TEXTUREACCESS_STREAMING, 300, 200); err != nil {
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
	var buttonPress = make(chan byte)
	defer arduino.Close()
	go func() {
		b := make([]byte, 1)
		for true {
			arduino.Read(b)
			if b[0] > ' ' {
				buttonPress <- b[0]
			}
		}
	}()

	buttonPressed := time.Time{}
	// buttonPressed = time.Now() // TESTY TEST

	for framecount := 0; ; framecount++ {
		select {
		case button := <-buttonPress:
			if button == '2' {
				buttonPressed = time.Now()
			} else if button == '2' && snapfiles[0] != "" {
				exec.Command("/usr/bin/obexftp", "--nopath", "--noconn", "--uuid", "none",
					"--bluetooth", "C4:30:18:19:C6:3D", "--channel", "4", "-p", snapfiles[0]).Run()
			}
		}
		renderer.Clear()
		for {
			if frame, _ := cam.ReadFrame(); frame != nil && len(frame) != 0 {
				tex.Update(&sdl.Rect{X: 0, Y: 0, W: int32(320), H: int32(240)}, frame, 2*int(320))
			} else {
				break
			}
		}
		renderer.Copy(tex, &sdl.Rect{X: 1, Y: 14, W: 318, H: 212}, &sdl.Rect{X: 0, Y: 0, W: 900, H: 600})
		renderer.Copy(snaps[0], &sdl.Rect{X: 0, Y: 0, W: 300, H: 200}, &sdl.Rect{X: 0, Y: 800, W: 430, H: 287})
		renderer.Copy(snaps[1], &sdl.Rect{X: 0, Y: 0, W: 300, H: 200}, &sdl.Rect{X: 470, Y: 800, W: 430, H: 287})
		renderer.Copy(snaps[2], &sdl.Rect{X: 0, Y: 0, W: 300, H: 200}, &sdl.Rect{X: 0, Y: 1237, W: 430, H: 287})
		renderer.Copy(snaps[3], &sdl.Rect{X: 0, Y: 0, W: 300, H: 200}, &sdl.Rect{X: 470, Y: 1237, W: 430, H: 287})

		if !buttonPressed.IsZero() {
			if time.Since(buttonPressed) > time.Millisecond*5000 {
				// unpress button
				buttonPressed = time.Time{}
				arduino.Write([]byte{'R', '\r', '\n'})
			} else if time.Since(buttonPressed) > time.Millisecond*4500 {
				// flash a white screen
				renderer.SetDrawColor(255, 255, 255, 255)
				renderer.Clear()
				renderer.Present()
				renderer.SetDrawColor(0, 0, 0, 255)
				// re-initialize webcam for high-res shot
				cam.StopStreaming()
				cam.Close()
				if cam, err = initCam(2304, 1536); err != nil {
					log.Fatalf("failed to re-initialize webcam: %v", err)
				}
				cam.WaitForFrame(10)
				arduino.Write([]byte{'B', '\r', '\n'})
				if frame, _ := cam.ReadFrame(); frame != nil && len(frame) != 0 {
					img := &image.YCbCr{
						Y:              make([]byte, int(2304*1536)),
						Cb:             make([]byte, int((2304*1536)/2)),
						Cr:             make([]byte, int((2304*1536)/2)),
						YStride:        int(2304),
						CStride:        int(2304 / 2),
						SubsampleRatio: image.YCbCrSubsampleRatio422,
						Rect:           image.Rectangle{Min: image.Point{0, 0}, Max: image.Point{int(2304), int(1536)}},
					}
					for i := 0; i < int(2304*1536); i++ {
						img.Y[i] = frame[i*2]
						if i%2 == 0 {
							img.Cb[i/2] = frame[i*2+1]
						} else {
							img.Cr[i/2] = frame[i*2+1]
						}
					}
					filename := filepath.Join("snaps", fmt.Sprintf("%d.jpg", time.Now().Unix()))
					if fp, err := os.Create(filename); err == nil {
						jpeg.Encode(fp, img, &jpeg.Options{Quality: 90})
						fp.Close()
					}
					// we want to get a 300x200 RGBA snap to display
					snap := image.NewRGBA(image.Rect(0, 0, 300, 200))
					draw.Draw(snap, snap.Bounds(), resize.Resize(300, 200, img, resize.Bicubic), image.ZP, draw.Over)
					snaps[0], snaps[1], snaps[2], snaps[3] = snaps[3], snaps[0], snaps[1], snaps[2]
					snaps[0].Update(&sdl.Rect{X: 0, Y: 0, W: 300, H: 200}, snap.Pix, snap.Stride)
					snapfiles[0], snapfiles[1], snapfiles[2], snapfiles[3] = filename, snapfiles[0], snapfiles[1], snapfiles[2]
				}
				// downgrade webcam to streaming size
				cam.StopStreaming()
				cam.Close()
				if cam, err = initCam(320, 240); err != nil {
					log.Fatalf("failed to re-re-initialize webcam: %v", err)
				}
			} else if time.Since(buttonPressed) > time.Millisecond*3000 {
				arduino.Write([]byte{'A', '\r', '\n'})
				_, _, texWidth, texHeight, _ := texes[0].Query()
				renderer.Copy(texes[0],
					&sdl.Rect{X: 0, Y: 0, W: texWidth, H: texHeight},
					&sdl.Rect{X: (900 - texWidth) / 2, Y: (1600 - texHeight) / 2, W: texWidth, H: texHeight})
			} else if time.Since(buttonPressed) > time.Millisecond*1500 {
				arduino.Write([]byte{'C', '\r', '\n'})
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
