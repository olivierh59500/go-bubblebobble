package app

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"time"

	"bubblebobble/internal/game"
	"bubblebobble/internal/netplay"
	"bubblebobble/internal/platform"
	"bubblebobble/resources"
	"github.com/hajimehoshi/ebiten/v2"
)

func contentSignature() string {
	b, _ := resources.Files.ReadFile("levels.json")
	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:])
}

func (a *App) beginBluetooth(host bool) {
	if !a.bridge.Available.Load() {
		a.netStatus = "BLUETOOTH IS AVAILABLE ON ANDROID DEVICES"
		return
	}
	a.stopNetwork()
	a.netHost = host
	a.netRun = 0
	a.netFinalRun = 0
	a.netSounds = nil
	a.netToken = make([]byte, 16)
	if _, err := rand.Read(a.netToken); err != nil {
		a.netStatus = "COULD NOT PREPARE THE CONNECTION"
		return
	}
	a.netGeneration = a.bridge.Begin()
	a.netStatus = "SELECT A BLUETOOTH DEVICE"
	if host {
		pending, address, err := netplay.Host(a.netToken, contentSignature(), a.store.Profile().Name)
		if err != nil {
			a.failNetwork("COULD NOT START BLUETOOTH HOST")
			return
		}
		a.pending = pending
		a.bridge.BindCancel(pending.Close)
		_, port, _ := net.SplitHostPort(address)
		a.bridge.Queue(fmt.Sprintf("BT_HOST|%d|%s|%s", a.netGeneration, port, hex.EncodeToString(a.netToken)))
		a.netStatus = "WAITING FOR THE SECOND PLAYER"
	} else {
		a.bridge.Queue(fmt.Sprintf("BT_JOIN|%d|%s", a.netGeneration, hex.EncodeToString(a.netToken)))
	}
}

func (a *App) updateNetwork() {
	for _, e := range a.bridge.Events() {
		switch e.Kind {
		case "status":
			a.netStatus = strings.ToUpper(e.Text)
		case "failure":
			a.failNetwork(e.Text)
		case "ready":
			if a.netHost || a.pending != nil || len(a.netToken) != 16 {
				continue
			}
			address, err := platform.Loopback(e.Address)
			if err != nil {
				a.failNetwork("INVALID BLUETOOTH CONNECTION")
				continue
			}
			a.pending = netplay.Join(address, a.netToken, contentSignature(), a.store.Profile().Name)
			a.bridge.BindCancel(a.pending.Close)
			a.netStatus = "CONNECTING TO " + strings.ToUpper(e.Text)
		}
	}
	if a.pending != nil && a.peer == nil {
		peer, err := a.pending.Ready()
		if err != nil {
			a.failNetwork("COULD NOT CONNECT. PLEASE TRY AGAIN.")
			return
		}
		if peer != nil {
			a.peer = peer
			a.bridge.BindPause(func() { peer.Control("pause") })
			clear(a.netToken)
			a.netToken = nil
		}
	}
	if a.peer == nil {
		return
	}
	ready, name, err := a.peer.Status()
	if err != nil {
		message := "CONNECTION CLOSED. HOST OR JOIN TO PLAY AGAIN."
		if strings.Contains(err.Error(), "incompatible") {
			message = "BOTH PHONES MUST USE THE SAME GAME VERSION"
		}
		a.failNetwork(message)
		return
	}
	if !ready {
		return
	}
	a.playerNames[a.playerIndex()] = a.store.Profile().Name
	a.playerNames[1-a.playerIndex()] = name
	if a.netHost && a.netRun == 0 {
		a.startCoop()
	}
	for _, command := range a.peer.TakeControls() {
		if a.match != nil {
			a.match.Paused = command == "pause"
		}
	}
	if !a.netHost {
		if update := a.peer.TakeUpdate(); update != nil {
			if update.Run < a.netRun {
				return
			}
			if a.netRun != update.Run {
				g, err := game.NewPlayers(a.levels, 1, 2)
				if err != nil {
					a.failNetwork("COULD NOT LOAD THE CAMPAIGN")
					return
				}
				a.match = g
				a.netRun = update.Run
				a.recorded = false
				a.navigate(pagePlay)
			}
			if a.match.Level.Number == update.Snapshot.Round {
				a.rememberPositions()
			} else {
				a.renderPositions = nil
			}
			if err := a.match.ApplySnapshot(update.Snapshot); err != nil {
				a.failNetwork("INVALID GAME STATE RECEIVED")
				return
			}
			for _, sound := range a.match.Sounds {
				a.audio.play(sound)
			}
		}
	}
}

func (a *App) startCoop() {
	g, err := game.NewPlayers(a.levels, uint64(time.Now().UnixNano()), 2)
	if err != nil {
		a.failNetwork("COULD NOT LOAD THE CAMPAIGN")
		return
	}
	a.match = g
	a.netRun++
	a.recorded = false
	a.netSounds = nil
	a.navigate(pagePlay)
}

func (a *App) advanceMatch(in game.Input) {
	if a.peer == nil {
		a.match.Step(in)
		return
	}
	if a.netHost {
		a.match.StepPlayers([]game.Input{in, a.peer.Input(time.Now())})
	} else {
		a.peer.SendInput(in)
		a.match.Sounds = nil
	}
}

func (a *App) publishNetwork() {
	if a.peer != nil && a.netHost && a.netRun > 0 && a.match != nil {
		final := a.match.State == game.Won || a.match.State == game.GameOver
		if final && a.netFinalRun == a.netRun || !final && a.frame%3 != 0 {
			return
		}
		s := a.match.Snapshot()
		s.Sounds = append([]string(nil), a.netSounds...)
		a.netSounds = nil
		a.peer.SendState(a.netRun, s)
		if final {
			a.netFinalRun = a.netRun
		}
	}
	if a.peer == nil {
		a.netSounds = nil
	}
}

func (a *App) setPaused(paused bool) {
	if a.match == nil || a.match.Paused == paused {
		return
	}
	a.match.Paused = paused
	if a.peer != nil {
		command := "resume"
		if paused {
			command = "pause"
		}
		a.peer.Control(command)
	}
}

func (a *App) stopNetwork() {
	if a.bridge != nil {
		a.bridge.Stop()
	}
	if a.peer != nil {
		a.peer.Close()
		a.peer = nil
	}
	if a.pending != nil {
		a.pending.Close()
		a.pending = nil
	}
	clear(a.netToken)
	a.netToken = nil
	a.netRun = 0
	a.netSounds = nil
	a.renderPositions = nil
}

func (a *App) failNetwork(message string) {
	// A complete result may arrive immediately before EOF or native teardown.
	// Record those frames before discarding the connection's pending updates.
	if a.peer != nil && !a.netHost {
		for update := a.peer.TakeUpdate(); update != nil; update = a.peer.TakeUpdate() {
			if update.Run < a.netRun || update.Snapshot.State != game.Won && update.Snapshot.State != game.GameOver {
				continue
			}
			if a.match == nil || update.Run != a.netRun {
				a.match, _ = game.NewPlayers(a.levels, 1, 2)
				a.netRun = update.Run
				a.recorded = false
			}
			if a.match != nil && a.match.ApplySnapshot(update.Snapshot) == nil {
				a.record(update.Snapshot.State == game.Won)
			}
		}
	}
	a.stopNetwork()
	a.netStatus = strings.ToUpper(message)
	a.navigate(pageBluetooth)
}
func (a *App) playerIndex() int {
	if a.peer != nil && !a.netHost {
		return 1
	}
	return 0
}
func (a *App) avatar(index int) string {
	if a.match != nil && len(a.match.Players) == 2 {
		if index == 1 {
			return "blue"
		}
		return "green"
	}
	return a.store.Profile().Avatar
}

func (a *App) drawBluetooth(dst *ebiten.Image) {
	a.heading(dst, "BLUETOOTH CO-OP", "TWO PHONES. TWO DRAGONS. ONE CAMPAIGN.")
	a.sprite(dst, "green/walk", a.frame, 224, 158, 72, 1)
	a.sprite(dst, "blue/walk", a.frame, 472, 158, 72, -1)
	a.center(dst, a.netStatus, 12, 263, yellow)
	a.center(dst, "HOST: GREEN DRAGON / JOIN: BLUE DRAGON", 12, 530, green)
	a.center(dst, "3 LIVES EACH / SHARED TEAM SCORE", 12, 564, white)
	a.center(dst, "ALLOW NEARBY DEVICES WHEN ANDROID ASKS", 11, 616, muted)
}
