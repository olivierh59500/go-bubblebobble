package main

import (
	"fmt"
	"io"
	"os"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	"github.com/hajimehoshi/go-mp3"
)

type soundData struct {
	Bytes  []byte
	Volume float64
}

type musicData struct {
	Path   string
	Volume float64
}

type audioManager struct {
	ctx         *audio.Context
	sounds      map[string]soundData
	music       map[string]musicData
	musicPlayer *audio.Player
	openFiles   []*os.File
	players     []*audio.Player
}

type audioConfig struct {
	Sound []struct {
		Name   string   `json:"name"`
		Path   string   `json:"path"`
		Volume *float64 `json:"volume"`
	} `json:"sound"`
	Music []struct {
		Name   string   `json:"name"`
		Path   string   `json:"path"`
		Volume *float64 `json:"volume"`
	} `json:"music"`
}

func loadAudioManager() (*audioManager, error) {
	manager := &audioManager{
		ctx:    audio.NewContext(audioSampleRate),
		sounds: map[string]soundData{},
		music:  map[string]musicData{},
	}
	var config audioConfig
	if err := loadJSON("res/sounds/Audio.json", &config); err != nil {
		return manager, err
	}
	var loadErrors []error
	for _, sound := range config.Sound {
		volume := 1.0
		if sound.Volume != nil {
			volume = *sound.Volume
		}
		file, err := os.Open(sound.Path)
		if err != nil {
			loadErrors = append(loadErrors, fmt.Errorf("open sound %s: %w", sound.Name, err))
			continue
		}
		stream, err := wav.DecodeWithSampleRate(audioSampleRate, file)
		if err != nil {
			_ = file.Close()
			loadErrors = append(loadErrors, fmt.Errorf("decode sound %s: %w", sound.Name, err))
			continue
		}
		bytes, err := io.ReadAll(stream)
		_ = file.Close()
		if err != nil {
			loadErrors = append(loadErrors, fmt.Errorf("read sound %s: %w", sound.Name, err))
			continue
		}
		manager.sounds[sound.Name] = soundData{Bytes: bytes, Volume: volume}
	}
	for _, music := range config.Music {
		volume := 1.0
		if music.Volume != nil {
			volume = *music.Volume
		}
		manager.music[music.Name] = musicData{Path: music.Path, Volume: volume}
	}
	if len(loadErrors) > 0 {
		return manager, fmt.Errorf("audio loaded with %d sound error(s): %w", len(loadErrors), loadErrors[0])
	}
	return manager, nil
}

func (a *audioManager) playSound(name string) {
	if a == nil || a.ctx == nil {
		return
	}
	sound, ok := a.sounds[name]
	if !ok || len(sound.Bytes) == 0 {
		return
	}
	player := a.ctx.NewPlayerFromBytes(sound.Bytes)
	player.SetVolume(sound.Volume)
	player.Play()
	a.players = append(a.players, player)
}

func (a *audioManager) playMusic(name string) {
	if a == nil || a.ctx == nil {
		return
	}
	if a.musicPlayer != nil {
		if !a.musicPlayer.IsPlaying() {
			a.musicPlayer.Play()
		}
		return
	}
	music, ok := a.music[name]
	if !ok {
		return
	}
	file, err := os.Open(music.Path)
	if err != nil {
		return
	}
	decoder, err := mp3.NewDecoder(file)
	if err != nil {
		_ = file.Close()
		return
	}
	if decoder.SampleRate() != audioSampleRate || decoder.Length() <= 0 {
		_ = file.Close()
		return
	}
	player, err := a.ctx.NewPlayer(audio.NewInfiniteLoop(decoder, decoder.Length()))
	if err != nil {
		_ = file.Close()
		return
	}
	player.SetVolume(music.Volume)
	player.Play()
	a.musicPlayer = player
	a.openFiles = append(a.openFiles, file)
}

func (a *audioManager) update() {
	if a == nil {
		return
	}
	players := a.players[:0]
	for _, player := range a.players {
		if player.IsPlaying() {
			players = append(players, player)
			continue
		}
		_ = player.Close()
	}
	a.players = players
}
