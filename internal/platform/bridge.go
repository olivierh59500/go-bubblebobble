// Package platform contains the thread-safe boundary to Android callbacks.
package platform

import (
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

type Event struct {
	Generation          int64
	Kind, Text, Address string
}

type Bridge struct {
	Available  atomic.Bool
	InputReset atomic.Uint64
	Back       atomic.Bool
	Pause      atomic.Bool
	mu         sync.Mutex
	generation int64
	active     bool
	commands   []string
	events     []Event
	stop       func()
	pause      func()
}

func (b *Bridge) Begin() int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.generation++
	b.active = true
	b.events = nil
	return b.generation
}
func (b *Bridge) Queue(command string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.commands) < 16 {
		b.commands = append(b.commands, command)
	}
}
func (b *Bridge) Poll() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.commands) == 0 {
		return ""
	}
	c := b.commands[0]
	b.commands = b.commands[1:]
	return c
}
func (b *Bridge) Post(e Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.active || e.Generation != b.generation {
		return
	}
	e.Text = Clean(e.Text)
	if len(b.events) < 32 {
		b.events = append(b.events, e)
	}
}
func (b *Bridge) Events() []Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	e := b.events
	b.events = nil
	return e
}
func (b *Bridge) BindCancel(stop func()) { b.mu.Lock(); b.stop = stop; b.mu.Unlock() }
func (b *Bridge) BindPause(pause func()) { b.mu.Lock(); b.pause = pause; b.mu.Unlock() }
func (b *Bridge) Stop() {
	b.mu.Lock()
	stop := b.stop
	b.stop = nil
	b.pause = nil
	b.active = false
	b.events = nil
	b.commands = []string{fmt.Sprintf("BT_CANCEL|%d", b.generation)}
	b.mu.Unlock()
	if stop != nil {
		stop()
	}
}
func (b *Bridge) CancelFromPlatform() {
	b.mu.Lock()
	stop := b.stop
	generation := b.generation
	b.mu.Unlock()
	if stop != nil {
		stop()
	}
	b.Post(Event{Generation: generation, Kind: "failure", Text: "Bluetooth connection closed"})
}
func (b *Bridge) Suspend() {
	b.InputReset.Add(1)
	b.Pause.Store(true)
	b.mu.Lock()
	pause := b.pause
	b.mu.Unlock()
	if pause != nil {
		pause()
	}
}

func Loopback(address string) (string, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return "", fmt.Errorf("invalid Bluetooth endpoint")
	}
	ip, err := netip.ParseAddr(host)
	if err != nil || !ip.IsLoopback() {
		return "", fmt.Errorf("Bluetooth endpoint must be local")
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return "", fmt.Errorf("invalid Bluetooth port")
	}
	return net.JoinHostPort(ip.String(), strconv.Itoa(n)), nil
}

func Clean(text string) string {
	text = strings.Map(func(r rune) rune {
		if r < 32 || r > 126 {
			return ' '
		}
		return r
	}, text)
	text = strings.Join(strings.Fields(text), " ")
	if len(text) > 160 {
		text = text[:160]
	}
	return text
}
