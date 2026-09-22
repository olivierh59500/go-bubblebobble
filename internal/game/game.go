// Package game implements a deterministic arcade simulation independent of rendering.
package game

import (
	"fmt"
	"math"
	"math/rand/v2"
)

type State int

const (
	Ready State = iota
	Playing
	Clearing
	GameOver
	Won
)

type Input struct {
	Move       int
	Jump, Fire bool
}

type Body struct {
	X, Y, VX, VY float64
	Ground       bool
}

type Player struct {
	Body
	Dir                               int
	Invincible, Dead, Shoot, Cooldown int
	WalkFrame                         int
	Power                             Powers
}

type EnemyState int

const (
	Active EnemyState = iota
	Trapped
	Defeated
	Carried
)

type Enemy struct {
	Body
	CarriedBy, CarryTicks           int
	ID                              int
	Kind                            EnemyKind
	State                           EnemyState
	Dir, Vertical                   int
	Angry                           bool
	Age, Cooldown, Shoot, DeadTicks int
	Phase                           float64
}

type Bubble struct {
	Body
	ID, Dir, Age, EnemyID, CapturedAge int
	Distance, MaxDistance, Speed       float64
	Floating, Removed                  bool
	Element                            Element
}

type Projectile struct {
	Body
	Laser   bool
	Age     int
	Removed bool
}

type Effect struct {
	Body
	ID             int
	Element        Element
	Dir, Age, Life int
	Spread         bool
}

type Particle struct {
	Body
	Text string
	Life int
}

type Game struct {
	Campaign                            []Level
	Level                               Level
	LevelIndex                          int
	Custom                              bool
	State                               State
	Paused                              bool
	Tick, LevelTicks, Countdown, Freeze int
	Hurry                               bool
	Score, Lives                        int
	Player                              Player
	Enemies                             []Enemy
	Bubbles                             []Bubble
	Projectiles                         []Projectile
	Items                               []Item
	Effects                             []Effect
	Particles                           []Particle
	Sounds                              []string
	Counters                            Counters
	spawnedSpecial, nextID, extraLifeAt int
	jumpBuffer, coyote                  int
	lastJump                            bool
	rng                                 *rand.Rand
}

func New(levels []Level, seed uint64) (*Game, error) {
	if len(levels) == 0 {
		return nil, fmt.Errorf("no levels supplied")
	}
	for _, level := range levels {
		if err := level.Validate(); err != nil {
			return nil, err
		}
	}
	g := &Game{Campaign: levels, Lives: 3, extraLifeAt: 30000, rng: rand.New(rand.NewPCG(seed, seed^0xa0761d6478bd642f))}
	g.loadLevel(0)
	return g, nil
}

func (g *Game) loadLevel(index int) {
	if index >= len(g.Campaign) {
		g.State = Won
		g.Sounds = append(g.Sounds, "opening")
		return
	}
	g.LevelIndex = index
	g.Level = g.Campaign[index].Clone()
	g.LevelTicks, g.Freeze, g.spawnedSpecial = 0, 0, 0
	g.Hurry, g.Paused = false, false
	g.Bubbles, g.Projectiles, g.Items, g.Effects, g.Particles = nil, nil, nil, nil, nil
	g.Enemies = nil
	for _, spawn := range g.Level.Enemies {
		x, y := g.spawnPosition(spawn.Cell)
		g.nextID++
		dir := -1
		if g.rng.IntN(2) == 0 {
			dir = 1
		}
		g.Enemies = append(g.Enemies, Enemy{Body: Body{X: x, Y: y}, ID: g.nextID, Kind: spawn.Kind, Dir: dir, Vertical: 1, Cooldown: 180, Phase: g.rng.Float64() * 2 * math.Pi})
	}
	g.respawn()
	g.Player.Invincible = 0
	g.State, g.Countdown = Ready, TPS
}

func (g *Game) respawn() {
	x, y := g.spawnPosition(g.Level.Player)
	g.Player = Player{Body: Body{X: x, Y: y}, Dir: 1, Invincible: 210}
	g.jumpBuffer, g.coyote = 0, 0
}

func (g *Game) Step(in Input) {
	g.Sounds = g.Sounds[:0]
	if g.Paused || g.State == GameOver || g.State == Won {
		return
	}
	g.Tick++
	if g.State == Ready {
		g.Countdown--
		if g.Countdown <= 0 {
			g.State = Playing
		}
		return
	}
	if g.Player.Dead > 0 {
		g.Player.Dead--
		if g.Player.Dead == 0 {
			if g.Lives == 0 {
				g.State = GameOver
				return
			}
			g.respawn()
			g.Projectiles = nil
		}
	} else {
		g.updatePlayer(in)
	}
	if g.State == Playing {
		if g.Freeze > 0 {
			g.Freeze--
		} else {
			g.LevelTicks++
			if g.LevelTicks >= 30*TPS && !g.Hurry {
				g.Hurry = true
				for i := range g.Enemies {
					g.Enemies[i].Angry = true
				}
			}
			g.updateEnemies()
			g.updateProjectiles()
		}
		g.spawnSpecial()
	}
	g.updateBubbles(in)
	g.updateEffects()
	if g.updateItems() {
		return
	}
	g.updateDefeated()
	g.updateParticles()
	if g.State == Playing {
		g.checkPowerSpawns()
		g.checkPlayerHit()
		if g.remainingEnemies() == 0 && g.Player.Dead == 0 {
			g.State, g.Countdown = Clearing, 7*TPS
			g.Projectiles = nil
		}
	} else if g.State == Clearing {
		g.Countdown--
		if g.Countdown == 0 {
			g.loadLevel(g.LevelIndex + 1)
		}
	}
}

func (g *Game) updatePlayer(in Input) {
	p := &g.Player
	if p.Invincible > 0 {
		p.Invincible--
	}
	if p.Shoot > 0 {
		p.Shoot--
	}
	if p.Cooldown > 0 {
		p.Cooldown--
	}
	if in.Jump && !g.lastJump {
		g.jumpBuffer = 8
	}
	g.lastJump = in.Jump
	if g.jumpBuffer > 0 {
		g.jumpBuffer--
	}
	if p.Ground {
		g.coyote = 6
	} else if g.coyote > 0 {
		g.coyote--
	}
	if g.jumpBuffer > 0 && g.coyote > 0 {
		p.VY, p.Ground = -11.5, false
		g.coyote, g.jumpBuffer = 0, 0
		g.Counters.Jumps++
		if p.Power.Amethyst {
			g.addScore(500, p.X, p.Y)
		}
		g.Sounds = append(g.Sounds, "jump")
	}
	speed := 3.0
	if p.Power.Shoes {
		speed = 5
	}
	in.Move = max(-1, min(1, in.Move))
	if in.Move != 0 {
		p.Dir = in.Move
		p.WalkFrame++
	}
	oldX := p.X
	g.moveActor(&p.Body, float64(in.Move)*speed, true)
	distance := math.Abs(p.X - oldX)
	g.Counters.Walked += distance
	if p.Power.Crystal {
		p.Power.WalkPoints += distance
		for p.Power.WalkPoints >= 8 {
			g.addScore(10, p.X, p.Y)
			p.Power.WalkPoints -= 8
		}
	}
	if in.Fire && p.Cooldown == 0 {
		g.shootBubble()
		p.Cooldown = 18
		if p.Power.Yellow {
			p.Cooldown = 5
		}
		p.Shoot = 12
	}
}

func (g *Game) checkPlayerHit() {
	p := &g.Player
	if p.Invincible > 0 || p.Dead > 0 {
		return
	}
	for _, e := range g.Enemies {
		if e.State == Active && intersects(p.X+5, p.Y+5, 34, 36, e.X+4, e.Y+4, 36, 36) {
			g.hurtPlayer()
			return
		}
	}
	for i := range g.Projectiles {
		b := &g.Projectiles[i]
		if !b.Removed && intersects(p.X+5, p.Y+5, 34, 36, b.X, b.Y, 16, 20) {
			b.Removed = true
			g.hurtPlayer()
			return
		}
	}
}

func (g *Game) hurtPlayer() {
	if g.Player.Dead > 0 || g.Player.Invincible > 0 {
		return
	}
	g.Lives--
	g.Player.Dead = 90
	g.Player.Power = Powers{}
	g.Sounds = append(g.Sounds, "death")
}

func (g *Game) remainingEnemies() int {
	n := 0
	for _, e := range g.Enemies {
		if e.State != Defeated {
			n++
		}
	}
	return n
}

func (g *Game) RemainingEnemies() int { return g.remainingEnemies() }

func (g *Game) enemy(id int) *Enemy {
	for i := range g.Enemies {
		if g.Enemies[i].ID == id {
			return &g.Enemies[i]
		}
	}
	return nil
}

func (g *Game) defeat(e *Enemy, multiplier int) {
	if e.State == Defeated {
		return
	}
	e.State, e.DeadTicks = Defeated, 50
	e.VX, e.VY = float64(g.Player.Dir)*4, -8
	g.addScore(100*multiplier, e.X, e.Y)
}

func (g *Game) addScore(points int, x, y float64) {
	g.Score += points
	if points >= 100 {
		g.Particles = append(g.Particles, Particle{Body: Body{X: x, Y: y}, Text: fmt.Sprint(points), Life: 45})
	}
	for g.Score >= g.extraLifeAt {
		g.Lives++
		g.extraLifeAt += 100000
		g.Particles = append(g.Particles, Particle{Body: Body{X: g.Player.X, Y: g.Player.Y - 24}, Text: "1UP", Life: 90})
	}
}

func (g *Game) updateParticles() {
	kept := g.Particles[:0]
	for _, p := range g.Particles {
		p.Life--
		p.Y -= 0.6
		if p.Life > 0 {
			kept = append(kept, p)
		}
	}
	g.Particles = kept
}

func intersects(x, y, w, h, xx, yy, ww, hh float64) bool {
	return x < xx+ww && x+w > xx && y < yy+hh && y+h > yy
}
func direction(n float64) int {
	if n < 0 {
		return -1
	}
	return 1
}
