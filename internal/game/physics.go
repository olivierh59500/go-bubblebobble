package game

import "math"

func (g *Game) spawnPosition(cell Cell) (float64, float64) {
	x, y := cell.Position(ActorSize)
	if !g.blocked(x, y, ActorSize) {
		return x, y
	}
	best := math.MaxFloat64
	bestX, bestY := x, y
	for row := 1; row < Rows-2; row++ {
		for col := 2; col < Columns-3; col++ {
			xx, yy := float64(col*Tile), float64(row*Tile)
			distance := math.Abs(xx-x) + math.Abs(yy-y)
			if distance < best && !g.blocked(xx, yy, ActorSize) {
				best, bestX, bestY = distance, xx, yy
			}
		}
	}
	return bestX, bestY
}

func (g *Game) blocked(x, y, size float64) bool {
	for row := int(math.Floor(y / Tile)); row <= int(math.Floor((y+size-0.01)/Tile)); row++ {
		for col := int(math.Floor(x / Tile)); col <= int(math.Floor((x+size-0.01)/Tile)); col++ {
			if g.Level.Solid(col, row) {
				return true
			}
		}
	}
	return false
}

// Platforms are one-way while rising. Sweeping the feet avoids tunnelling on descent.
func (g *Game) moveActor(b *Body, dx float64, gravity bool) {
	inside := g.blocked(b.X+2, b.Y+2, ActorSize-4)
	steps := max(1, int(math.Ceil(math.Abs(dx))))
	for i := 0; i < steps; i++ {
		x := max(float64(2*Tile), min(float64(Width-2*Tile-ActorSize), b.X+dx/float64(steps)))
		if b.VY >= 0 && !inside && g.blocked(x+2, b.Y+2, ActorSize-4) {
			break
		}
		b.X = x
	}
	if gravity {
		b.VY = min(12, b.VY+0.5)
	}
	oldFeet := b.Y + ActorSize
	newY := b.Y + b.VY
	b.Ground = false
	if b.VY >= 0 {
		first := int(math.Ceil(oldFeet / Tile))
		last := int(math.Floor((newY + ActorSize) / Tile))
		for row := first; row <= last; row++ {
			for col := int((b.X + 4) / Tile); col <= int((b.X+ActorSize-4)/Tile); col++ {
				if g.Level.Solid(col, row) {
					newY = float64(row*Tile - ActorSize)
					b.VY = 0
					b.Ground = true
					break
				}
			}
			if b.Ground {
				break
			}
		}
	}
	b.Y = newY
	if b.Y < Tile {
		b.Y = Tile
		b.VY = max(0, b.VY)
	}
	if b.Y > Height {
		b.Y = Tile
		b.VY = 2
	}
}

func (g *Game) moveFlying(e *Enemy, dx, dy float64) {
	steps := max(1, int(math.Ceil(max(math.Abs(dx), math.Abs(dy)))))
	for i := 0; i < steps; i++ {
		x := e.X + dx/float64(steps)
		if g.blocked(x+3, e.Y+3, ActorSize-6) {
			e.Dir *= -1
			break
		}
		e.X = x
	}
	for i := 0; i < steps; i++ {
		y := e.Y + dy/float64(steps)
		if y < Tile || y > Height-ActorSize || g.blocked(e.X+3, y+3, ActorSize-6) {
			e.Vertical *= -1
			break
		}
		e.Y = y
	}
}
