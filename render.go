package main

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
)

func (g *game) drawTitle(screen *ebiten.Image) {
	for _, sprite := range g.titleSprites {
		g.drawSprite(screen, sprite.Sprite, sprite.Pos, sprite.ScaleX, sprite.ScaleY, sprite.Color)
	}
	for _, textLine := range g.titleTexts {
		g.drawText(screen, textLine)
	}
}

func (g *game) drawGameplay(screen *ebiten.Image) {
	g.drawLevel(screen)
	for i := range g.items {
		item := &g.items[i]
		g.drawSprite(screen, g.assets.mustSprite(itemSpriteName(item.Type)), item.Pos, 2, 2, colorWhite)
	}
	for i := range g.floatTexts {
		t := &g.floatTexts[i]
		g.drawSprite(screen, t.Sprite, t.Pos, 2, 2, colorWhite)
	}
	for i := range g.projectiles {
		p := &g.projectiles[i]
		g.drawSprite(screen, p.Animator.sprite(), p.Pos, 2, 2, colorWhite)
	}
	for i := range g.enemies {
		g.drawEnemy(screen, &g.enemies[i])
	}
	for i := range g.tumbles {
		t := &g.tumbles[i]
		g.drawSprite(screen, t.Animator.sprite(), t.Pos, 2, 2, colorWhite)
	}
	for i := range g.bubbles {
		g.drawBubble(screen, &g.bubbles[i])
	}
	for i := range g.dragons {
		g.drawDragon(screen, &g.dragons[i])
	}
	g.drawText(screen, uiText{Pos: vec2i{X: bpSize(26, 0), Y: 0}, Text: "HI SCORE\n" + itoa(g.score), Color: colorGreen, FontSize: 32})
	g.drawText(screen, uiText{Pos: vec2i{X: 0, Y: bpSize(0, 2)}, Text: itoa(g.levelNumber), Color: colorWhite, FontSize: 32})
}

func (g *game) drawLevel(screen *ebiten.Image) {
	if g.level == nil {
		return
	}
	block := g.assets.mustSprite("Block-Level" + itoa(g.levelNumber%100))
	shadowRight := g.assets.mustSprite("TileShadowRight")
	shadowBottom := g.assets.mustSprite("TileShadowBottem")

	for x := 2; x < levelWidth+2; x++ {
		for y := 0; y < levelHeight; y++ {
			tileX := x - 2
			if g.level.Tiles.isEmpty(tileX, y) {
				continue
			}
			addRight := outOfLevelRange(tileX+1, y) || g.level.Tiles.isEmpty(tileX+1, y)
			addBottom := outOfLevelRange(tileX, y+1) || g.level.Tiles.isEmpty(tileX, y+1)
			if addRight {
				g.drawSprite(screen, shadowRight, position{X: (x + 1) * unitsPerBlock, Y: y * unitsPerBlock, Dir: -1}, 2, 2, g.level.ShadeRight)
			}
			if addBottom {
				g.drawSprite(screen, shadowBottom, position{X: x * unitsPerBlock, Y: (y + 1) * unitsPerBlock, Dir: -1}, 2, 2, g.level.ShadeBottom)
			}
		}
	}
	for _, x := range []int{0, 1, 30, 31} {
		for y := 0; y < levelHeight; y++ {
			if x == 1 {
				g.drawSprite(screen, shadowRight, position{X: (x + 1) * unitsPerBlock, Y: y * unitsPerBlock, Dir: -1}, 2, 2, g.level.ShadeRight)
			}
		}
	}
	for x := 2; x < levelWidth+2; x++ {
		for y := 0; y < levelHeight; y++ {
			if !g.level.Tiles.isEmpty(x-2, y) {
				g.drawSprite(screen, block, position{X: x * unitsPerBlock, Y: y * unitsPerBlock, Dir: -1}, 2, 2, colorWhite)
			}
		}
	}
	for _, x := range []int{0, 1, 30, 31} {
		for y := 0; y < levelHeight; y++ {
			g.drawSprite(screen, block, position{X: x * unitsPerBlock, Y: y * unitsPerBlock, Dir: -1}, 2, 2, colorWhite)
		}
	}
}

func (g *game) drawEnemy(screen *ebiten.Image, e *enemy) {
	switch e.Mode {
	case enemyModeBubbled:
		g.drawSprite(screen, e.Float.Animator.sprite(), e.Pos, 2, 2, colorWhite)
	case enemyModePopping:
		col := colorWhite
		if !e.Pop.PrePop {
			col = colorBubblePop
		}
		g.drawSprite(screen, e.Pop.Animator.sprite(), e.Pos, 2, 2, col)
	default:
		if e.Appearance != nil {
			g.drawSprite(screen, e.Appearance.Animator.sprite(), e.Pos, 2, 2, colorWhite)
		} else if e.Boss != nil {
			g.drawSprite(screen, e.Boss.Animator.sprite(), e.Pos, 2, 2, colorWhite)
		} else if e.Walking != nil {
			g.drawSprite(screen, e.Walking.Animator.sprite(), e.Pos, 2, 2, colorWhite)
		} else if e.Flying != nil {
			g.drawSprite(screen, e.Flying.Animator.sprite(), e.Pos, 2, 2, colorWhite)
		}
	}
}

func (g *game) drawDragon(screen *ebiten.Image, d *dragon) {
	if d.Hit != nil {
		g.drawSprite(screen, d.Hit.Animator.sprite(), d.Pos, 2, 2, colorWhite)
		return
	}
	col := colorWhite
	if d.InvFrames > 0 {
		if (d.InvFrames/2)%2 == 0 {
			col = colorTransparent
		} else {
			col = color.RGBA{R: 255, G: 230, B: 200, A: 255}
		}
	}
	g.drawSprite(screen, d.Animator.sprite(), d.Pos, 2, 2, col)
}

func (g *game) drawBubble(screen *ebiten.Image, b *bubble) {
	switch b.Kind {
	case bubbleShooting:
		g.drawSprite(screen, b.Shoot.Animator.sprite(), b.Pos, 2, 2, colorWhite)
	case bubbleFloating:
		g.drawSprite(screen, b.Float.Animator.sprite(), b.Pos, 2, 2, colorWhite)
	case bubblePopping:
		col := colorWhite
		if !b.Pop.PrePop {
			col = colorBubblePop
		}
		g.drawSprite(screen, b.Pop.Animator.sprite(), b.Pos, 2, 2, col)
	}
}

func (g *game) drawSprite(screen *ebiten.Image, handle spriteHandle, pos position, scaleX, scaleY float64, tint color.RGBA) {
	if tint.A == 0 {
		return
	}
	sprite, ok := g.assets.sprite(handle)
	if !ok {
		return
	}
	source := sprite.Image.SubImage(sprite.Rect).(*ebiten.Image)
	opts := &ebiten.DrawImageOptions{}
	drawScaleX := scaleX * scalingFactor
	drawScaleY := scaleY * scalingFactor
	width := float64(sprite.Rect.Dx()) * drawScaleX
	x := float64(pos.X)
	if pos.Dir > 0 {
		opts.GeoM.Scale(-drawScaleX, drawScaleY)
		x = x*scalingFactor + width
	} else {
		opts.GeoM.Scale(drawScaleX, drawScaleY)
		x = (x + float64(sprite.XOffset)) * scalingFactor
	}
	y := float64(pos.Y+sprite.YOffset) * scalingFactor
	opts.GeoM.Translate(x, y)
	applyColorScale(opts, tint)
	screen.DrawImage(source, opts)
}

func (g *game) drawText(screen *ebiten.Image, ui uiText) {
	face := g.assets.font(ui.FontSize)
	if face == nil {
		return
	}
	metrics := face.Metrics()
	ascent := metrics.Ascent.Ceil()
	lineHeight := metrics.Height.Ceil()
	for i, line := range strings.Split(ui.Text, "\n") {
		text.Draw(screen, line, face, ui.Pos.X, ui.Pos.Y+ascent+i*lineHeight, ui.Color)
	}
}
