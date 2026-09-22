package main

import (
	"bytes"
	"fmt"
	"io"

	"bubblebobble/resources"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

const sampleRate = 44100

type soundSystem struct {
	context                  *audio.Context
	sounds                   map[string][]byte
	players                  []*audio.Player
	music                    *audio.Player
	track                    string
	musicVolume, soundVolume float64
	muted                    bool
}

func newSoundSystem(muted bool) (*soundSystem, error) {
	a := &soundSystem{sounds: map[string][]byte{}, muted: muted}
	// A muted launch also works on machines without an audio device.
	if !muted {
		a.context = audio.NewContext(sampleRate)
	}
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
		b, err := resources.Files.ReadFile("audio/" + track + ".ogg")
		if err != nil {
			return err
		}
		stream, err := vorbis.DecodeWithSampleRate(sampleRate, bytes.NewReader(b))
		if err != nil {
			return err
		}
		a.music, err = a.context.NewPlayer(audio.NewInfiniteLoop(stream, stream.Length()))
		if err != nil {
			return err
		}
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
	if !a.muted && a.context == nil {
		a.context = audio.NewContext(sampleRate)
	}
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
}
