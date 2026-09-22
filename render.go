package main

import (
	"fmt"
	"image/color"
	"image/png"
	"os"
	"strings"

	"bubblebobble/internal/game"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var (
	ink    = color.RGBA{8, 12, 27, 255}
	panel  = color.RGBA{19, 29, 49, 255}
	white  = color.RGBA{235, 245, 255, 255}
	muted  = color.RGBA{130, 159, 177, 255}
	green  = color.RGBA{112, 240, 137, 255}
	pink   = color.RGBA{255, 108, 177, 255}
	yellow = color.RGBA{255, 220, 106, 255}
)

func (a *app) Draw(screen *ebiten.Image) {
	screen.Fill(ink)
	switch a.page {
	case pageTitle, pageMenu:
		a.drawMenu(screen)
	case pagePlay:
		a.drawMatch(screen)
	case pageResult:
		a.drawMatch(screen)
		a.drawResult(screen)
	case pageProfiles:
		a.drawProfiles(screen)
	case pageNewProfile:
		a.drawNewProfile(screen)
	case pageScores:
		a.drawScores(screen)
	case pageSettings:
		a.drawSettings(screen)
	case pageHelp:
		a.drawHelp(screen)
	case pageCredits:
		a.drawCredits(screen)
	case pageEditor:
		a.drawEditor(screen)
	}
	for i, b := range a.buttons() {
		a.drawButton(screen, b, i == a.selected)
	}
	if a.statusTicks > 0 {
		box(screen, 0, 680, screenWidth, 40, panel)
		a.center(screen, strings.ToUpper(a.status), 12, 700, yellow)
	}
	if a.smokeFrames > 0 && a.frame >= a.smokeFrames && a.capturePath != "" && !a.captured {
		f, err := os.Create(a.capturePath)
		if err == nil {
			err = png.Encode(f, screen)
			if closeErr := f.Close(); err == nil {
				err = closeErr
			}
		}
		a.captureErr = err
		a.captured = true
	}
}

func box(dst *ebiten.Image, x, y, w, h float64, c color.Color) {
	vector.FillRect(dst, float32(x), float32(y), float32(w), float32(h), c, false)
}

func drawImage(dst, src *ebiten.Image, x, y, w, h float64, flip bool) {
	if src == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	sx, sy := w/float64(src.Bounds().Dx()), h/float64(src.Bounds().Dy())
	if flip {
		op.GeoM.Scale(-sx, sy)
		op.GeoM.Translate(x+w, y)
	} else {
		op.GeoM.Scale(sx, sy)
		op.GeoM.Translate(x, y)
	}
	dst.DrawImage(src, op)
}

func (a *app) sprite(dst *ebiten.Image, name string, tick int, x, y, size float64, dir int) {
	drawImage(dst, a.art.frame(name, tick), x, y, size, size, dir > 0)
}

func (a *app) label(dst *ebiten.Image, s string, size, x, y float64, c color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	op.LineSpacing = size * 1.65
	text.Draw(dst, s, &text.GoTextFace{Source: a.art.font, Size: size}, op)
}

func (a *app) center(dst *ebiten.Image, s string, size, y float64, c color.Color) {
	face := &text.GoTextFace{Source: a.art.font, Size: size}
	w, _ := text.Measure(s, face, size*1.65)
	if w > screenWidth-40 {
		size *= float64(screenWidth-40) / w
		w = screenWidth - 40
	}
	a.label(dst, s, size, (screenWidth-w)/2, y, c)
}

func (a *app) heading(dst *ebiten.Image, title, subtitle string) {
	box(dst, 24, 30, 36, 4, green)
	a.label(dst, "BUBBLE BOBBLE", 11, 76, 26, green)
	a.center(dst, title, 28, 65, white)
	if subtitle != "" {
		a.center(dst, subtitle, 11, 110, muted)
	}
}

func (a *app) drawButton(dst *ebiten.Image, b button, selected bool) {
	x, y, w, h := float64(b.bounds.Min.X), float64(b.bounds.Min.Y), float64(b.bounds.Dx()), float64(b.bounds.Dy())
	bg, fg := panel, white
	if selected {
		bg, fg = green, ink
	}
	box(dst, x, y, w, h, bg)
	if selected {
		box(dst, x-5, y, 2, h, green)
	}
	size := 14.0
	face := &text.GoTextFace{Source: a.art.font, Size: size}
	tw, _ := text.Measure(b.label, face, size)
	if tw > w-20 {
		size *= (w - 20) / tw
		tw = w - 20
	}
	a.label(dst, b.label, size, x+(w-tw)/2, y+(h-size)/2, fg)
}

func (a *app) drawMenu(dst *ebiten.Image) {
	for i := 0; i < 18; i++ {
		x := float64((i*137 + 39) % screenWidth)
		y := float64((i*91 - a.frame/2) % screenHeight)
		if y < 0 {
			y += screenHeight
		}
		vector.StrokeCircle(dst, float32(x), float32(y), float32(8+i%4*5), 1, color.RGBA{35, 65, 85, 255}, true)
	}
	drawImage(dst, a.art.logo, 209, 26, 350, 252, false)
	profile := a.store.Profile()
	a.center(dst, fmt.Sprintf("%s  /  BEST %07d", profile.Name, profile.HighScore), 12, 294, green)
	a.sprite(dst, "green/walk", a.frame, 80, 550, 96, 1)
	a.sprite(dst, "blue/walk", a.frame, 592, 550, 96, -1)
	if a.page == pageTitle {
		a.center(dst, "25 ROUNDS. ONE BUBBLY ADVENTURE.", 16, 360, white)
		if a.frame%90 < 65 {
			a.center(dst, "PRESS ENTER TO START", 20, 438, yellow)
		}
	}
	a.center(dst, "ARROWS TO SELECT   ENTER TO CONFIRM", 10, 678, muted)
}

func (a *app) drawMatch(dst *ebiten.Image) {
	g := a.match
	const offset = 48
	box(dst, 0, offset, game.Width, game.Height, color.Black)
	for row, tiles := range g.Level.Tiles {
		for col, tile := range tiles {
			if tile != 0 {
				drawImage(dst, a.art.tiles[tile], float64(col*game.Tile), float64(row*game.Tile+offset), game.Tile, game.Tile, false)
			}
		}
	}
	for _, item := range g.Items {
		if item.Age > 600 && item.Age%12 < 6 {
			continue
		}
		name := string(item.Power)
		if name == "" {
			name = fmt.Sprintf("food/%d", item.Food)
		}
		a.sprite(dst, name, 0, item.X, item.Y+offset, 32, -1)
	}
	for _, e := range g.Enemies {
		if e.State == game.Trapped || e.State == game.Defeated && e.DeadTicks == 0 {
			continue
		}
		state := "walk"
		if e.Angry {
			state = "angry"
		}
		if e.Kind == game.Mighta && e.Shoot > 0 {
			state = "shoot"
			if e.Angry {
				state = "shoot-angry"
			}
		}
		if e.State == game.Defeated || e.State == game.Carried {
			state = "dead"
		}
		a.sprite(dst, string(e.Kind)+"/"+state, e.Age, e.X, e.Y+offset, 48, e.Dir)
	}
	for _, b := range g.Bubbles {
		name := "bubble"
		tick := min(b.Age, 30)
		if b.Element != "" {
			name = string(b.Element) + "-bubble"
			tick = b.Age
		}
		if b.Age > 450 && b.EnemyID == 0 && b.Element == "" {
			name = "bubble-warning"
			tick = b.Age
		}
		if b.EnemyID != 0 {
			for _, e := range g.Enemies {
				if e.ID == b.EnemyID {
					state := "bubble"
					if b.CapturedAge > 1000 {
						state = "warning"
					}
					if b.CapturedAge > 1200 {
						state = "escape"
					}
					name = string(e.Kind) + "/" + state
					tick = b.Age
					break
				}
			}
		}
		a.sprite(dst, name, tick, b.X, b.Y+offset, 44, -1)
	}
	for _, p := range g.Projectiles {
		name, w, h := "boulder", 24.0, 24.0
		if p.Laser {
			name, w, h = "laser", 16, 28
		}
		drawImage(dst, a.art.frame(name, p.Age), p.X, p.Y+offset, w, h, false)
	}
	for _, e := range g.Effects {
		name := string(e.Element)
		if e.Element == game.Fire && e.Ground {
			name = "flame"
		}
		drawImage(dst, a.art.frame(name, e.Age), e.X, e.Y+offset, 40, 32, e.Dir > 0)
	}
	p := g.Player
	if p.Invincible == 0 || p.Invincible%12 < 6 {
		state, tick := "walk", p.WalkFrame
		if !p.Ground {
			state, tick = "jump", g.Tick
		}
		if p.Shoot > 0 {
			state, tick = "shoot", 12-p.Shoot
		}
		if p.Dead > 0 {
			state, tick = "dead", 90-p.Dead
		}
		a.sprite(dst, a.store.Profile().Avatar+"/"+state, tick, p.X, p.Y+offset, 48, p.Dir)
	}
	for _, p := range g.Particles {
		if p.Text != "" {
			a.label(dst, p.Text, 12, p.X, p.Y+offset, yellow)
		} else {
			a.sprite(dst, "pop", 0, p.X, p.Y+offset, 44, -1)
		}
	}
	box(dst, 0, 0, screenWidth, 48, ink)
	a.label(dst, a.store.Profile().Name, 10, 24, 7, green)
	a.label(dst, fmt.Sprintf("%07d", g.Score), 20, 24, 22, white)
	a.center(dst, fmt.Sprintf("ROUND %02d", g.Level.Number), 18, 18, yellow)
	a.label(dst, "HIGH SCORE", 10, 590, 7, pink)
	a.label(dst, fmt.Sprintf("%07d", max(g.Score, a.store.Profile().HighScore)), 20, 590, 22, white)
	box(dst, 0, 672, screenWidth, 48, ink)
	a.sprite(dst, a.store.Profile().Avatar+"/walk", 0, 20, 680, 28, -1)
	a.label(dst, fmt.Sprintf("x%d", g.Lives), 14, 54, 687, white)
	a.label(dst, fmt.Sprintf("ENEMIES %02d", g.RemainingEnemies()), 11, 140, 690, muted)
	a.label(dst, "ESC PAUSE  M SOUND", 11, 530, 690, muted)
	x := 320.0
	for _, power := range []struct {
		active bool
		name   string
	}{{p.Power.Pink, "pink_candy"}, {p.Power.Blue, "blue_candy"}, {p.Power.Yellow, "yellow_candy"}, {p.Power.Shoes, "shoes"}, {p.Power.Crystal, "crystal_ring"}, {p.Power.Amethyst, "amethyst_ring"}, {p.Power.Ruby, "ruby_ring"}} {
		if power.active {
			a.sprite(dst, power.name, 0, x, 681, 24, -1)
			x += 26
		}
	}
	if g.State == game.Ready {
		a.banner(dst, fmt.Sprintf("ROUND %02d", g.Level.Number), "GET READY", 240)
	}
	if g.State == game.Clearing {
		a.banner(dst, "ROUND CLEAR!", fmt.Sprintf("COLLECT YOUR BONUS  %d", (g.Countdown+59)/60), 84)
	}
	if g.Hurry && g.State == game.Playing && g.LevelTicks < 33*60 && g.Tick%30 < 24 {
		a.banner(dst, "HURRY UP!", "", 240)
	}
	if g.Freeze > 0 {
		a.center(dst, fmt.Sprintf("TIME STOP %d", (g.Freeze+59)/60), 14, 86, yellow)
	}
	if g.Paused {
		box(dst, 0, 0, screenWidth, screenHeight, color.RGBA{0, 0, 0, 195})
		box(dst, 124, 210, 520, 310, ink)
		a.center(dst, "PAUSED", 32, 252, green)
	}
}

func (a *app) banner(dst *ebiten.Image, title, subtitle string, y float64) {
	box(dst, 108, y, 552, 82, color.RGBA{8, 12, 27, 230})
	a.center(dst, title, 24, y+15, yellow)
	if subtitle != "" {
		a.center(dst, subtitle, 12, y+53, white)
	}
}

func (a *app) drawResult(dst *ebiten.Image) {
	box(dst, 0, 0, screenWidth, screenHeight, color.RGBA{0, 0, 0, 220})
	title, sub := "GAME OVER", "A NEW ADVENTURE IS ONE BUBBLE AWAY."
	if a.match.State == game.Won {
		title, sub = "YOU WIN!", "ALL 25 ROUNDS CLEARED. WELL PLAYED!"
	}
	if a.match.Custom {
		sub = "CUSTOM LEVEL PLAY TEST COMPLETE"
	}
	a.center(dst, title, 44, 168, green)
	a.center(dst, sub, 13, 252, white)
	a.center(dst, fmt.Sprintf("SCORE  %07d", a.match.Score), 26, 312, yellow)
	p := a.store.Profile()
	if !a.match.Custom {
		a.center(dst, fmt.Sprintf("BEST %07d  /  LEVEL %d  /  XP %d", p.HighScore, p.XP/100+1, p.XP), 12, 368, muted)
	}
}

func (a *app) drawProfiles(dst *ebiten.Image) {
	a.heading(dst, "PLAYER PROFILES", "SELECT A PLAYER OR CREATE A NEW PROFILE")
	for i, p := range a.store.Data.Profiles {
		a.sprite(dst, p.Avatar+"/walk", a.frame, 114, float64(132+i*46), 36, 1)
	}
	a.center(dst, "PROGRESS AND HIGH SCORES ARE SAVED AUTOMATICALLY", 10, 680, muted)
}

func (a *app) drawNewProfile(dst *ebiten.Image) {
	a.heading(dst, "NEW PLAYER", "YOUR NAME: UP TO 16 LETTERS OR NUMBERS")
	box(dst, 164, 254, 440, 70, panel)
	name := a.name
	if a.frame%60 < 30 {
		name += "_"
	}
	a.center(dst, name, 22, 278, green)
}

func (a *app) drawScores(dst *ebiten.Image) {
	a.heading(dst, "HIGH SCORES", "LOCAL PLAYERS / ALL CAMPAIGN RUNS")
	a.label(dst, "PLAYER", 12, 104, 166, muted)
	a.label(dst, "SCORE", 12, 354, 166, muted)
	a.label(dst, "ROUND", 12, 500, 166, muted)
	a.label(dst, "W / L", 12, 604, 166, muted)
	for i, p := range a.store.Leaderboard() {
		y := float64(202 + i*45)
		box(dst, 80, y-8, 608, 38, panel)
		a.label(dst, p.Name, 12, 104, y, white)
		a.label(dst, fmt.Sprintf("%07d", p.HighScore), 12, 354, y, yellow)
		a.label(dst, fmt.Sprintf("%02d", p.BestRound), 12, 512, y, green)
		a.label(dst, fmt.Sprintf("%d/%d", p.Won, p.Lost), 12, 604, y, white)
	}
}

func (a *app) drawSettings(dst *ebiten.Image) {
	a.heading(dst, "SETTINGS", "ENTER TO CHANGE / LEFT AND RIGHT TO ADJUST")
	p := a.store.Profile()
	a.sprite(dst, p.Avatar+"/walk", a.frame, 336, 476, 96, -1)
	a.center(dst, fmt.Sprintf("%s  /  PLAYER LEVEL %d", p.Name, p.XP/100+1), 12, 592, green)
	a.center(dst, fmt.Sprintf("PLAYED %d  WON %d  LOST %d  XP %d", p.Played, p.Won, p.Lost, p.XP), 10, 622, muted)
}

func (a *app) drawHelp(dst *ebiten.Image) {
	a.heading(dst, "HOW TO PLAY", "TRAP THE ENEMIES. POP THE BUBBLES. COLLECT THE FRUIT.")
	lines := []struct{ title, detail string }{
		{"MOVE", "LEFT / RIGHT OR A / D"},
		{"JUMP", "UP OR W - HOLD TO BOUNCE ON BUBBLES"},
		{"BLOW BUBBLES", "SPACE OR CTRL - TOUCH A BUBBLE TO POP IT"},
		{"TAKE A BREAK", "ESC TO PAUSE / F11 FULLSCREEN / M MUTE"},
		{"GAMEPAD", "STICK OR D-PAD / A JUMP / X FIRE / START PAUSE"},
		{"BONUSES", "CANDIES, SHOES, RINGS, PARASOLS, CLOCK, DYNAMITE"},
		{"ELEMENTS", "WATER FLOWS, LIGHTNING STRIKES, FIRE SPREADS"},
		{"SURVIVE", "3 LIVES / ENEMIES GET ANGRY AFTER 30 SECONDS"},
	}
	for i, line := range lines {
		y := float64(158 + i*54)
		a.label(dst, line.title, 13, 52, y, green)
		a.label(dst, line.detail, 11, 52, y+23, white)
	}
}

func (a *app) drawCredits(dst *ebiten.Image) {
	a.heading(dst, "CREDITS", "A FAN REMAKE OF THE ARCADE CLASSIC")
	for i, line := range []string{"ORIGINAL GAME AND CHARACTERS", "TAITO", "", "GAME ADAPTATION AND LEVEL DESIGN", "MALAKH SOFTWARE - 2026", "", "GO EDITION POWERED BY EBITENGINE", "", "SOFTWARE LICENSE: MIT", "ARTWORK AND AUDIO BELONG TO THEIR OWNERS"} {
		c := white
		if i == 1 || i == 4 || i == 6 {
			c = green
		}
		a.center(dst, line, 13, float64(176+i*36), c)
	}
}

func (a *app) drawEditor(dst *ebiten.Image) {
	a.heading(dst, "LEVEL EDITOR", "")
	e := &a.editor
	box(dst, editorX, editorY, game.Columns*editorTile, game.Rows*editorTile, color.Black)
	for row, tiles := range e.level.Tiles {
		for col, tile := range tiles {
			x, y := float64(editorX+col*editorTile), float64(editorY+row*editorTile)
			if tile != 0 {
				drawImage(dst, a.art.tiles[tile], x, y, editorTile, editorTile, false)
			} else {
				box(dst, x, y, editorTile, 1, panel)
				box(dst, x, y, 1, editorTile, panel)
			}
		}
	}
	drawSpawn := func(c game.Cell, name string, marker string) {
		x, y := c.Position(game.ActorSize)
		x, y = editorX+x*2/3, editorY+y*2/3
		a.sprite(dst, name, a.frame, x, y, 30, -1)
		if marker != "" {
			a.label(dst, marker, 9, x, y-9, yellow)
		}
	}
	for _, enemy := range e.level.Enemies {
		drawSpawn(enemy.Cell, string(enemy.Kind)+"/walk", "")
	}
	drawSpawn(e.level.Player, a.store.Profile().Avatar+"/walk", "P")
	drawSpawn(e.level.FoodSpawn, "food/1", "F")
	drawSpawn(e.level.PowerSpawn, "pink_candy", "B")
	x, y := ebiten.CursorPosition()
	if x >= editorX+2*editorTile && x < editorX+(game.Columns-2)*editorTile && y >= editorY && y < editorY+game.Rows*editorTile {
		x = editorX + (x-editorX)/editorTile*editorTile
		y = editorY + (y-editorY)/editorTile*editorTile
		vector.StrokeRect(dst, float32(x), float32(y), editorTile, editorTile, 2, green, false)
	}
	a.label(dst, "LEFT: DRAW   RIGHT: ERASE   CTRL-Z: UNDO", 10, 24, 572, muted)
	a.label(dst, fmt.Sprintf("ENEMIES %02d / 32", len(e.level.Enemies)), 10, 554, 550, green)
	a.center(dst, "F5 PLAY TEST / ESC RETURN / P PLAYER / F FOOD / B BONUS", 10, 693, muted)
}
