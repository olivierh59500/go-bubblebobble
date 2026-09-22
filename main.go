package main

import (
	"flag"
	"image"
	"log"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
)

const screenWidth, screenHeight = 768, 720

func main() {
	directory := flag.String("data-dir", "", "directory for profiles, settings and the custom level")
	mute := flag.Bool("mute", false, "start without audio")
	frames := flag.Int("smoke-frames", 0, "exit after this many frames (automated validation)")
	scene := flag.String("smoke-scene", "menu", "scene for automated validation: menu, game, pause, win, gameover, editor, profiles, settings, scores, help, credits")
	round := flag.Int("smoke-round", 1, "campaign round for automated validation")
	capture := flag.String("screenshot", "", "save the final automated frame as PNG")
	flag.Parse()
	if *directory == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			base = "."
		}
		*directory = filepath.Join(base, "bubblebobble")
	}
	a, err := newApp(*directory, *mute)
	if err != nil {
		log.Fatal(err)
	}
	defer a.audio.close()
	if *frames > 0 {
		a.smokeFrames, a.capturePath = *frames, *capture
		if err = a.setupSmoke(*scene, *round); err != nil {
			log.Fatal(err)
		}
	}
	ebiten.SetTPS(60)
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Bubble Bobble")
	if icon, err := readImage("icon.png"); err == nil {
		ebiten.SetWindowIcon([]image.Image{icon})
	}
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowClosingHandled(true)
	ebiten.SetRunnableOnUnfocused(true)
	ebiten.SetFullscreen(a.store.Data.Settings.Fullscreen)
	if err = ebiten.RunGame(a); err != nil {
		log.Fatal(err)
	}
}
