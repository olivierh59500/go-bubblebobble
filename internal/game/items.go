package game

type PowerUp string

const (
	PinkCandy     PowerUp = "pink_candy"
	BlueCandy     PowerUp = "blue_candy"
	YellowCandy   PowerUp = "yellow_candy"
	Shoes         PowerUp = "shoes"
	OrangeParasol PowerUp = "orange_parasol"
	RedParasol    PowerUp = "red_parasol"
	PurpleParasol PowerUp = "purple_parasol"
	CrystalRing   PowerUp = "crystal_ring"
	AmethystRing  PowerUp = "amethyst_ring"
	RubyRing      PowerUp = "ruby_ring"
	Dynamite      PowerUp = "dynamite"
	Clock         PowerUp = "clock"
)

var PowerUps = []PowerUp{PinkCandy, BlueCandy, YellowCandy, Shoes, OrangeParasol, RedParasol, PurpleParasol, CrystalRing, AmethystRing, RubyRing, Dynamite, Clock}

type Powers struct {
	Pink, Blue, Yellow, Shoes, Crystal, Amethyst, Ruby bool
	WalkPoints                                         float64
}

type Counters struct {
	Blown, Popped, Jumps, Water, Thunder, Fire int
	Pink, Blue, Yellow                         int
	Walked                                     float64
	waterStage                                 int
	pending                                    []PowerUp
}

type Item struct {
	Body
	Stationary bool
	Power      PowerUp
	Food, Age  int
}

func FoodScore(level int) int {
	if level <= 10 {
		return max(1, level) * 10
	}
	return 100 + (min(level, 25)-10)*50
}

func (g *Game) spawnFood(x, y float64) {
	x = max(float64(2*Tile), min(float64(Width-2*Tile-32), x))
	y = max(float64(Tile), min(float64(Height-2*Tile), y))
	g.Items = append(g.Items, Item{Body: Body{X: x, Y: y}, Food: g.Level.Number})
}

func (g *Game) checkPowerSpawns() {
	c := &g.Counters
	for _, rule := range []struct {
		count     *int
		threshold int
		power     PowerUp
	}{
		{&c.Blown, 35, PinkCandy}, {&c.Popped, 35, BlueCandy}, {&c.Jumps, 35, YellowCandy},
		{&c.Thunder, 12, Clock}, {&c.Fire, 13, Dynamite}, {&c.Blue, 3, CrystalRing}, {&c.Yellow, 3, AmethystRing}, {&c.Pink, 3, RubyRing},
	} {
		for *rule.count >= rule.threshold {
			*rule.count -= rule.threshold
			c.pending = append(c.pending, rule.power)
		}
	}
	if c.Walked >= Width*15 {
		c.Walked -= Width * 15
		c.pending = append(c.pending, Shoes)
	}
	// Preserve water progress so all three parasols remain reachable.
	if c.Water >= 15 && c.waterStage == 0 {
		c.pending = append(c.pending, OrangeParasol)
		c.waterStage = 1
	}
	if c.Water >= 20 && c.waterStage == 1 {
		c.pending = append(c.pending, RedParasol)
		c.waterStage = 2
	}
	if c.Water >= 25 && c.waterStage == 2 {
		c.pending = append(c.pending, PurpleParasol)
		c.Water -= 25
		c.waterStage = 0
	}
	if len(c.pending) == 0 {
		return
	}
	for _, item := range g.Items {
		if item.Power != "" {
			return
		}
	}
	x, y := g.Level.PowerSpawn.Position(32)
	g.Items = append(g.Items, Item{Body: Body{X: x, Y: y}, Power: c.pending[0]})
	c.pending = c.pending[1:]
}

func (g *Game) updateItems() bool {
	kept := g.Items[:0]
	for _, item := range g.Items {
		item.Age++
		if item.Age > 12*TPS {
			continue
		}
		if item.Power == "" && !item.Stationary {
			oldFeet := item.Y + 32
			item.VY = min(6, item.VY+0.3)
			newFeet := oldFeet + item.VY
			row := int(oldFeet / Tile)
			if float64(row*Tile) < oldFeet {
				row++
			}
			if float64(row*Tile) <= newFeet && (g.Level.Solid(int(item.X/Tile), row) || g.Level.Solid(int((item.X+31)/Tile), row)) {
				item.Y = float64(row*Tile - 32)
				item.VY = 0
			} else {
				item.Y += item.VY
			}
			if item.Y > Height {
				item.X, item.Y = g.Level.FoodSpawn.Position(32)
				item.VY = 0
			}
		}
		collected := false
		for player := range g.Players {
			p := g.Players[player]
			if !p.Alive() || !intersects(p.X, p.Y, ActorSize, ActorSize, item.X, item.Y, 32, 32) {
				continue
			}
			if item.Power != "" {
				if g.applyPower(item.Power, player) {
					return true
				}
			} else {
				g.addScore(FoodScore(item.Food), item.X, item.Y)
				g.Sounds = append(g.Sounds, "food")
			}
			collected = true
			break
		}
		if collected {
			continue
		}

		kept = append(kept, item)
	}
	g.Items = kept
	return false
}

func (g *Game) applyPower(power PowerUp, player int) bool {
	p := &g.Players[player].Power
	points, warp := 100, 0
	switch power {
	case PinkCandy:
		p.Pink = true
		g.Counters.Pink++
	case BlueCandy:
		p.Blue = true
		g.Counters.Blue++
	case YellowCandy:
		p.Yellow = true
		g.Counters.Yellow++
	case Shoes:
		p.Shoes = true
	case OrangeParasol:
		points, warp = 200, 3
	case RedParasol:
		points, warp = 200, 5
	case PurpleParasol:
		points, warp = 200, 7
	case CrystalRing:
		points = 1000
		p.Crystal = true
	case AmethystRing:
		points = 1000
		p.Amethyst = true
	case RubyRing:
		points = 1000
		p.Ruby = true
	case Dynamite:
		points = 200
		for i := range g.Enemies {
			g.defeat(&g.Enemies[i], 2)
		}
		for i := range g.Bubbles {
			if g.Bubbles[i].EnemyID != 0 {
				g.Bubbles[i].Removed = true
			}
		}
		g.Projectiles = nil
		g.Sounds = append(g.Sounds, "explosion")
	case Clock:
		points = 200
		g.Freeze = 7 * TPS
	}
	g.addScore(points, g.Players[player].X, g.Players[player].Y)
	g.Sounds = append(g.Sounds, "item")
	if warp > 0 {
		g.loadLevel(g.LevelIndex + warp)
		return true
	}
	return false
}
