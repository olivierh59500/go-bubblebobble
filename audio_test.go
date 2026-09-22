package main

import (
	"bytes"
	"io"
	"testing"

	"bubblebobble/resources"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
)

func TestAllAudioDecodesWithoutDevice(t *testing.T) {
	a, err := newSoundSystem(true)
	if err != nil {
		t.Fatal(err)
	}
	if a.context != nil {
		t.Fatal("muted startup opened an audio device")
	}
	for name, data := range a.sounds {
		if len(data) == 0 || len(data)%4 != 0 {
			t.Errorf("invalid stereo PCM for %s", name)
		}
	}
	for _, name := range []string{"menu", "opening", "theme", "hurry"} {
		data, err := resources.Files.ReadFile("audio/" + name + ".ogg")
		if err != nil {
			t.Fatal(err)
		}
		stream, err := vorbis.DecodeWithSampleRate(sampleRate, bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		n, err := io.Copy(io.Discard, stream)
		if err != nil {
			t.Fatal(err)
		}
		if n != stream.Length() || n == 0 || n%4 != 0 {
			t.Errorf("%s has invalid decoded length %d, declared %d", name, n, stream.Length())
		}
		if _, err := stream.Seek(0, io.SeekStart); err != nil {
			t.Fatal(err)
		}
		buf := make([]byte, 1024)
		if _, err := io.ReadFull(stream, buf); err != nil {
			t.Fatal(err)
		}
	}
}
