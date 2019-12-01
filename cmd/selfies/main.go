package main

import (
	"log"
	"os"

	"github.com/redbo/selfies"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

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

	sdl.WarpMouseGlobal(900, 1600)
	sdl.ShowCursor(sdl.DISABLE)

	s, err := selfies.NewSelfies()
	if err != nil {
		log.Fatalf("failed to start selfies: %v", err)
	}
	s.Run()
}
