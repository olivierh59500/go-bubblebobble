// Package ym adapts the YM2149 synthesizer to Ebitengine's PCM reader.
package ym

import (
	"fmt"
	"io"
	"sync"

	"github.com/olivierh59500/ym-player/pkg/stsound"
)

const SampleRate = 48000

type Stream struct {
	mu               sync.Mutex
	synth            *stsound.StSound
	mono             [4096]int16
	monoPos, monoLen int
	tail             [4]byte
	start, end       int
}

func New(data []byte, loop bool) (*Stream, error) {
	s := stsound.CreateWithRate(SampleRate)
	if err := s.LoadMemory(data); err != nil {
		s.Destroy()
		return nil, fmt.Errorf("load YM: %w", err)
	}
	s.SetLoopMode(loop)
	return &Stream{synth: s}, nil
}

// Read writes signed 16-bit little-endian stereo without allocating. A partial
// frame is retained so even byte-sized reads preserve sample boundaries.
func (s *Stream) Read(dst []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(dst) == 0 {
		return 0, nil
	}
	if s.synth == nil {
		return 0, io.EOF
	}
	n := 0
	if s.start < s.end {
		count := copy(dst, s.tail[s.start:s.end])
		s.start += count
		n += count
	}
	for len(dst)-n >= 4 {
		if !s.fill() {
			return n, io.EOF
		}
		frames := min((len(dst)-n)/4, s.monoLen-s.monoPos)
		for _, value := range s.mono[s.monoPos : s.monoPos+frames] {
			v := value / 2
			dst[n], dst[n+1], dst[n+2], dst[n+3] = byte(v), byte(v>>8), byte(v), byte(v>>8)
			n += 4
		}
		s.monoPos += frames
	}
	if n < len(dst) {
		if !s.fill() {
			return n, io.EOF
		}
		v := s.mono[s.monoPos] / 2
		s.monoPos++
		s.tail = [4]byte{byte(v), byte(v >> 8), byte(v), byte(v >> 8)}
		s.start = copy(dst[n:], s.tail[:])
		s.end = 4
		n += s.start
	}
	return n, nil
}

func (s *Stream) fill() bool {
	if s.monoPos < s.monoLen {
		return true
	}
	if !s.synth.Compute(s.mono[:], len(s.mono)) {
		return false
	}
	s.monoPos, s.monoLen = 0, len(s.mono)
	return true
}

func (s *Stream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.synth != nil {
		s.synth.Destroy()
		s.synth = nil
	}
	s.start, s.end = 0, 0
	return nil
}
