package app

import "image"

const ScreenWidth, ScreenHeight = 768, 720
const screenWidth, screenHeight = ScreenWidth, ScreenHeight

func (a *App) Close()            { a.stopNetwork(); a.audio.close() }
func (a *App) Fullscreen() bool  { return a.store.Data.Settings.Fullscreen }
func Icon() (image.Image, error) { return readImage("icon.png") }
func (a *App) ConfigureSmoke(frames int, scene string, round int, path string) error {
	a.smokeFrames, a.capturePath = frames, path
	return a.SetupSmoke(scene, round)
}
