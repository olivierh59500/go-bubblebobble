package ym

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"testing"

	"bubblebobble/resources"
)

func stream(t *testing.T, n int, loop bool) *Stream {
	t.Helper()
	b, err := resources.Files.ReadFile(fmt.Sprintf("audio/bubble-%d.ym", n))
	if err != nil {
		t.Fatal(err)
	}
	s, err := New(b, loop)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestEveryTrackProducesStereoAndLoops(t *testing.T) {
	for i := 1; i <= 5; i++ {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			s := stream(t, i, true)
			b := make([]byte, 4096*4)
			nonzero := false
			for sample := 0; sample < SampleRate*65; sample += 4096 {
				n, err := s.Read(b)
				if err != nil || n != len(b) {
					t.Fatalf("read %d: %v", n, err)
				}
				for k := 0; k < len(b); k += 4 {
					if b[k] != b[k+2] || b[k+1] != b[k+3] {
						t.Fatal("stereo mismatch")
					}
					nonzero = nonzero || b[k] != 0 || b[k+1] != 0
				}
			}
			if !nonzero {
				t.Fatal("silent track")
			}
		})
	}
}

func TestNoAllocationsAndShortReads(t *testing.T) {
	a, b := stream(t, 1, true), stream(t, 1, true)
	whole := make([]byte, 17003)
	fragmented := make([]byte, len(whole))
	if _, err := io.ReadFull(a, whole); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(fragmented); {
		size := min(i%17+1, len(fragmented)-i)
		n, err := b.Read(fragmented[i : i+size])
		if err != nil {
			t.Fatal(err)
		}
		i += n
	}
	if !bytes.Equal(whole, fragmented) {
		t.Fatal("short reads changed PCM alignment")
	}
	buffer := make([]byte, 4096*4)
	if allocations := testing.AllocsPerRun(100, func() { a.Read(buffer) }); allocations != 0 {
		t.Fatalf("audio callback allocates: %f", allocations)
	}
}

func TestCloseAndFinitePlayback(t *testing.T) {
	s := stream(t, 5, false)
	n, err := io.Copy(io.Discard, s)
	if err != nil || n <= 0 || n > int64(SampleRate*4*7) {
		t.Fatalf("finite stream: bytes=%d err=%v", n, err)
	}
	s.Close()
	s.Close()
	if n, err := s.Read(make([]byte, 4)); n != 0 || err != io.EOF {
		t.Fatal("read after close")
	}
	if _, err := New([]byte("invalid"), true); err == nil {
		t.Fatal("invalid YM accepted")
	}
}

func TestConcurrentReadAndClose(t *testing.T) {
	s := stream(t, 1, true)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 1024)
		for i := 0; i < 100; i++ {
			s.Read(buf)
		}
	}()
	s.Close()
	wg.Wait()
}
