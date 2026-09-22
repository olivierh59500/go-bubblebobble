package game

import "math"

const bubbleSize = 40

func (g *Game) shootBubble(player int) {
	p := &g.Players[player]
	speed, reach := 8.0, 200.0
	if p.Power.Blue {
		speed = 16
	}
	if p.Power.Pink {
		reach = 400
	}
	g.nextID++
	g.Bubbles = append(g.Bubbles, Bubble{Body: Body{X: p.X + float64(p.Dir)*18, Y: p.Y + 2}, ID: g.nextID, Dir: p.Dir, Speed: speed, MaxDistance: reach})
	g.Counters.Blown++
	if p.Power.Ruby {
		g.addScore(100, p.X, p.Y)
	}
	g.Sounds = append(g.Sounds, "shoot")
}

func (g *Game) updateBubbles(inputs []Input) {
	for i := range g.Bubbles {
		b := &g.Bubbles[i]
		if b.Removed {
			continue
		}
		b.Age++
		if b.Element != "" {
			g.moveSpecialBubble(b)
		} else if !b.Floating {
			for step := 0; step < int(b.Speed); step++ {
				b.X += float64(b.Dir)
				b.Distance++
				for j := range g.Enemies {
					e := &g.Enemies[j]
					if e.State == Active && intersects(b.X, b.Y, bubbleSize, bubbleSize, e.X+3, e.Y+3, 38, 38) {
						b.EnemyID = e.ID
						e.State = Trapped
						b.Floating = true
						break
					}
				}
				if b.Floating || b.Distance >= b.MaxDistance || g.blocked(b.X+4, b.Y+4, bubbleSize-8) {
					b.Floating = true
					break
				}
			}
		} else {
			ceiling := float64(2 * Tile)
			if g.Level.Number == 24 {
				ceiling = 9 * Tile
			}
			if b.Y > ceiling {
				b.Y -= 1.15
			} else {
				b.Y = ceiling + math.Sin(float64(b.Age)/28)*4
			}
			b.X += math.Sin(float64(b.ID)*1.7+float64(b.Age)/60) * 0.2
		}
		b.X = max(float64(2*Tile), min(float64(Width-2*Tile-bubbleSize), b.X))
		if b.EnemyID != 0 {
			b.CapturedAge++
			if e := g.enemy(b.EnemyID); e != nil {
				e.X, e.Y = b.X, b.Y
			}
			if b.CapturedAge > 1270 {
				if e := g.enemy(b.EnemyID); e != nil {
					e.State = Active
					e.Angry = true
					e.VY = 0
					e.Y += 4
				}
				b.Removed = true
				continue
			}
		} else if b.Age > 600 {
			b.Removed = true
			continue
		}
		for player := range g.Players {
			in := Input{}
			if player < len(inputs) {
				in = inputs[player]
			}
			p := &g.Players[player]
			if !p.Alive() || b.Age <= 20 || b.Removed {
				continue
			}
			if in.Jump && p.VY >= 0 && p.Y+ActorSize >= b.Y && p.Y+ActorSize-p.VY <= b.Y+10 && p.X+ActorSize > b.X+4 && p.X < b.X+bubbleSize-4 {
				p.Y = b.Y - ActorSize
				p.VY = -10.5
				p.Ground = false
				g.Sounds = append(g.Sounds, "jump")
				continue
			}
			if intersects(p.X+4, p.Y+4, 36, 36, b.X, b.Y, bubbleSize, bubbleSize) {
				g.popBubble(i, player)
			}
		}
	}
	kept := g.Bubbles[:0]
	for _, b := range g.Bubbles {
		if !b.Removed {
			kept = append(kept, b)
		}
	}
	g.Bubbles = kept
}

func (g *Game) popBubble(index, player int) {
	queue := []int{index}
	combo := 0
	for len(queue) > 0 {
		i := queue[0]
		queue = queue[1:]
		b := &g.Bubbles[i]
		if b.Removed {
			continue
		}
		b.Removed = true
		g.Counters.Popped++
		g.Particles = append(g.Particles, Particle{Body: Body{X: b.X, Y: b.Y}, Life: 15})
		if e := g.enemy(b.EnemyID); e != nil && e.State == Trapped {
			combo++
			g.defeat(e, 1<<min(combo-1, 5))
			x, y := g.Level.FoodSpawn.Position(32)
			g.Items = append(g.Items, Item{Body: Body{X: x, Y: y}, Food: g.Level.Number, Stationary: true})
		}
		if b.Element != "" {
			g.releaseElement(b, player)
		}
		for j := range g.Bubbles {
			other := &g.Bubbles[j]
			if !other.Removed && other.Age > 20 && intersects(b.X-6, b.Y-6, bubbleSize+12, bubbleSize+12, other.X, other.Y, bubbleSize, bubbleSize) {
				queue = append(queue, j)
			}
		}
	}
	g.Sounds = append(g.Sounds, "pop")
}

func (g *Game) spawnSpecial() {
	if g.Level.Special == "" || g.spawnedSpecial >= g.Level.SpecialCount {
		return
	}
	if g.Freeze == 0 && g.LevelTicks >= 1 && g.spawnedSpecial <= (g.LevelTicks-1)/450 {
		g.nextID++
		g.Bubbles = append(g.Bubbles, Bubble{Body: Body{X: 252, Y: Tile}, ID: g.nextID, Dir: 1, Age: 21, Floating: true, Element: g.Level.Special})
		g.spawnedSpecial++
	}
}

func (g *Game) moveSpecialBubble(b *Bubble) {
	if !g.blocked(b.X+4, b.Y+4+3, bubbleSize-8) {
		b.Y += 3
	} else {
		next := b.X + float64(b.Dir)*2
		if g.blocked(next+4, b.Y+4, bubbleSize-8) {
			b.Dir *= -1
		} else {
			b.X = next
		}
	}
	if b.Y > Height {
		b.Y = Tile
	}
}

func (g *Game) releaseElement(b *Bubble, player int) {
	g.nextID++
	e := Effect{ID: g.nextID, Body: Body{X: b.X, Y: b.Y}, Element: b.Element, Dir: g.Players[player].Dir, Life: 300}
	switch b.Element {
	case Water:
		g.Counters.Water++
		e.Life = 420
		g.Sounds = append(g.Sounds, "water")
	case Thunder:
		g.Counters.Thunder++
		e.Life = 90
		g.Sounds = append(g.Sounds, "laser")
	case Fire:
		g.Counters.Fire++
		g.Sounds = append(g.Sounds, "fire")
	}
	g.Effects = append(g.Effects, e)
}

func (g *Game) updateEffects() {
	var spawned []Effect
	for i := range g.Effects {
		e := &g.Effects[i]
		e.Age++
		e.Life--
		size := 32.0
		switch e.Element {
		case Thunder:
			// Sweep lightning so a fast bolt cannot pass through an enemy.
			for step := 0; step < 12; step++ {
				e.X += float64(e.Dir)
				g.hitWithElement(e, 40, 24)
			}
			if e.X < 2*Tile || e.X > Width-2*Tile {
				e.Life = 0
			}
		case Water:
			if !g.blocked(e.X+3, e.Y+6, size-6) {
				e.Y += 3
			} else {
				next := e.X + float64(e.Dir)*3
				if g.blocked(next+3, e.Y+3, size-6) {
					e.Dir *= -1
				} else {
					e.X = next
				}
			}
			if e.Age%12 == 0 && e.Age <= 72 && !e.Spread {
				spawned = append(spawned, Effect{Body: e.Body, Element: Water, Dir: e.Dir, Life: 150, Spread: true})
			}
			if e.Y > Height {
				e.Life = 0
			}
			g.hitWithElement(e, size, size)
			g.carryOnWater(e)
		case Fire:
			if !e.Ground {
				if !g.blocked(e.X+3, e.Y+7, size-6) {
					e.Y += 4
				} else {
					e.Ground = true
					if !e.Spread {
						for _, dir := range []int{-1, 1} {
							spawned = append(spawned, Effect{Body: Body{X: e.X + float64(dir)*28, Y: e.Y}, Element: Fire, Dir: dir, Life: 300, Spread: true})
						}
					}
				}
			}
			if e.Y > Height {
				e.Life = 0
			}
			g.hitWithElement(e, size, size)
		}
	}
	kept := g.Effects[:0]
	for _, e := range g.Effects {
		if e.Life > 0 {
			kept = append(kept, e)
		}
	}
	g.Effects = append(kept, spawned...)
}

func (g *Game) hitWithElement(effect *Effect, w, h float64) {
	if effect.Element == Fire && !effect.Ground {
		return
	}
	for i := range g.Enemies {
		e := &g.Enemies[i]
		if e.State == Active && intersects(effect.X, effect.Y, w, h, e.X, e.Y, ActorSize, ActorSize) {
			if effect.Element == Water && !effect.Spread {
				e.State = Carried
				e.CarriedBy = effect.ID
				e.CarryTicks = 0
			} else if effect.Element != Water {
				g.defeat(e, 2)
			}
		}
	}
}

func (g *Game) carryOnWater(effect *Effect) {
	if effect.Spread {
		return
	}
	for i := range g.Enemies {
		e := &g.Enemies[i]
		if e.State == Carried && e.CarriedBy == effect.ID {
			e.X, e.Y = effect.X, effect.Y-12
			e.CarryTicks++
			if effect.Life <= 0 || e.CarryTicks >= 3*TPS || e.Y > Height-ActorSize {
				g.defeat(e, 2)
			}
		}
	}
	for player := range g.Players {
		p := &g.Players[player]
		if p.Alive() && p.VY >= 0 && intersects(p.X, p.Y, ActorSize, ActorSize, effect.X, effect.Y, 32, 32) {
			x, y := effect.X, effect.Y-12
			if !g.blocked(x, y, ActorSize) {
				p.X, p.Y = x, y
			}
		}
	}
}
