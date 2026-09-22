package netplay

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"

	"bubblebobble/internal/game"
	"bubblebobble/resources"
)

func waitFor(t *testing.T, f func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !f() {
		if time.Now().After(deadline) {
			t.Fatal("network event timed out")
		}
		time.Sleep(time.Millisecond)
	}
}
func peers(t *testing.T) (*Peer, *Peer) {
	t.Helper()
	a, b := net.Pipe()
	host, guest := New(a, true, "campaign", "BUB"), New(b, false, "campaign", "BOB")
	t.Cleanup(func() { host.Close(); guest.Close(); host.Wait(); guest.Wait() })
	waitFor(t, func() bool { h, _, _ := host.Status(); g, _, _ := guest.Status(); return h && g })
	return host, guest
}

func TestCooperativeSessionInputSnapshotsPauseAndDisconnect(t *testing.T) {
	host, guest := peers(t)
	levels, err := game.LoadCampaign(resources.Files)
	if err != nil {
		t.Fatal(err)
	}
	g, err := game.NewPlayers(levels, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	g.State = game.Playing
	guest.SendInput(game.Input{Move: -1, Fire: true})
	waitFor(t, func() bool { return host.Input(time.Now()).Fire })
	for i := 0; i < 120; i++ {
		g.StepPlayers([]game.Input{{Move: 1, Fire: true}, host.Input(time.Now())})
		if i%3 == 0 {
			host.SendState(1, g.Snapshot())
		}
	}
	host.SendState(1, g.Snapshot())
	var final *Update
	waitFor(t, func() bool {
		if s := guest.TakeUpdate(); s != nil {
			final = s
		}
		return final != nil && final.Snapshot.Tick == g.Tick
	})
	if final.Snapshot.Players[0] != g.Players[0] || final.Snapshot.Players[1] != g.Players[1] || final.Run != 1 {
		t.Fatal("guest diverged from authoritative simulation")
	}
	guest.Control("pause")
	waitFor(t, func() bool { c := host.TakeControls(); return len(c) == 1 && c[0] == "pause" })
	if host.Input(time.Now().Add(time.Second)) != (game.Input{}) {
		t.Fatal("stale controls remained held")
	}
	guest.Close()
	waitFor(t, func() bool { _, _, err := host.Status(); return err != nil })
	if host.Input(time.Now()) != (game.Input{}) {
		t.Fatal("disconnect did not clear controls")
	}
}

func TestIncompatiblePeerAndMalformedFrames(t *testing.T) {
	a, b := net.Pipe()
	host, guest := New(a, true, "first", "BUB"), New(b, false, "different", "BOB")
	defer host.Close()
	defer guest.Close()
	waitFor(t, func() bool { _, _, e := host.Status(); return e != nil })
	host.Wait()
	guest.Wait()
	var oversized [4]byte
	binary.BigEndian.PutUint32(oversized[:], maxWireSize+1)
	if _, err := readFrame(bytes.NewReader(oversized[:])); err == nil {
		t.Fatal("oversized packet accepted")
	}
	if _, err := readFrame(bytes.NewReader([]byte{0, 0, 0, 5, 1})); err == nil {
		t.Fatal("truncated packet accepted")
	}
	p := &Peer{host: true}
	if p.accept(frame{Kind: "input", Input: &game.Input{Move: 900}}) == nil {
		t.Fatal("invalid direction accepted")
	}
	if p.accept(frame{Kind: "state", State: &game.Snapshot{}}) == nil {
		t.Fatal("guest was allowed to overwrite host state")
	}
}

func TestFramingHandlesFragmentedWrites(t *testing.T) {
	var buffer bytes.Buffer
	f := frame{Kind: "input", Input: &game.Input{Move: 1, Jump: true}}
	if err := writeFrame(shortWriter{&buffer}, f); err != nil {
		t.Fatal(err)
	}
	got, err := readFrame(&buffer)
	if err != nil {
		t.Fatal(err)
	}
	if got.Input == nil || *got.Input != *f.Input {
		t.Fatal("fragmented frame corrupted")
	}
}

type shortWriter struct{ io.Writer }

func (s shortWriter) Write(b []byte) (int, error) { return s.Writer.Write(b[:min(3, len(b))]) }

func TestClosingSlowPeerUnblocksBothWorkers(t *testing.T) {
	a, b := net.Pipe()
	p := New(a, true, "campaign", "BUB")
	p.Close()
	b.Close()
	done := make(chan struct{})
	go func() { p.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("socket goroutines leaked on cancellation")
	}
}

func TestFinalScoreSurvivesImmediateRestart(t *testing.T) {
	levels, err := game.LoadCampaign(resources.Files)
	if err != nil {
		t.Fatal(err)
	}
	g, err := game.NewPlayers(levels, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	s := g.Snapshot()
	s.State = game.Won
	s.Score = 12300
	s.Tick = 900
	p := &Peer{}
	if err = p.accept(frame{Kind: "state", State: &s, Run: 1}); err != nil {
		t.Fatal(err)
	}
	next := g.Snapshot()
	if err = p.accept(frame{Kind: "state", State: &next, Run: 2}); err != nil {
		t.Fatal(err)
	}
	final := p.TakeUpdate()
	if final == nil || final.Run != 1 || final.Snapshot.Score != 12300 || final.Snapshot.State != game.Won {
		t.Fatal("restart overwrote final score")
	}
	if update := p.TakeUpdate(); update == nil || update.Run != 2 {
		t.Fatal("restart frame was lost")
	}
	if err = p.accept(frame{Kind: "state", State: &s, Run: 1}); err != nil {
		t.Fatal(err)
	}
	if p.TakeUpdate() != nil {
		t.Fatal("stale run was replayed")
	}
}
