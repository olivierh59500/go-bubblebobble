package main

import (
	"os"
	"testing"

	"github.com/hajimehoshi/go-mp3"
)

func TestLoadAudioManagerLoadsConfiguredAssets(t *testing.T) {
	audio, err := loadAudioManager()
	if err != nil {
		t.Fatalf("load audio: %v", err)
	}
	if len(audio.sounds) != 8 {
		t.Fatalf("expected 8 configured sounds, got %d", len(audio.sounds))
	}
	if len(audio.sounds["bubble-shoot-sound"].Bytes) == 0 {
		t.Fatal("bubble shoot sound decoded to empty PCM data")
	}
	if audio.music["main-theme"].Path == "" {
		t.Fatal("main theme music was not registered")
	}
}

func TestMainThemeMatchesAudioContextSampleRate(t *testing.T) {
	file, err := os.Open("res/sounds/tim-follin-atari/02 Bubble Bobble - Ingame-Title__Loop.mp3")
	if err != nil {
		t.Fatalf("open main theme: %v", err)
	}
	defer file.Close()
	decoder, err := mp3.NewDecoder(file)
	if err != nil {
		t.Fatalf("decode main theme: %v", err)
	}
	if decoder.SampleRate() != audioSampleRate {
		t.Fatalf("main theme sample rate is %d, audio context is %d", decoder.SampleRate(), audioSampleRate)
	}
	if decoder.Length() <= 0 {
		t.Fatal("main theme decoded length should be available for looping")
	}
}
