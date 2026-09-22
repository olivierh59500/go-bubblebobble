package game

import "testing"

func coop(t *testing.T) *Game {
	t.Helper()
	g, err := NewPlayers([]Level{testLevel()}, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	g.State = Playing
	return g
}

func TestCoopIndependentControlsLivesAndBonuses(t *testing.T) {
	g := coop(t)
	x0, x1 := g.Players[0].X, g.Players[1].X
	g.StepPlayers([]Input{{Move: 1, Fire: true}, {Move: -1, Fire: true}})
	if g.Players[0].X <= x0 || g.Players[1].X >= x1 || len(g.Bubbles) != 2 {
		t.Fatal("both players must move and shoot independently")
	}
	g.applyPower(Shoes, 1)
	if g.Players[0].Power.Shoes || !g.Players[1].Power.Shoes {
		t.Fatal("bonus went to the wrong dragon")
	}
	g.hurtPlayer(1)
	if g.Players[0].Lives != 3 || g.Players[1].Lives != 2 || g.Players[0].Dead != 0 {
		t.Fatal("death affected the other player")
	}
	g.Players[1].Lives = 0
	for i := 0; i < 90; i++ {
		g.StepPlayers([]Input{{}, {}})
	}
	if g.State == GameOver || g.Players[1].Alive() {
		t.Fatal("one survivor must keep the campaign running")
	}
	g.Players[0].Lives = 1
	g.Players[0].Invincible = 0
	g.hurtPlayer(0)
	for i := 0; i < 90; i++ {
		g.StepPlayers(nil)
	}
	if g.State != GameOver {
		t.Fatal("both eliminated players must end the run")
	}
}

func TestCoopSecondPlayerCanPopCollectAndRevive(t *testing.T) {
	g := coop(t)
	g.Players[0].Lives = 0
	g.Players[1].X, g.Players[1].Y = 300, 300
	g.Enemies[0].State = Trapped
	g.Bubbles = []Bubble{{Body: Body{X: 300, Y: 300}, Floating: true, Age: 50, EnemyID: g.Enemies[0].ID}}
	g.updateBubbles([]Input{{}, {}})
	if g.Enemies[0].State != Defeated || g.Score != 100 {
		t.Fatal("second dragon could not pop a captured enemy")
	}
	g.Items = []Item{{Body: Body{X: 300, Y: 300}, Power: BlueCandy}}
	g.updateItems()
	if !g.Players[1].Power.Blue || g.Players[0].Power.Blue {
		t.Fatal("item ownership is incorrect")
	}
	g.addScore(30000, 300, 300)
	if !g.Players[0].Alive() || g.Players[0].Invincible == 0 {
		t.Fatal("team extra life did not revive eliminated teammate")
	}
}

func TestSnapshotValidationAndIsolation(t *testing.T) {
	g := coop(t)
	s := g.Snapshot()
	g.Players[0].X += 100
	if s.Players[0].X == g.Players[0].X {
		t.Fatal("snapshot shares mutable simulation data")
	}
	guest, err := NewPlayers(campaign(t), 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err = guest.ApplySnapshot(s); err != nil {
		t.Fatal(err)
	}
	if guest.Players[0] != s.Players[0] {
		t.Fatal("guest did not apply authoritative player")
	}
	s.Round = 999
	if guest.ApplySnapshot(s) == nil {
		t.Fatal("invalid snapshot accepted")
	}
}
