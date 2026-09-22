package platform

import (
	"sync"
	"testing"
)

func TestCancelledCallbacksCannotReviveConnection(t *testing.T) {
	b := &Bridge{}
	first := b.Begin()
	b.Stop()
	second := b.Begin()
	b.Post(Event{Generation: first, Kind: "ready", Address: "127.0.0.1:1234"})
	b.Post(Event{Generation: second, Kind: "status", Text: "Connected"})
	e := b.Events()
	if len(e) != 1 || e[0].Kind != "status" {
		t.Fatal("cancelled callback escaped into new session")
	}
	for _, address := range []string{"example.com:42", "192.168.0.1:42", "127.0.0.1:0", "127.0.0.1:99999"} {
		if _, err := Loopback(address); err == nil {
			t.Fatalf("invalid endpoint accepted: %s", address)
		}
	}
	if _, err := Loopback("127.0.0.1:1234"); err != nil {
		t.Fatal(err)
	}
}

func TestLifecycleAndConcurrentCallbacks(t *testing.T) {
	b := &Bridge{}
	generation := b.Begin()
	paused := false
	closed := false
	b.BindPause(func() { paused = true })
	b.BindCancel(func() { closed = true })
	b.Suspend()
	if !paused || !b.Pause.Load() || b.InputReset.Load() != 1 {
		t.Fatal("suspension did not reset controls and notify peer")
	}
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < 100; n++ {
				b.Post(Event{Generation: generation, Kind: "status", Text: "Searching"})
				b.Poll()
			}
		}()
	}
	wg.Wait()
	if len(b.Events()) > 32 {
		t.Fatal("unbounded callback queue")
	}
	b.CancelFromPlatform()
	if !closed {
		t.Fatal("native teardown left transport open")
	}
}
