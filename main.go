package main

import (
	"bubblebobble/internal/app"
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
	touch := flag.Bool("touch", false, "show virtual controls for desktop touch testing")
	frames := flag.Int("smoke-frames", 0, "exit after this many frames (automated validation)")
	scene := flag.String("smoke-scene", "menu", "scene for automated validation: menu, game, coop, pause, win, gameover, editor, profiles, new-profile, settings, scores, help, bluetooth, credits")
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
	a, err := app.New(*directory, *mute)
	if err != nil {
		log.Fatal(err)
	}
	defer a.Close()
	a.EnableTouch(*touch)
	if *frames > 0 {
		if err = a.ConfigureSmoke(*frames, *scene, *round, *capture); err != nil {
			log.Fatal(err)
		}
	}
	ebiten.SetTPS(60)
	ebiten.SetWindowSize(screenWidth, screenHeight)
	if *touch {
		ebiten.SetWindowSize(1280, 720)
	}
	ebiten.SetWindowTitle("Bubble Bobble")
	if icon, err := app.Icon(); err == nil {
		ebiten.SetWindowIcon([]image.Image{icon})
	}
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowClosingHandled(true)
	ebiten.SetRunnableOnUnfocused(true)
	ebiten.SetFullscreen(a.Fullscreen())
	if err = ebiten.RunGame(a); err != nil {
		log.Fatal(err)
	}
}
