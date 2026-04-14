package main

import "github.com/hajimehoshi/ebiten/v2"

type actionKey int

const (
	actionJump actionKey = iota
	actionFire
)

func (g *game) keyDown(action actionKey, color dragonColor) bool {
	jumpKey := ebiten.KeySpace
	fireKey := ebiten.KeyA
	if g.playerCount > 1 {
		if color == dragonGreen {
			jumpKey = ebiten.KeyArrowUp
			fireKey = ebiten.KeyL
		} else {
			jumpKey = ebiten.KeyS
			fireKey = ebiten.KeyT
		}
	}
	switch action {
	case actionJump:
		return ebiten.IsKeyPressed(jumpKey)
	case actionFire:
		return ebiten.IsKeyPressed(fireKey)
	default:
		return false
	}
}

func (g *game) xAxis(color dragonColor) int {
	dir := 0
	rightKey := ebiten.KeyArrowRight
	leftKey := ebiten.KeyArrowLeft
	if color == dragonBlue {
		rightKey = ebiten.KeyD
		leftKey = ebiten.KeyA
	}
	if ebiten.IsKeyPressed(rightKey) {
		dir++
	}
	if ebiten.IsKeyPressed(leftKey) {
		dir--
	}
	return sign(dir)
}
