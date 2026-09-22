package app

import (
	"bubblebobble/internal/game"
	"math"
)

func blendBody(from, to game.Body, alpha float64) game.Body {
	if math.Abs(from.X-to.X) > 6*game.Tile || math.Abs(from.Y-to.Y) > 6*game.Tile {
		return to
	}
	alpha = max(0, min(1, alpha))
	to.X = from.X + (to.X-from.X)*alpha
	to.Y = from.Y + (to.Y-from.Y)*alpha
	return to
}

func (a *App) rememberPositions() {
	positions := make(map[int]game.Body, len(a.match.Players)+len(a.match.Enemies)+len(a.match.Bubbles))
	for i, p := range a.match.Players {
		positions[-i-1] = a.renderBody(-i-1, p.Body)
	}
	for _, e := range a.match.Enemies {
		positions[e.ID] = a.renderBody(e.ID, e.Body)
	}
	for _, b := range a.match.Bubbles {
		positions[b.ID] = a.renderBody(b.ID, b.Body)
	}
	a.renderPositions = positions
	a.renderFrame = a.frame
}

func (a *App) renderBody(id int, body game.Body) game.Body {
	if a.peer == nil || a.netHost {
		return body
	}
	if previous, ok := a.renderPositions[id]; ok {
		return blendBody(previous, body, float64(a.frame-a.renderFrame)/3)
	}
	return body
}
