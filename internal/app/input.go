package app

import (
	"bubblebobble/internal/game"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func pressed(keys ...ebiten.Key) bool {
	for _, key := range keys {
		if inpututil.IsKeyJustPressed(key) {
			return true
		}
	}
	return false
}

func held(keys ...ebiten.Key) bool {
	for _, key := range keys {
		if ebiten.IsKeyPressed(key) {
			return true
		}
	}
	return false
}

func padButton(button ebiten.StandardGamepadButton, just bool) bool {
	for _, id := range ebiten.AppendGamepadIDs(nil) {
		if !ebiten.IsStandardGamepadLayoutAvailable(id) {
			continue
		}
		if just && inpututil.IsStandardGamepadButtonJustPressed(id, button) || !just && ebiten.IsStandardGamepadButtonPressed(id, button) {
			return true
		}
	}
	return false
}

func playInput() game.Input {
	in := game.Input{Jump: held(ebiten.KeyArrowUp, ebiten.KeyW), Fire: held(ebiten.KeySpace, ebiten.KeyControl)}
	if held(ebiten.KeyArrowLeft, ebiten.KeyA) {
		in.Move--
	}
	if held(ebiten.KeyArrowRight, ebiten.KeyD) {
		in.Move++
	}
	for _, id := range ebiten.AppendGamepadIDs(nil) {
		if !ebiten.IsStandardGamepadLayoutAvailable(id) {
			continue
		}
		x := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickHorizontal)
		if x < -0.25 {
			in.Move = -1
		} else if x > 0.25 {
			in.Move = 1
		}
	}
	if padButton(ebiten.StandardGamepadButtonLeftLeft, false) {
		in.Move = -1
	}
	if padButton(ebiten.StandardGamepadButtonLeftRight, false) {
		in.Move = 1
	}
	in.Jump = in.Jump || padButton(ebiten.StandardGamepadButtonRightBottom, false)
	in.Fire = in.Fire || padButton(ebiten.StandardGamepadButtonRightLeft, false)
	return in
}

func confirmPressed() bool {
	return pressed(ebiten.KeyEnter, ebiten.KeySpace) || padButton(ebiten.StandardGamepadButtonRightBottom, true)
}
func backPressed() bool {
	return pressed(ebiten.KeyEscape) || padButton(ebiten.StandardGamepadButtonRightRight, true)
}
