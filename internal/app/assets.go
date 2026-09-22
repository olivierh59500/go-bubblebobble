package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"

	"bubblebobble/resources"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type artwork struct {
	animations map[string][]*ebiten.Image
	tiles      [26]*ebiten.Image
	logo       *ebiten.Image
	font       *text.GoTextFaceSource
}

func loadArtwork() (*artwork, error) {
	a := &artwork{animations: map[string][]*ebiten.Image{}}
	data, err := resources.Files.ReadFile("sprites.json")
	if err != nil {
		return nil, err
	}
	var manifest map[string][]string
	if err = json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	for name, paths := range manifest {
		if len(paths) == 0 {
			return nil, fmt.Errorf("empty animation %s", name)
		}
		for _, path := range paths {
			img, err := readImage(path)
			if err != nil {
				return nil, err
			}
			a.animations[name] = append(a.animations[name], ebiten.NewImageFromImage(img))
		}
	}
	for i := 1; i <= 25; i++ {
		img, err := readImage(fmt.Sprintf("tiles/%02d.png", i))
		if err != nil {
			return nil, err
		}
		a.tiles[i] = ebiten.NewImageFromImage(img)
	}
	logo, err := readImage("logo.png")
	if err != nil {
		return nil, err
	}
	a.logo = ebiten.NewImageFromImage(logo)
	font, err := resources.Files.ReadFile("arcade.ttf")
	if err != nil {
		return nil, err
	}
	a.font, err = text.NewGoTextFaceSource(bytes.NewReader(font))
	return a, err
}

func readImage(path string) (image.Image, error) {
	b, err := resources.Files.ReadFile(path)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return img, nil
}

func (a *artwork) frame(name string, tick int) *ebiten.Image {
	frames := a.animations[name]
	if len(frames) == 0 {
		return nil
	}
	return frames[max(0, tick)/10%len(frames)]
}
