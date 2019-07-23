package main

import (
	"fmt"
	"image"
	"image/draw"
	"io/ioutil"
	"log"
	"os"
	"time"
	"unsafe"

	"github.com/blackjack/webcam"
	"github.com/golang/freetype/truetype"
	"github.com/veandco/go-sdl2/sdl"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

func textImage(text string) *image.RGBA {
	fontBytes, err := ioutil.ReadFile("Raleway-Black.ttf")
	if err != nil {
		log.Fatalf("failed to read fong: %v", err)
	}
	f, err := truetype.Parse(fontBytes)
	if err != nil {
		log.Fatalf("failed to parse font: %v", err)
	}
	im := image.NewRGBA(image.Rect(0, 0, 900, 1600))
	draw.Draw(im, im.Bounds(), image.Transparent, image.ZP, draw.Src)
	d := &font.Drawer{
		Dst: im,
		Src: image.White,
		Face: truetype.NewFace(f, &truetype.Options{
			Size: 100,
			DPI:  300,
		}),
	}
	d.Dot = fixed.Point26_6{
		X: (fixed.I(900) - d.MeasureString(text)) / 2,
		Y: fixed.I(800),
	}
	d.DrawString(text)
	return im
}

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

	window, err := sdl.CreateWindow("SELFIES", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		900, 1600, sdl.WINDOW_SHOWN|sdl.WINDOW_FULLSCREEN|sdl.WINDOW_BORDERLESS)
	if err != nil {
		log.Fatalf("failed to create window: %v", err)
	}
	defer window.Destroy()

	surface, err := window.GetSurface()
	if err != nil {
		log.Fatalf("failed to create surface: %v", err)
	}

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
		log.Fatalf("failed to begin cam streaming: %v", err)
	}
	cam.SetBufferCount(1)

	text3 := textImage("3")
	textSurface, err := sdl.CreateRGBSurface(0, 900, 1600, 32, 0, 0, 0, 0)
	if err != nil {
		log.Fatalf("error making text surface: %v", err)
	}
	dp := textSurface.Pixels()
	for i := range dp {
		dp[i] = byte(text3.Pix[i])
	}
	copy(dp, text3.Pix)

	for {
		t := time.Now()
		/*
			if button.EdgeDetected() { // cleeeeeck
				fmt.Println("CLEEEEEECK")
			}
		*/
		for {
			if frame, frameIndex, _ := cam.GetFrame(); frame != nil && len(frame) != 0 {
				frameSurface, err := sdl.CreateRGBSurface(0, 320, 240, 32, 0, 0, 0, 0)
				if err != nil {
					log.Fatalf("error making frame surface: %v", err)
				}
				dp := frameSurface.Pixels()
				sdl.ConvertPixels(320, 240, sdl.PIXELFORMAT_YUY2, unsafe.Pointer(&frame[0]), 320*2,
					frameSurface.Format.Format, unsafe.Pointer(&dp[0]), int(frameSurface.Pitch))
				cam.ReleaseFrame(frameIndex)
				frameSurface.BlitScaled(&sdl.Rect{X: 10, Y: 20, W: 300, H: 200}, surface, &sdl.Rect{X: 0, Y: 0, W: 900, H: 600})
				frameSurface.Free()
			} else {
				break
			}
		}
		textSurface.Blit(&sdl.Rect{X: 0, Y: 0, W: 900, H: 1600}, surface, &sdl.Rect{X: 0, Y: 0, W: 900, H: 1600})
		window.UpdateSurface()
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
