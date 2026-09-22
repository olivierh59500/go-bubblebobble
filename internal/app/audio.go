package app

import (
	"bubblebobble/internal/ym"
	"bytes"
	"fmt"
	"io"

	"bubblebobble/resources"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

const sampleRate = ym.SampleRate

type soundSystem struct {
	context                  *audio.Context
	sounds                   map[string][]byte
	players                  []*audio.Player
	music                    *audio.Player
	stream                   *ym.Stream
	track                    string
	musicVolume, soundVolume float64
	muted                    bool
}

func newSoundSystem(muted bool) (*soundSystem, error) {
	a := &soundSystem{sounds: map[string][]byte{}, muted: muted}
	// The audio device is opened from Update, after Android creates its view.
	for _, name := range []string{"shoot", "jump", "pop", "death", "food", "item", "fire", "water", "explosion", "laser"} {
		b, err := resources.Files.ReadFile("audio/" + name + ".wav")
		if err != nil {
			return nil, err
		}
		stream, err := wav.DecodeWithSampleRate(sampleRate, bytes.NewReader(b))
		if err != nil {
			return nil, fmt.Errorf("decode %s: %w", name, err)
		}
		a.sounds[name], err = io.ReadAll(stream)
		if err != nil {
			return nil, err
		}
	}
	return a, nil
}

func (a *soundSystem) setVolumes(music, sound int) {
	a.musicVolume = float64(music) / 100
	a.soundVolume = float64(sound) / 100
	if a.music != nil {
		a.music.SetVolume(a.musicVolume)
	}
}

func (a *soundSystem) play(name string) {
	if a.muted || a.context == nil || a.soundVolume == 0 || len(a.players) >= 24 {
		return
	}
	b := a.sounds[name]
	if len(b) == 0 {
		return
	}
	p := a.context.NewPlayerFromBytes(b)
	p.SetVolume(a.soundVolume)
	p.Play()
	a.players = append(a.players, p)
}

func (a *soundSystem) update(track string, paused bool) error {
	if !a.muted && a.context == nil {
		a.context = audio.NewContext(sampleRate)
	}
	kept := a.players[:0]
	for _, p := range a.players {
		if p.IsPlaying() {
			kept = append(kept, p)
		} else {
			p.PauseAndStopReading()
		}
	}
	a.players = kept
	if a.muted || a.context == nil {
		if a.music != nil {
			a.music.Pause()
		}
		return nil
	}
	if a.track != track {
		if a.music != nil {
			a.music.PauseAndStopReading()
			a.music = nil
		}
		if a.stream != nil {
			a.stream.Close()
			a.stream = nil
		}
		b, err := resources.Files.ReadFile("audio/" + musicTracks[track] + ".ym")
		if err != nil {
			return err
		}
		stream, err := ym.New(b, true)
		if err != nil {
			return err
		}
		a.music, err = a.context.NewPlayer(stream)
		if err != nil {
			stream.Close()
			return err
		}
		a.stream = stream
		a.music.SetVolume(a.musicVolume)
		a.track = track
	}
	if paused {
		a.music.Pause()
	} else if !a.music.IsPlaying() {
		a.music.Play()
	}
	return nil
}

func (a *soundSystem) toggleMute() {
	a.muted = !a.muted
	if a.muted {
		for _, p := range a.players {
			p.PauseAndStopReading()
		}
		a.players = nil
	}
}

func (a *soundSystem) close() {
	if a.music != nil {
		a.music.PauseAndStopReading()
	}
	for _, p := range a.players {
		p.PauseAndStopReading()
	}
	if a.stream != nil {
		a.stream.Close()
		a.stream = nil
	}
}

var musicTracks = map[string]string{
	"theme": "bubble-1", "menu": "bubble-2", "opening": "bubble-2",
	"hurry": "bubble-3", "gameover": "bubble-4", "win": "bubble-5",
}
