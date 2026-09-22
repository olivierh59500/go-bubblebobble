package game

import (
	"fmt"
	"math"
)

// Snapshot is a renderable authoritative frame. It omits the campaign, RNG and
// bonus counters: guests never simulate an independent copy of the match.
type Snapshot struct {
	Round                                      int
	State                                      State
	Paused, Hurry                              bool
	Tick, LevelTicks, Countdown, Freeze, Score int
	Players                                    []Player
	Enemies                                    []Enemy
	Bubbles                                    []Bubble
	Projectiles                                []Projectile
	Items                                      []Item
	Effects                                    []Effect
	Particles                                  []Particle
	Sounds                                     []string
}

func (g *Game) Snapshot() Snapshot {
	return Snapshot{Round: g.Level.Number, State: g.State, Paused: g.Paused, Hurry: g.Hurry, Tick: g.Tick, LevelTicks: g.LevelTicks, Countdown: g.Countdown, Freeze: g.Freeze, Score: g.Score,
		Players: append([]Player(nil), g.Players...), Enemies: append([]Enemy(nil), g.Enemies...), Bubbles: append([]Bubble(nil), g.Bubbles...), Projectiles: append([]Projectile(nil), g.Projectiles...), Items: append([]Item(nil), g.Items...), Effects: append([]Effect(nil), g.Effects...), Particles: append([]Particle(nil), g.Particles...), Sounds: append([]string(nil), g.Sounds...)}
}

func (s Snapshot) Validate() error {
	if s.Round < 1 || s.Round > 25 || s.State < Ready || s.State > Won || s.Score < 0 || s.Tick < 0 || len(s.Players) != 2 || len(s.Enemies) > 32 || len(s.Bubbles) > 300 || len(s.Projectiles) > 128 || len(s.Items) > 128 || len(s.Effects) > 256 || len(s.Particles) > 512 || len(s.Sounds) > 64 {
		return fmt.Errorf("invalid multiplayer frame")
	}
	valid := func(b Body) bool {
		for _, v := range []float64{b.X, b.Y, b.VX, b.VY} {
			if math.IsNaN(v) || math.IsInf(v, 0) || math.Abs(v) > 10000 {
				return false
			}
		}
		return true
	}
	for _, p := range s.Players {
		if p.Lives < 0 || p.Lives > 1000 || !valid(p.Body) {
			return fmt.Errorf("invalid player")
		}
	}
	for _, e := range s.Enemies {
		if !valid(e.Body) {
			return fmt.Errorf("invalid enemy")
		}
	}
	for _, b := range s.Bubbles {
		if !valid(b.Body) {
			return fmt.Errorf("invalid bubble")
		}
	}
	for _, p := range s.Projectiles {
		if !valid(p.Body) {
			return fmt.Errorf("invalid projectile")
		}
	}
	for _, i := range s.Items {
		if !valid(i.Body) {
			return fmt.Errorf("invalid item")
		}
	}
	for _, e := range s.Effects {
		if !valid(e.Body) {
			return fmt.Errorf("invalid effect")
		}
	}
	for _, p := range s.Particles {
		if !valid(p.Body) || len(p.Text) > 32 {
			return fmt.Errorf("invalid particle")
		}
	}
	return nil
}

func (g *Game) ApplySnapshot(s Snapshot) error {
	if err := s.Validate(); err != nil {
		return err
	}
	if len(g.Campaign) < s.Round {
		return fmt.Errorf("unknown multiplayer round")
	}
	if g.Level.Number != s.Round {
		g.Level = g.Campaign[s.Round-1].Clone()
	}
	g.LevelIndex = s.Round - 1
	g.State = s.State
	g.Paused = s.Paused
	g.Hurry = s.Hurry
	g.Tick, g.LevelTicks, g.Countdown, g.Freeze, g.Score = s.Tick, s.LevelTicks, s.Countdown, s.Freeze, s.Score
	g.Players, g.Enemies, g.Bubbles = s.Players, s.Enemies, s.Bubbles
	g.Projectiles, g.Items, g.Effects, g.Particles, g.Sounds = s.Projectiles, s.Items, s.Effects, s.Particles, s.Sounds
	return nil
}
