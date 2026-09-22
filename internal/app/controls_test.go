package app

import (
	"bubblebobble/internal/game"
	"testing"
)

func TestVirtualControlsMultitouchAndLayout(t *testing.T) {
	for _, size := range [][2]int{{2424, 1080}, {1920, 1080}, {1080, 2424}, {768, 720}, {6000, 800}} {
		l := makeControls(size[0], size[1], true)
		if l.Width < screenWidth || l.Width > 1920 || l.SceneX < 0 {
			t.Fatalf("invalid layout: %+v", l)
		}
		for _, c := range []circle{l.Pad, l.Fire, l.Pause} {
			if c.X-c.R < 0 || c.Y-c.R < 0 || c.X+c.R > float64(l.Width) || c.Y+c.R > float64(l.Height) {
				t.Fatal("controls outside screen")
			}
		}
		var s controlsState
		s.press(l, int(l.Pad.X+45), int(l.Pad.Y-40))
		s.press(l, int(l.Fire.X), int(l.Fire.Y))
		if !s.Right || !s.Jump || !s.Fire || s.Left {
			t.Fatal("direction, jump and fire did not combine")
		}
		s = controlsState{}
		s.press(l, int(l.Pad.X-45), int(l.Pad.Y))
		if !s.Left || s.Fire || s.Jump || s.Right {
			t.Fatal("released controls stuck")
		}
	}
	if l := makeControls(2424, 1080, false); l.Width != 768 || l.Height != 720 {
		t.Fatal("desktop scene changed without touch mode")
	}
}

func TestRemoteRenderInterpolationDoesNotExtrapolate(t *testing.T) {
	a, b := game.Body{X: 100, Y: 200}, game.Body{X: 130, Y: 230}
	if got := blendBody(a, b, 0.5); got.X != 115 || got.Y != 215 {
		t.Fatal("incorrect interpolated position")
	}
	if blendBody(a, b, 2) != b {
		t.Fatal("extrapolated beyond authoritative state")
	}
	b.Y = 600
	if blendBody(a, b, 0.5) != b {
		t.Fatal("respawn was interpolated through the level")
	}
}
