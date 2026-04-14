package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

type spriteHandle int

const invalidSpriteHandle spriteHandle = -1

type sprite struct {
	Image   *ebiten.Image
	Rect    image.Rectangle
	XOffset int
	YOffset int
	Name    string
}

type assets struct {
	sprites    []sprite
	spriteMap  map[string]spriteHandle
	animations map[string]*animation
	fontData   *opentype.Font
	fontFaces  map[int]font.Face
}

type asepriteSlicesFile struct {
	Meta struct {
		Slices []struct {
			Name string `json:"name"`
			Keys []struct {
				Bounds struct {
					X int `json:"x"`
					Y int `json:"y"`
					W int `json:"w"`
					H int `json:"h"`
				} `json:"bounds"`
				Pivot *struct {
					X int `json:"x"`
					Y int `json:"y"`
				} `json:"pivot"`
			} `json:"keys"`
		} `json:"slices"`
	} `json:"meta"`
}

type animationsFile struct {
	Animations []struct {
		Name            string `json:"name"`
		FramesPerSprite int    `json:"FramesPerSprite"`
	} `json:"animations"`
}

func loadAssets() (*assets, error) {
	a := &assets{
		spriteMap:  map[string]spriteHandle{},
		animations: map[string]*animation{},
		fontFaces:  map[int]font.Face{},
	}
	if err := a.addSpriteSheet("res/sprites/MainSpriteSheet.png", "res/sprites/MainSpriteSheet.json"); err != nil {
		return nil, err
	}
	if err := a.addSpriteSheet("res/sprites/LevelTiles.png", "res/sprites/LevelTiles.json"); err != nil {
		return nil, err
	}
	if err := a.addSingleSprite("res/sprites/LevelTileShadowRight.png", "TileShadowRight"); err != nil {
		return nil, err
	}
	if err := a.addSingleSprite("res/sprites/LevelTileShadowBottem.png", "TileShadowBottem"); err != nil {
		return nil, err
	}
	if err := a.loadAnimations("res/sprites/Animations.json"); err != nil {
		return nil, err
	}
	fontBytes, err := os.ReadFile("res/fonts/C64_Pro_Mono-STYLE.ttf")
	if err != nil {
		return nil, fmt.Errorf("load font: %w", err)
	}
	fontData, err := opentype.Parse(fontBytes)
	if err != nil {
		return nil, fmt.Errorf("parse font: %w", err)
	}
	a.fontData = fontData
	return a, nil
}

func (a *assets) addSpriteSheet(imagePath, jsonPath string) error {
	img, err := loadImage(imagePath)
	if err != nil {
		return err
	}
	var data asepriteSlicesFile
	if err := loadJSON(jsonPath, &data); err != nil {
		return err
	}
	for _, slice := range data.Meta.Slices {
		if len(slice.Keys) == 0 {
			continue
		}
		key := slice.Keys[0]
		xOffset := 0
		yOffset := 0
		if key.Pivot != nil {
			xOffset = key.Pivot.X
			yOffset = key.Pivot.Y
		}
		rect := image.Rect(key.Bounds.X, key.Bounds.Y, key.Bounds.X+key.Bounds.W, key.Bounds.Y+key.Bounds.H)
		a.addSprite(sprite{Image: img, Rect: rect, XOffset: xOffset, YOffset: yOffset, Name: slice.Name})
	}
	return nil
}

func (a *assets) addSingleSprite(imagePath, name string) error {
	img, err := loadImage(imagePath)
	if err != nil {
		return err
	}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	a.addSprite(sprite{Image: img, Rect: image.Rect(0, 0, w, h), Name: name})
	return nil
}

func (a *assets) addSprite(s sprite) {
	handle := spriteHandle(len(a.sprites))
	a.sprites = append(a.sprites, s)
	a.spriteMap[s.Name] = handle
}

func (a *assets) loadAnimations(path string) error {
	var data animationsFile
	if err := loadJSON(path, &data); err != nil {
		return err
	}
	for _, def := range data.Animations {
		anim := &animation{Name: def.Name, FramesPerSprite: def.FramesPerSprite}
		for i := 1; ; i++ {
			handle, ok := a.spriteHandle(def.Name + "-" + itoa(i))
			if !ok {
				break
			}
			anim.Frames = append(anim.Frames, handle)
		}
		a.animations[def.Name] = anim
	}
	return nil
}

func (a *assets) spriteHandle(name string) (spriteHandle, bool) {
	handle, ok := a.spriteMap[name]
	return handle, ok
}

func (a *assets) mustSprite(name string) spriteHandle {
	handle, ok := a.spriteHandle(name)
	if !ok {
		return invalidSpriteHandle
	}
	return handle
}

func (a *assets) sprite(handle spriteHandle) (sprite, bool) {
	if handle < 0 || int(handle) >= len(a.sprites) {
		return sprite{}, false
	}
	return a.sprites[handle], true
}

func (a *assets) animation(name string) *animation {
	if anim, ok := a.animations[name]; ok {
		return anim
	}
	return &animation{Name: name, FramesPerSprite: 1}
}

func (a *assets) font(size int) font.Face {
	if face, ok := a.fontFaces[size]; ok {
		return face
	}
	face, err := opentype.NewFace(a.fontData, &opentype.FaceOptions{
		Size:    float64(size),
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil
	}
	a.fontFaces[size] = face
	return face
}

func loadImage(path string) (*ebiten.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open image %s: %w", path, err)
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("decode image %s: %w", path, err)
	}
	return ebiten.NewImageFromImage(img), nil
}

func loadJSON(path string, out any) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open json %s: %w", path, err)
	}
	defer file.Close()
	if err := json.NewDecoder(file).Decode(out); err != nil {
		return fmt.Errorf("decode json %s: %w", path, err)
	}
	return nil
}

func applyColorScale(opts *ebiten.DrawImageOptions, c color.RGBA) {
	opts.ColorScale.ScaleWithColor(c)
}
