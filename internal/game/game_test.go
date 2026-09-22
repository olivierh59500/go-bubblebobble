package game

import (
	"math"
	"reflect"
	"testing"

	"bubblebobble/resources"
)

func testLevel() Level {
	l := Level{Number: 1, Player: Cell{4, 25}, PowerSpawn: Cell{10, 25}, FoodSpawn: Cell{16, 25}, Enemies: []EnemySpawn{{Kind: ZenChan, Cell: Cell{24, 25}}}}
	for row := 0; row < Rows; row++ {
		tiles := make([]int, Columns)
		for col := range tiles {
			if col < 2 || col >= Columns-2 || row == 0 || row == Rows-1 {
				tiles[col] = 1
			}
		}
		l.Tiles = append(l.Tiles, tiles)
	}
	return l
}

func testGame(t *testing.T) *Game {
	t.Helper()
	g, err := New([]Level{testLevel()}, 1)
	if err != nil {
		t.Fatal(err)
	}
	g.State = Playing
	return g
}

func campaign(t *testing.T) []Level {
	t.Helper()
	l, err := LoadCampaign(resources.Files)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func TestCampaignDataAndSpawns(t *testing.T) {
	levels := campaign(t)
	tileCounts := []int{226, 214, 228, 252, 227, 238, 212, 220, 214, 278, 236, 254, 221, 274, 331, 424, 332, 310, 336, 280, 332, 356, 250, 370, 263}
	enemyCounts := []int{3, 4, 4, 6, 4, 4, 4, 4, 5, 7, 7, 6, 7, 7, 7, 7, 7, 6, 7, 5, 7, 4, 5, 7, 6}
	seen := map[EnemyKind]bool{}
	for i, l := range levels {
		n := 0
		for _, row := range l.Tiles {
			for _, tile := range row {
				if tile != 0 {
					n++
				}
			}
		}
		if n != tileCounts[i] || len(l.Enemies) != enemyCounts[i] {
			t.Errorf("round %d data changed: tiles=%d enemies=%d", i+1, n, len(l.Enemies))
		}
		g, err := New([]Level{l}, 1)
		if err != nil {
			t.Fatal(err)
		}
		if g.blocked(g.Player.X, g.Player.Y, ActorSize) {
			t.Errorf("round %d player embedded in a wall", i+1)
		}
		for _, e := range g.Enemies {
			seen[e.Kind] = true
			if g.blocked(e.X, e.Y, ActorSize) {
				t.Errorf("round %d %s embedded in a wall", i+1, e.Kind)
			}
		}
	}
	if len(seen) != 6 {
		t.Fatalf("found %d enemy types", len(seen))
	}
}

func TestInvalidLevelsRejected(t *testing.T) {
	for _, mutate := range []func(*Level){
		func(l *Level) { l.Tiles = l.Tiles[:25] }, func(l *Level) { l.Tiles[0] = l.Tiles[0][:31] },
		func(l *Level) { l.Enemies[0].Kind = "unknown" }, func(l *Level) { l.Enemies = nil },
		func(l *Level) { l.Player.Col = -1 }, func(l *Level) { l.Tiles[10][0] = 0 }, func(l *Level) { l.Special = "ice" },
	} {
		l := testLevel()
		mutate(&l)
		if _, err := New([]Level{l}, 1); err == nil {
			t.Fatal("accepted invalid level")
		}
	}
}

func TestPlatformsJumpLandingWallsAndWrap(t *testing.T) {
	g := testGame(t)
	for col := 5; col < 15; col++ {
		g.Level.Tiles[12][col] = 1
	}
	b := Body{X: 8 * Tile, Y: 12 * Tile, VY: -10}
	for i := 0; i < 10; i++ {
		g.moveActor(&b, 0, false)
	}
	if b.Y >= 12*Tile-ActorSize {
		t.Fatal("jump did not pass through platform")
	}
	b.VY = 80
	g.moveActor(&b, 0, false)
	if !b.Ground || b.Y != 12*Tile-ActorSize {
		t.Fatalf("missed platform on fast descent: %+v", b)
	}
	b = Body{X: 2 * Tile, Y: 500}
	g.moveActor(&b, -100, false)
	if b.X < 2*Tile {
		t.Fatal("crossed left wall")
	}
	for col := 2; col < Columns-2; col++ {
		g.Level.Tiles[Rows-1][col] = 0
	}
	b = Body{X: 100, Y: Height - 2, VY: 8}
	g.moveActor(&b, 0, false)
	if b.Y != Tile {
		t.Fatal("bottom opening did not wrap")
	}
}

func TestBubbleCapturePopAndCombo(t *testing.T) {
	g := testGame(t)
	g.Player.X, g.Player.Y = 100, 400
	g.Enemies[0].X, g.Enemies[0].Y = 170, 400
	g.shootBubble()
	for i := 0; i < 10; i++ {
		g.updateBubbles(Input{})
	}
	if g.Enemies[0].State != Trapped {
		t.Fatal("shot failed to trap enemy")
	}
	b := &g.Bubbles[0]
	b.Age = 30
	g.Player.X, g.Player.Y = b.X, b.Y
	g.updateBubbles(Input{})
	if g.Enemies[0].State != Defeated || g.Score != 100 {
		t.Fatalf("pop result: state=%v score=%d", g.Enemies[0].State, g.Score)
	}
	for i := 0; i < 60; i++ {
		g.updateDefeated()
	}
	if len(g.Items) != 2 {
		t.Fatalf("expected a bonus fruit and an enemy drop, got %d", len(g.Items))
	}
	for i := 0; i < 60; i++ {
		g.updateDefeated()
	}
	if len(g.Items) != 2 {
		t.Fatal("fruit was spawned twice")
	}
	g = testGame(t)
	g.Enemies = append(g.Enemies, Enemy{ID: 2, State: Trapped})
	g.Enemies[0].State = Trapped
	g.Bubbles = []Bubble{{Body: Body{X: 100, Y: 100}, Age: 30, EnemyID: g.Enemies[0].ID}, {Body: Body{X: 138, Y: 100}, Age: 30, EnemyID: 2}}
	g.popBubble(0)
	if g.Score != 300 || !g.Bubbles[1].Removed {
		t.Fatalf("chain pop failed: score=%d", g.Score)
	}
}

func TestBubbleEscapeAndBounce(t *testing.T) {
	g := testGame(t)
	e := &g.Enemies[0]
	e.State = Trapped
	g.Bubbles = []Bubble{{Body: Body{X: 300, Y: 100}, Age: 1500, CapturedAge: 1270, EnemyID: e.ID, Floating: true}}
	g.updateBubbles(Input{})
	if e.State != Active || !e.Angry || len(g.Bubbles) != 0 {
		t.Fatal("enemy did not escape angry")
	}
	g.Player.Body = Body{X: 200, Y: 200 - ActorSize, VY: 3}
	g.Bubbles = []Bubble{{Body: Body{X: 200, Y: 200}, Age: 50, Floating: true}}
	g.updateBubbles(Input{Jump: true})
	if g.Player.VY >= 0 || len(g.Bubbles) != 1 {
		t.Fatal("jumping should bounce without popping")
	}
}

func TestPauseFreezeAndHurry(t *testing.T) {
	g := testGame(t)
	g.Paused = true
	g.Freeze = 100
	before := g.Player
	for i := 0; i < 2000; i++ {
		g.Step(Input{Move: 1, Fire: true, Jump: true})
	}
	if g.Tick != 0 || g.LevelTicks != 0 || g.Freeze != 100 || g.Player != before {
		t.Fatal("pause advanced simulation")
	}
	g.Paused = false
	e := g.Enemies[0]
	g.Step(Input{Move: 1})
	if g.Enemies[0] != e || g.Player.X == before.X || g.Freeze != 99 {
		t.Fatal("clock should freeze enemies while player can move")
	}
	g.Freeze = 0
	g.LevelTicks = 30*TPS - 1
	g.Step(Input{})
	if !g.Hurry || !g.Enemies[0].Angry {
		t.Fatal("hurry up did not activate")
	}
}

func TestLivesRespawnAndGameOver(t *testing.T) {
	g := testGame(t)
	g.Player.Power.Shoes = true
	g.hurtPlayer()
	g.hurtPlayer()
	if g.Lives != 2 || g.Player.Dead != 90 {
		t.Fatal("one collision consumed multiple lives")
	}
	for i := 0; i < 90; i++ {
		g.Step(Input{})
	}
	if g.Player.Dead != 0 || g.Player.Invincible == 0 || g.Player.Power.Shoes {
		t.Fatal("invalid respawn")
	}
	g.hurtPlayer()
	if g.Lives != 2 {
		t.Fatal("invulnerability ignored")
	}
	g.Lives = 1
	g.Player.Invincible = 0
	g.hurtPlayer()
	for i := 0; i < 90; i++ {
		g.Step(Input{})
	}
	if g.State != GameOver || g.Lives != 0 {
		t.Fatalf("final life did not end run: %v %d", g.State, g.Lives)
	}
}

func TestEveryPowerUp(t *testing.T) {
	for _, power := range PowerUps {
		t.Run(string(power), func(t *testing.T) {
			g := testGame(t)
			transition := g.applyPower(power)
			switch power {
			case PinkCandy:
				if !g.Player.Power.Pink {
					t.Fatal("missing range")
				}
			case BlueCandy:
				if !g.Player.Power.Blue {
					t.Fatal("missing speed")
				}
			case YellowCandy:
				if !g.Player.Power.Yellow {
					t.Fatal("missing fire rate")
				}
			case Shoes:
				if !g.Player.Power.Shoes {
					t.Fatal("missing shoes")
				}
			case CrystalRing:
				if !g.Player.Power.Crystal {
					t.Fatal("missing ring")
				}
			case AmethystRing:
				if !g.Player.Power.Amethyst {
					t.Fatal("missing ring")
				}
			case RubyRing:
				if !g.Player.Power.Ruby {
					t.Fatal("missing ring")
				}
			case OrangeParasol, RedParasol, PurpleParasol:
				if !transition || g.State != Won {
					t.Fatal("warp beyond final round should win")
				}
			case Clock:
				if g.Freeze != 420 {
					t.Fatal("incorrect clock duration")
				}
			case Dynamite:
				if g.remainingEnemies() != 0 {
					t.Fatal("dynamite left enemies alive")
				}
			}
			if g.Score <= 0 {
				t.Fatal("no pickup score")
			}
		})
	}
}

func TestCounterThresholdsAndParasols(t *testing.T) {
	g := testGame(t)
	g.Counters.Blown = 36
	g.checkPowerSpawns()
	if len(g.Items) != 1 || g.Items[0].Power != PinkCandy || g.Counters.Blown != 1 {
		t.Fatal("threshold overshoot lost a bonus")
	}
	for _, tc := range []struct {
		water int
		power PowerUp
	}{{15, OrangeParasol}, {20, RedParasol}, {25, PurpleParasol}} {
		g.Items = nil
		g.Counters.Water = tc.water
		g.checkPowerSpawns()
		if len(g.Items) != 1 || g.Items[0].Power != tc.power {
			t.Fatalf("water %d failed to unlock %s", tc.water, tc.power)
		}
	}
}

func TestSpecialBubblesAndElementDamage(t *testing.T) {
	for _, element := range []Element{Water, Thunder, Fire} {
		t.Run(string(element), func(t *testing.T) {
			g := testGame(t)
			g.Level.Special = element
			g.Level.SpecialCount = 2
			for i := 1; i <= 1350; i++ {
				g.LevelTicks = i
				g.spawnSpecial()
			}
			if len(g.Bubbles) != 2 {
				t.Fatalf("spawned %d bubbles, want 2", len(g.Bubbles))
			}
			g.Bubbles[0].X, g.Bubbles[0].Y = g.Enemies[0].X, g.Enemies[0].Y
			g.releaseElement(&g.Bubbles[0])
			if element == Fire {
				g.Effects[0].Ground = true
			}
			g.updateEffects()
			if element == Water {
				if g.Enemies[0].State != Carried {
					t.Fatal("water did not carry enemy")
				}
				for i := 0; i < 3*TPS; i++ {
					g.updateEffects()
				}
			}
			if g.Enemies[0].State != Defeated {
				t.Fatal("element did not defeat enemy")
			}
		})
	}
}

func TestSpecialScheduleSurvivesClockAndFireUnlocksDynamite(t *testing.T) {
	g := testGame(t)
	g.Level.Special, g.Level.SpecialCount = Water, 3
	g.LevelTicks = 1
	g.spawnSpecial()
	g.Freeze = 10
	for i := 0; i < 10; i++ {
		g.spawnSpecial()
	}
	if len(g.Bubbles) != 1 {
		t.Fatal("clock duplicated a scheduled special bubble")
	}
	g.Freeze = 0
	g.spawnSpecial()
	if len(g.Bubbles) != 1 {
		t.Fatal("same tick spawned another bubble")
	}
	g.LevelTicks = 451
	g.spawnSpecial()
	if len(g.Bubbles) != 2 {
		t.Fatal("next scheduled special bubble missing")
	}
	for _, l := range campaign(t) {
		if l.Special == Fire && l.SpecialCount < 13 {
			t.Fatal("fire budget cannot unlock dynamite")
		}
	}
}

func TestEnemyProjectilesAndCollision(t *testing.T) {
	for _, kind := range []EnemyKind{Mighta, Invader} {
		t.Run(string(kind), func(t *testing.T) {
			g := testGame(t)
			e := &g.Enemies[0]
			e.Kind = kind
			e.Cooldown = 0
			e.X = 300
			e.Y = g.Player.Y
			g.updateEnemies()
			if len(g.Projectiles) != 1 {
				t.Fatal("enemy did not shoot")
			}
			p := &g.Projectiles[0]
			if kind == Mighta && (p.Laser || p.VX == 0 || p.VY != 0) {
				t.Fatal("Mighta must throw horizontally")
			}
			if kind == Invader && (!p.Laser || p.VX != 0 || p.VY <= 0) {
				t.Fatal("Invader must shoot downwards")
			}
			p.X, p.Y = g.Player.X+8, g.Player.Y+8
			g.checkPlayerHit()
			if g.Lives != 2 || !p.Removed {
				t.Fatal("projectile did not damage player exactly once")
			}
			g.checkPlayerHit()
			if g.Lives != 2 {
				t.Fatal("projectile dealt repeated damage")
			}
		})
	}
}

func TestCampaignProgressionEndsAtRound25(t *testing.T) {
	g, err := New(campaign(t), 1)
	if err != nil {
		t.Fatal(err)
	}
	for round := 1; round <= 25; round++ {
		if g.Level.Number != round {
			t.Fatalf("expected round %d got %d", round, g.Level.Number)
		}
		g.State = Playing
		for i := range g.Enemies {
			g.defeat(&g.Enemies[i], 1)
		}
		g.Step(Input{})
		if g.State != Clearing {
			t.Fatal("round did not clear")
		}
		for i := 0; i < 7*TPS; i++ {
			g.Step(Input{})
		}
	}
	if g.State != Won {
		t.Fatalf("campaign state %v, want victory", g.State)
	}
}

func TestSeededSimulationAcrossCampaign(t *testing.T) {
	for _, level := range campaign(t) {
		a, err := New([]Level{level}, 42)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := New([]Level{level}, 42)
		for tick := 0; tick < 3000; tick++ {
			in := Input{Move: 1 - 2*(tick/140%2), Jump: tick%80 < 30, Fire: tick%5 != 0}
			a.Step(in)
			b.Step(in)
			if math.IsNaN(a.Player.X) || a.Player.X < 2*Tile || a.Player.X > Width-2*Tile-ActorSize {
				t.Fatalf("round %d player escaped arena", level.Number)
			}
			if len(a.Bubbles) > 130 || len(a.Effects) > 200 || len(a.Projectiles) > 100 {
				t.Fatal("unbounded entity growth")
			}
		}
		if a.Score != b.Score || a.State != b.State || !reflect.DeepEqual(a.Enemies, b.Enemies) || a.Player != b.Player {
			t.Fatal("simulation is not deterministic")
		}
	}
}
