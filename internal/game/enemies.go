package game

import "math"

func (g *Game) updateEnemies() {
	for i := range g.Enemies {
		e := &g.Enemies[i]
		e.Age++
		if e.State == Trapped || e.State == Carried {
			continue
		}
		if e.State == Defeated {
			continue
		}
		if e.Cooldown > 0 {
			e.Cooldown--
		}
		if e.Shoot > 0 {
			e.Shoot--
			continue
		}
		speed := 2.5
		if e.Angry {
			speed = 3.5
		}
		switch e.Kind {
		case Monsta:
			g.moveFlying(e, float64(e.Dir)*speed, float64(e.Vertical)*speed)
		case Pulpul:
			e.Phase += 0.035
			g.moveFlying(e, float64(e.Dir)*(1.4+math.Abs(math.Sin(e.Phase))*speed), float64(e.Vertical)*(0.7+math.Abs(math.Cos(2*e.Phase))*speed))
		default:
			if e.Kind == Invader {
				speed *= 1.5
			}
			if e.Ground {
				switch e.Kind {
				case Banebou:
					e.VY = -7.5
					if e.Angry {
						e.VY = -10
					}
				case ZenChan, Mighta:
					if e.Cooldown == 0 && g.Player.Y < e.Y-Tile {
						e.VY = -11.5
						e.Dir = direction(g.Player.X - e.X)
						e.Cooldown = 60
					}
				}
			}
			if e.Kind == Mighta && e.Cooldown == 0 && math.Abs(e.Y-g.Player.Y) < Tile && g.Player.Dead == 0 {
				e.Dir = direction(g.Player.X - e.X)
				g.Projectiles = append(g.Projectiles, Projectile{Body: Body{X: e.X + 20 + float64(e.Dir)*24, Y: e.Y + 14, VX: float64(e.Dir) * 6}})
				e.Cooldown, e.Shoot = 180, 30
			}
			if e.Kind == Invader && e.Cooldown == 0 {
				g.Projectiles = append(g.Projectiles, Projectile{Body: Body{X: e.X + 16, Y: e.Y + ActorSize, VY: 6}, Laser: true})
				e.Cooldown, e.Shoot = 180, 12
				g.Sounds = append(g.Sounds, "laser")
			}
			oldX := e.X
			g.moveActor(&e.Body, float64(e.Dir)*speed, true)
			if math.Abs(e.X-oldX) < 0.01 {
				e.Dir *= -1
			}
		}
	}
}

func (g *Game) updateDefeated() {
	for i := range g.Enemies {
		e := &g.Enemies[i]
		if e.State == Defeated && e.DeadTicks > 0 {
			e.DeadTicks--
			e.X += e.VX
			e.Y += e.VY
			e.VY += 0.45
			if e.DeadTicks == 0 {
				g.spawnFood(e.X, e.Y)
			}
		}
	}
}

func (g *Game) updateProjectiles() {
	kept := g.Projectiles[:0]
	for _, p := range g.Projectiles {
		p.Age++
		steps := max(1, int(math.Ceil(max(math.Abs(p.VX), math.Abs(p.VY)))))
		for i := 0; i < steps; i++ {
			p.X += p.VX / float64(steps)
			p.Y += p.VY / float64(steps)
			if g.blocked(p.X, p.Y, 16) || p.Y > Height {
				p.Removed = true
				break
			}
		}
		if !p.Removed && p.Age < 5*TPS {
			kept = append(kept, p)
		}
	}
	g.Projectiles = kept
}
