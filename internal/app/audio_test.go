package app

import (
	"testing"
)

func TestAllAudioDecodesWithoutDevice(t *testing.T) {
	deferred, err := newSoundSystem(false)
	if err != nil {
		t.Fatal(err)
	}
	if deferred.context != nil {
		t.Fatal("constructor opened audio before the first update")
	}
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
}
