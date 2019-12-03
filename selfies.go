package selfies

import (
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"io"
	"math"
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
)

type Selfies struct {
	screenWidth  int32
	screenHeight int32
	renderer     *sdl.Renderer
	cam          *webcam.Webcam
	tex          *sdl.Texture
	texes        []*sdl.Texture
	printtex     *sdl.Texture
	printingtex  *sdl.Texture
	arduino      io.ReadWriteCloser
	snaps        []*sdl.Texture
	snapfiles    []string
	savepath     string

	cleanups []func() error
}

var capFormat = webcam.PixelFormat(1448695129) // V4L2_PIX_FMT_YUYV - YUV 4:2:2
var capWidth int32 = 1280
var capHeight int32 = 720

func initCam(width, height int32) (*webcam.Webcam, error) {
	cam, err := webcam.Open("/dev/video0")
	if err != nil {
		return nil, err
	}
	if f, cw, ch, err := cam.SetImageFormat(capFormat, uint32(width), uint32(height)); err != nil {
		cam.Close()
		return nil, err
	} else if f != capFormat || cw != uint32(width) || ch != uint32(height) {
		cam.Close()
		return nil, fmt.Errorf("Unknown pixel format %d (%d/%d)", f, cw, ch)
	}
	if err = cam.SetBufferCount(1); err != nil {
		cam.Close()
		return nil, err
	}
	return cam, cam.StartStreaming()
}

func NewSelfies() (*Selfies, error) {
	s := &Selfies{
		screenWidth:  900,
		screenHeight: 1600,
	}
	window, err := sdl.CreateWindow("SELFIES", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		s.screenWidth, s.screenHeight, sdl.WINDOW_SHOWN|sdl.WINDOW_FULLSCREEN|sdl.WINDOW_BORDERLESS)
	if err != nil {
		s.Close()
		return nil, fmt.Errorf("failed to create renderer: %v", err)
	}
	s.cleanup(window.Destroy)

	if s.renderer, err = sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED); err != nil {
		s.Close()
		return nil, fmt.Errorf("error creating renderer: %v", err)
	}
	s.cleanup(s.renderer.Destroy)
	s.renderer.Clear()

	if s.cam, err = initCam(capWidth, capHeight); err != nil {
		s.Close()
		return nil, fmt.Errorf("failed to initialize webcam: %v", err)
	}
	s.cleanup(s.cam.Close)

	if s.tex, err = s.renderer.CreateTexture(sdl.PIXELFORMAT_YUY2, sdl.TEXTUREACCESS_STREAMING, int32(capWidth), int32(capHeight)); err != nil {
		s.Close()
		return nil, fmt.Errorf("error creating texture: %v", err)
	}
	s.cleanup(s.tex.Destroy)
	s.snaps = make([]*sdl.Texture, 4)
	s.snapfiles = make([]string, 4)
	for i := range s.snaps {
		if s.snaps[i], err = s.renderer.CreateTexture(sdl.PIXELFORMAT_ABGR8888, sdl.TEXTUREACCESS_STREAMING, 430, 287); err != nil {
			s.Close()
			return nil, fmt.Errorf("error creating texture: %v", err)
		}
		s.cleanup(s.snaps[i].Destroy)
	}
	usr, err := user.Current()
	if err != nil {
		s.Close()
		return nil, fmt.Errorf("failed to get home dir: %v", err)
	}
	s.savepath = filepath.Join(usr.HomeDir, "selfies", "snaps")

	font, err := makeFont(600)
	if err != nil {
		s.Close()
		return nil, fmt.Errorf("failed to read font: %v", err)
	}
	s.texes = make([]*sdl.Texture, 3)
	for i := 0; i < 3; i++ {
		surf, err := font.RenderUTF8Blended(strconv.Itoa(i+1), sdl.Color{R: 255, G: 255, B: 255, A: 255})
		if err != nil {
			s.Close()
			return nil, fmt.Errorf("failed to render text: %v", err)
		}
		if s.texes[i], err = s.renderer.CreateTextureFromSurface(surf); err != nil {
			s.Close()
			return nil, fmt.Errorf("failed to create texture from surface: %v", err)
		}
		s.cleanup(s.texes[i].Destroy)
		surf.Free()
		s.texes[i].SetBlendMode(sdl.BLENDMODE_BLEND)
	}

	font, err = makeFont(30)
	if err != nil {
		s.Close()
		return nil, fmt.Errorf("failed to read font: %v", err)
	}
	surf, err := font.RenderUTF8Blended("Print", sdl.Color{R: 255, G: 255, B: 0, A: 255})
	if err != nil {
		s.Close()
		return nil, fmt.Errorf("failed to render text: %v", err)
	}
	defer surf.Free()
	if s.printtex, err = s.renderer.CreateTextureFromSurface(surf); err != nil {
		s.Close()
		return nil, fmt.Errorf("failed to create texture from surface: %v", err)
	}
	s.cleanup(s.printtex.Destroy)
	s.printtex.SetBlendMode(sdl.BLENDMODE_BLEND)

	if surf, err = font.RenderUTF8Blended("Printing", sdl.Color{R: 255, G: 0, B: 0, A: 255}); err != nil {
		s.Close()
		return nil, fmt.Errorf("failed to render text: %v", err)
	}
	defer surf.Free()
	if s.printingtex, err = s.renderer.CreateTextureFromSurface(surf); err != nil {
		s.Close()
		return nil, fmt.Errorf("failed to create texture from surface: %v", err)
	}
	s.cleanup(s.printingtex.Destroy)
	s.printingtex.SetBlendMode(sdl.BLENDMODE_BLEND)

	if s.arduino, err = serial.Open(serial.OpenOptions{
		PortName:        "/dev/ttyUSB0",
		BaudRate:        9600,
		DataBits:        8,
		StopBits:        1,
		MinimumReadSize: 1,
	}); err != nil {
		s.Close()
		return nil, fmt.Errorf("serial.Open: %v", err)
	}
	s.cleanup(s.arduino.Close)
	s.arduino.Write([]byte{'R', '\r', '\n'}) // send a reset

	return s, nil
}

func (s *Selfies) cleanup(f func() error) {
	s.cleanups = append(s.cleanups, f)
}

func (s *Selfies) Close() {
	for _, c := range s.cleanups {
		c()
	}
}

func frameToImage(frame []byte, width int, height int) image.Image {
	img := image.NewYCbCr(image.Rect(0, 0, width, height), image.YCbCrSubsampleRatio422)
	for i := 0; i < int(capWidth*capHeight); i++ {
		img.Y[i] = frame[i*2]
		if i%2 == 0 {
			img.Cb[i/2] = frame[i*2+1]
		} else {
			img.Cr[i/2] = frame[i*2+1]
		}
	}
	cropped := image.NewRGBA(image.Rect(0, 0, 1080, 720))
	if height != 720 {
		resized := resize.Resize(0, 720, img, resize.Bicubic)
		draw.Draw(cropped, cropped.Bounds(), resized, image.Point{(resized.Bounds().Dx() - 1080) / 2, 0}, draw.Over)
	} else {
		draw.Draw(cropped, cropped.Bounds(), img, image.Point{(img.Bounds().Dx() - 1080) / 2, 0}, draw.Over)
	}
	return cropped
}

func saveImage(img image.Image, filename string) {
	if fp, err := os.Create(filename); err == nil {
		jpeg.Encode(fp, img, &jpeg.Options{Quality: 95})
		fp.Sync()
		fp.Close()
	}
}

func printFile(filename string) {
	exec.Command("/usr/bin/obexftp", "--nopath", "--noconn", "--uuid", "none",
		"--bluetooth", "C4:30:18:19:C6:3D", "--channel", "4", "-p", filename).Run()
}

func (s *Selfies) drawCountdown(buttonPressed time.Time) {
	if time.Since(buttonPressed) > time.Millisecond*3000 {
		_, _, texWidth, texHeight, _ := s.texes[0].Query()
		s.renderer.Copy(s.texes[0],
			&sdl.Rect{X: 0, Y: 0, W: texWidth, H: texHeight},
			&sdl.Rect{X: (s.screenWidth - texWidth) / 2, Y: (s.screenHeight - texHeight) / 2, W: texWidth, H: texHeight})
	} else if time.Since(buttonPressed) > time.Millisecond*1500 {
		_, _, texWidth, texHeight, _ := s.texes[1].Query()
		s.renderer.Copy(s.texes[1],
			&sdl.Rect{X: 0, Y: 0, W: texWidth, H: texHeight},
			&sdl.Rect{X: (s.screenWidth - texWidth) / 2, Y: (s.screenHeight - texHeight) / 2, W: texWidth, H: texHeight})
	} else {
		_, _, texWidth, texHeight, _ := s.texes[2].Query()
		s.renderer.Copy(s.texes[2],
			&sdl.Rect{X: 0, Y: 0, W: texWidth, H: texHeight},
			&sdl.Rect{X: (s.screenWidth - texWidth) / 2, Y: (s.screenHeight - texHeight) / 2, W: texWidth, H: texHeight})
	}
}

func (s *Selfies) Run() {
	buttonPress := make(chan byte)

	go func() {
		start := time.Now()
		b := make([]byte, 1)
		for true {
			s.arduino.Read(b)
			if b[0] > ' ' && time.Since(start) > 5*time.Second { // ignore button presses for first few seconds
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
				s.arduino.Write([]byte{'R', '\r', '\n'})
				buttonPressed = time.Now()
			} else if button == '3' && s.snapfiles[0] != "" && time.Since(printCooldown) > time.Second*30 {
				printCooldown = time.Now()
				go func() {
					printnotify <- true
					printFile(s.snapfiles[0])
					printnotify <- false
				}()
			}
		case printing = <-printnotify:
		default:
		}
		s.renderer.Clear()
		for {
			if f, _ := s.cam.ReadFrame(); f != nil && len(f) != 0 {
				frame = f
				s.tex.Update(&sdl.Rect{X: 0, Y: 0, W: int32(capWidth), H: int32(720)}, frame, 2*int(capHeight))
			} else {
				break
			}
		}
		snapWidth := int32((s.screenWidth - 40) / 2)
		snapHeight := int32(int(math.Round((float64(snapWidth) / 3) * 2)))
		if s.snapfiles[0] != "" {
			var tex *sdl.Texture
			if printing {
				s.renderer.SetDrawColor(uint8(rand.Int()%255), uint8(rand.Int()%255), uint8(rand.Int()%255), 255)
				tex = s.printingtex
			} else {
				s.renderer.SetDrawColor(255, 255, 0, 255)
				tex = s.printtex
			}
			_, _, texWidth, texHeight, _ := tex.Query()
			s.renderer.FillRect(&sdl.Rect{X: 0, Y: 800, W: snapWidth, H: snapHeight})
			s.renderer.Copy(tex,
				&sdl.Rect{X: 0, Y: 0, W: texWidth, H: texHeight},
				&sdl.Rect{X: (snapWidth - texWidth) / 2, Y: (800 - texHeight), W: texWidth, H: texHeight})
			s.renderer.Copy(s.snaps[0], &sdl.Rect{X: 2, Y: 2, W: snapWidth, H: snapHeight},
				&sdl.Rect{X: 2, Y: 802, W: 426, H: 283})
		}
		s.renderer.SetDrawColor(0, 0, 0, 255)
		s.renderer.Copy(s.tex, &sdl.Rect{X: 0, Y: 0, W: capWidth, H: capHeight},
			&sdl.Rect{X: -90, Y: 0, W: 1080, H: 600})
		s.renderer.Copy(s.snaps[1], &sdl.Rect{X: 0, Y: 0, W: snapWidth, H: snapHeight},
			&sdl.Rect{X: 470, Y: 800, W: snapWidth, H: snapHeight})
		s.renderer.Copy(s.snaps[2], &sdl.Rect{X: 0, Y: 0, W: snapWidth, H: snapHeight},
			&sdl.Rect{X: 0, Y: 1237, W: snapWidth, H: snapHeight})
		s.renderer.Copy(s.snaps[3], &sdl.Rect{X: 0, Y: 0, W: snapWidth, H: snapHeight},
			&sdl.Rect{X: 470, Y: 1237, W: snapWidth, H: snapHeight})

		if !buttonPressed.IsZero() {
			if !focus && time.Since(buttonPressed) > time.Millisecond*4000 { // turn on focus lock
				focus = true
				s.arduino.Write([]byte{'A', '\r', '\n'})
			}
			if !lights && time.Since(buttonPressed) > time.Millisecond*3500 { // turn on lights
				lights = true
				s.arduino.Write([]byte{'B', '\r', '\n'})
			}
			if time.Since(buttonPressed) > time.Millisecond*4500 {
				if frame != nil && len(frame) != 0 {
					// flash a white screen
					s.renderer.SetDrawColor(255, 255, 255, 255)
					s.renderer.Clear()
					s.renderer.Present()
					s.renderer.SetDrawColor(0, 0, 0, 255)
					filename := filepath.Join(s.savepath, fmt.Sprintf("%d.jpg", time.Now().Unix()))
					cropped := frameToImage(frame, int(capWidth), int(capHeight))
					saveImage(cropped, filename)
					snap := image.NewRGBA(image.Rect(0, 0, int(snapWidth), int(snapHeight)))
					draw.Draw(snap, snap.Bounds(),
						resize.Resize(uint(snapWidth), uint(snapHeight), cropped, resize.Bicubic),
						image.ZP, draw.Over)
					s.snaps[0], s.snaps[1], s.snaps[2], s.snaps[3] = s.snaps[3], s.snaps[0], s.snaps[1], s.snaps[2]
					s.snaps[0].Update(&sdl.Rect{X: 0, Y: 0, W: snapWidth, H: snapHeight}, snap.Pix, snap.Stride)
					s.snapfiles[0], s.snapfiles[1], s.snapfiles[2], s.snapfiles[3] = filename, s.snapfiles[0], s.snapfiles[1], s.snapfiles[2]
				} else {
					fmt.Println("BAD FRAME")
				}
				s.arduino.Write([]byte{'C', '\r', '\n'}) // trigger shutter release
				time.Sleep(time.Millisecond * 200)
				s.arduino.Write([]byte{'R', '\r', '\n'})                 // reset all relays
				lights, focus, buttonPressed = false, false, time.Time{} // reset state machine
			} else {
				s.drawCountdown(buttonPressed)
			}
		}
		s.renderer.Present()
	}
}
