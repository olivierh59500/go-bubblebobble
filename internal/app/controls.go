package app

import (
	"image"
	"image/color"
	"math"

	"bubblebobble/internal/game"
	"bubblebobble/internal/platform"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type circle struct{ X, Y, R float64 }

func (c circle) contains(x, y int) bool { return math.Hypot(float64(x)-c.X, float64(y)-c.Y) <= c.R }

type controlsLayout struct {
	Pad, Fire, Pause      circle
	Width, Height, SceneX int
}
type controlsState struct{ Left, Right, Jump, Fire bool }

func makeControls(w, h int, enabled bool) controlsLayout {
	l := controlsLayout{Width: screenWidth, Height: screenHeight}
	if !enabled {
		return l
	}
	if h > 0 && float64(w)/float64(h) >= 1.5 {
		l.Width = max(1168, min(1920, (w*screenHeight+h-1)/h))
		l.SceneX = (l.Width - screenWidth) / 2
		left := float64(l.SceneX) / 2
		l.Pad = circle{left, 520, 94}
		l.Fire = circle{float64(l.Width) - left, 530, 70}
		l.Pause = circle{float64(l.Width) - left, 110, 34}
	} else {
		l.Height = screenHeight + 224
		l.Pad = circle{152, 832, 88}
		l.Fire = circle{616, 832, 70}
		l.Pause = circle{384, 832, 32}
	}
	return l
}

func (s *controlsState) press(l controlsLayout, x, y int) {
	if l.Pad.contains(x, y) {
		dx, dy := float64(x)-l.Pad.X, float64(y)-l.Pad.Y
		s.Left = s.Left || dx < -20
		s.Right = s.Right || dx > 20
		s.Jump = s.Jump || dy < -25
	}
	s.Fire = s.Fire || l.Fire.contains(x, y)
}

func (a *App) EnableTouch(enabled bool) { a.touchEnabled = enabled }
func (a *App) EnableMobile(bridge *platform.Bridge) {
	a.mobile = true
	a.bridge = bridge
	a.touchEnabled = true
}

func (a *App) readTouch() {
	a.touch = controlsState{}
	a.touchIDs = ebiten.AppendTouchIDs(a.touchIDs[:0])
	if len(a.touchIDs) > 0 && !a.touchEnabled {
		a.touchEnabled = true
	}
	if a.bridge.InputReset.Load() != a.inputSerial {
		a.inputSerial = a.bridge.InputReset.Load()
		a.ignoreTouch = true
	}
	if a.ignoreTouch {
		if len(a.touchIDs) == 0 && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			a.ignoreTouch = false
		}
		return
	}
	for _, id := range a.touchIDs {
		x, y := ebiten.TouchPosition(id)
		a.touch.press(a.controls, x, y)
	}
	if a.touchEnabled && len(a.touchIDs) == 0 && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		a.touch.press(a.controls, x, y)
	}
}

func (a *App) controlInput() game.Input {
	in := playInput()
	if a.ignoreTouch {
		return game.Input{}
	}
	if a.touch.Left && !a.touch.Right {
		in.Move = -1
	}
	if a.touch.Right && !a.touch.Left {
		in.Move = 1
	}
	in.Jump = in.Jump || a.touch.Jump
	in.Fire = in.Fire || a.touch.Fire
	return in
}

func (a *App) pointerPressed() (x, y int, ok bool) {
	if a.ignoreTouch {
		return 0, 0, false
	}
	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		x, y = ebiten.TouchPosition(id)
		return x - a.controls.SceneX, y, true
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y = ebiten.CursorPosition()
		return x - a.controls.SceneX, y, true
	}
	return 0, 0, false
}

func (a *App) backPressed() bool {
	if a.platformBack {
		return true
	}
	if a.touchEnabled {
		if x, y, ok := a.pointerPressed(); ok && a.controls.Pause.contains(x+a.controls.SceneX, y) {
			return true
		}
	}
	return backPressed()
}

func (a *App) drawControls(dst *ebiten.Image) {
	if !a.touchEnabled || a.page != pagePlay {
		return
	}
	l := a.controls
	draw := func(c circle, active bool, label string) {
		bg, fg := panel, green
		if active {
			bg, fg = green, ink
		}
		vector.FillCircle(dst, float32(c.X), float32(c.Y), float32(c.R), bg, true)
		vector.StrokeCircle(dst, float32(c.X), float32(c.Y), float32(c.R), 2, green, true)
		if label != "" {
			a.label(dst, label, 14, c.X-float64(len(label))*7, c.Y-7, fg)
		}
	}
	draw(l.Pad, false, "")
	arrow := func(dx, dy float64, direction int, active bool) {
		c := color.Color(muted)
		if active {
			c = green
		}
		x, y := float32(l.Pad.X+dx), float32(l.Pad.Y+dy)
		var path vector.Path
		if direction == 0 {
			path.MoveTo(x-12, y+7)
			path.LineTo(x, y-7)
			path.LineTo(x+12, y+7)
		} else {
			d := float32(direction)
			path.MoveTo(x-7*d, y-12)
			path.LineTo(x+7*d, y)
			path.LineTo(x-7*d, y+12)
		}
		options := &vector.StrokeOptions{Width: 4, LineJoin: vector.LineJoinRound}
		drawOptions := &vector.DrawPathOptions{AntiAlias: true}
		drawOptions.ColorScale.ScaleWithColor(c)
		vector.StrokePath(dst, &path, options, drawOptions)
	}
	arrow(-55, 0, -1, a.touch.Left)
	arrow(55, 0, 1, a.touch.Right)
	arrow(0, -52, 0, a.touch.Jump)
	a.label(dst, "JUMP", 11, l.Pad.X-22, l.Pad.Y+l.Pad.R+15, muted)
	draw(l.Fire, a.touch.Fire, "BUBBLE")
	draw(l.Pause, false, "II")
}

func (a *App) Layout(w, h int) (int, int) {
	a.controls = makeControls(w, h, a.touchEnabled)
	return a.controls.Width, a.controls.Height
}

func (a *App) scenePoint() image.Point {
	x, y := ebiten.CursorPosition()
	return image.Pt(x-a.controls.SceneX, y)
}
