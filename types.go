package main

import "image/color"

type vec2i struct {
	X int
	Y int
}

func (v vec2i) add(x, y int) vec2i {
	return vec2i{X: v.X + x, Y: v.Y + y}
}

func (v vec2i) dot(other vec2i) int {
	return v.X*other.X + v.Y*other.Y
}

func (v *vec2i) normalizeSign() {
	v.X = sign(v.X)
	v.Y = sign(v.Y)
}

type position struct {
	X   int
	Y   int
	Dir int
}

func (p position) vec() vec2i {
	return vec2i{X: p.X, Y: p.Y}
}

type collider struct {
	W       int
	H       int
	OffsetX int
	OffsetY int
}

func (c collider) flipX(boxWidth int) collider {
	c.OffsetX = boxWidth - (c.OffsetX + c.W)
	return c
}

func (c collider) center(x, y int) vec2i {
	return vec2i{X: x + c.OffsetX + c.W/2, Y: y + c.OffsetY + c.H/2}
}

func (c collider) left(pos vec2i) int {
	return pos.X + c.OffsetX
}

func (c collider) right(pos vec2i) int {
	return pos.X + c.OffsetX + c.W
}

func (c collider) top(pos vec2i) int {
	return pos.Y + c.OffsetY
}

func (c collider) bottom(pos vec2i) int {
	return pos.Y + c.OffsetY + c.H
}

var (
	colorTransparent = color.RGBA{A: 0}
	colorWhite       = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	colorBlack       = color.RGBA{A: 255}
	colorGreen       = color.RGBA{G: 228, B: 48, A: 255}
	colorBlue        = color.RGBA{R: 0, G: 121, B: 241, A: 255}
	colorDarkBlue    = color.RGBA{R: 0, G: 82, B: 172, A: 255}
	colorBubblePop   = color.RGBA{R: 255, G: 240, B: 215, A: 255}
)

var (
	walkingActorCollider = collider{W: bpSize(2, 0), H: bpSize(1, 0), OffsetX: 0, OffsetY: bpSize(1, 0)}
	fullActorCollider    = collider{W: bpSize(2, 0), H: bpSize(2, 0)}
	bubbleCollider       = collider{W: bpSize(0, 28), H: bpSize(2, 0)}
	bubbleRepelCollider  = collider{W: bpSize(0, 14), H: bpSize(1, 0), OffsetX: bpSize(0, 7), OffsetY: bpSize(0, 8)}
	bubblePopCollider    = collider{W: bpSize(0, 28), H: bpSize(2, 0)}
	bubbleJumpCollider   = collider{W: bpSize(0, 28), H: bpSize(0, 4), OffsetY: bpSize(0, -2)}
	enemyHitCollider     = collider{W: bpSize(0, 24), H: bpSize(1, 0), OffsetX: bpSize(0, 4), OffsetY: bpSize(1, -4)}

	dragonSpikeColliders = []collider{
		{W: bpSize(0, 12), H: bpSize(2, -4), OffsetX: bpSize(2, -12), OffsetY: 0},
		{W: bpSize(1, 0), H: bpSize(0, 4), OffsetX: bpSize(1, 0), OffsetY: bpSize(2, -6)},
	}
	weakDragonBubblePushCollider   = collider{W: bpSize(0, 4), H: bpSize(2, 0), OffsetX: bpSize(0, 4)}
	strongDragonBubblePushCollider = collider{W: bpSize(0, 4), H: bpSize(2, 0), OffsetX: bpSize(0, 8)}
	enemyProjectileCollider        = collider{W: bpSize(0, 12), H: bpSize(1, 0), OffsetX: bpSize(0, 10), OffsetY: bpSize(0, 8)}
	bossCollider                   = collider{W: bpSize(8, 0), H: bpSize(8, 0)}

	flyingTopLeftHorizontal     = collider{W: bpSize(0, 17), H: bpSize(0, 2), OffsetX: bpSize(0, 2), OffsetY: bpSize(0, 2)}
	flyingTopRightHorizontal    = collider{W: bpSize(0, 17), H: bpSize(0, 2), OffsetX: bpSize(2, -17) - bpSize(0, 2), OffsetY: bpSize(0, 2)}
	flyingBottomLeftHorizontal  = collider{W: bpSize(0, 17), H: bpSize(0, 2), OffsetX: bpSize(0, 2), OffsetY: bpSize(2, -2) - bpSize(0, 2)}
	flyingBottomRightHorizontal = collider{W: bpSize(0, 17), H: bpSize(0, 2), OffsetX: bpSize(2, -17) - bpSize(0, 2), OffsetY: bpSize(2, -2) - bpSize(0, 2)}
	flyingTopLeftVertical       = collider{W: bpSize(0, 2), H: bpSize(0, 17), OffsetX: bpSize(0, 2), OffsetY: bpSize(0, 2)}
	flyingTopRightVertical      = collider{W: bpSize(0, 2), H: bpSize(0, 17), OffsetX: bpSize(2, -2) - bpSize(0, 2), OffsetY: bpSize(0, 2)}
	flyingBottomLeftVertical    = collider{W: bpSize(0, 2), H: bpSize(0, 17), OffsetX: bpSize(0, 2), OffsetY: bpSize(2, -17) - bpSize(0, 2)}
	flyingBottomRightVertical   = collider{W: bpSize(0, 2), H: bpSize(0, 17), OffsetX: bpSize(2, -2) - bpSize(0, 2), OffsetY: bpSize(2, -17) - bpSize(0, 2)}
)

func sign(x int) int {
	switch {
	case x > 0:
		return 1
	case x < 0:
		return -1
	default:
		return 0
	}
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func overlaps(pos1 position, col1 collider, pos2 position, col2 collider) bool {
	xmin1 := pos1.X + col1.OffsetX
	xmax1 := xmin1 + col1.W
	ymin1 := pos1.Y + col1.OffsetY
	ymax1 := ymin1 + col1.H
	xmin2 := pos2.X + col2.OffsetX
	xmax2 := xmin2 + col2.W
	ymin2 := pos2.Y + col2.OffsetY
	ymax2 := ymin2 + col2.H
	return xmax1 > xmin2 && xmax2 > xmin1 && ymax1 > ymin2 && ymax2 > ymin1
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := false
	if v < 0 {
		neg = true
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
