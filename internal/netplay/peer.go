// Package netplay carries a cooperative match over a reliable byte stream.
// Bluetooth RFCOMM is supplied by Android; tests use the same protocol in memory.
package netplay

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"bubblebobble/internal/game"
)

const Protocol = "bubblebobble-coop-1"
const maxWireSize = 64 * 1024
const maxFrameSize = 256 * 1024

type compressor struct {
	buffer bytes.Buffer
	writer *zlib.Writer
}

var compressors = sync.Pool{New: func() any { c := &compressor{}; c.writer, _ = zlib.NewWriterLevel(&c.buffer, zlib.BestSpeed); return c }}

type frame struct {
	Kind     string         `json:"k"`
	Protocol string         `json:"v,omitempty"`
	Content  string         `json:"c,omitempty"`
	Name     string         `json:"n,omitempty"`
	Host     bool           `json:"h,omitempty"`
	Run      int            `json:"r,omitempty"`
	Input    *game.Input    `json:"i,omitempty"`
	State    *game.Snapshot `json:"s,omitempty"`
	Control  string         `json:"a,omitempty"`
}

type Update struct {
	Run      int
	Snapshot game.Snapshot
}

type Peer struct {
	conn                         net.Conn
	host                         bool
	mu                           sync.Mutex
	ready                        bool
	name                         string
	err                          error
	input                        game.Input
	inputAt                      time.Time
	latest                       *Update
	finals                       []*Update
	lastRun, lastTick, lastFinal int
	controls                     []string
	commands                     chan frame
	states                       chan frame
	done                         chan struct{}
	once                         sync.Once
	wg                           sync.WaitGroup
}

func New(conn net.Conn, host bool, content, name string) *Peer {
	p := &Peer{conn: conn, host: host, commands: make(chan frame, 16), states: make(chan frame, 1), done: make(chan struct{})}
	p.wg.Add(2)
	go p.read(content)
	go p.write(frame{Kind: "hello", Protocol: Protocol, Content: content, Host: host, Name: name})
	return p
}

func (p *Peer) Status() (ready bool, name string, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.ready, p.name, p.err
}
func (p *Peer) Done() <-chan struct{} { return p.done }
func (p *Peer) Close()                { p.fail(io.EOF) }
func (p *Peer) Wait()                 { p.wg.Wait() }

func (p *Peer) fail(err error) {
	p.once.Do(func() {
		p.mu.Lock()
		p.err = err
		p.input = game.Input{}
		p.mu.Unlock()
		close(p.done)
		p.conn.Close()
	})
}

func (p *Peer) Input(now time.Time) game.Input {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.err != nil || now.Sub(p.inputAt) > 250*time.Millisecond {
		return game.Input{}
	}
	return p.input
}

func (p *Peer) TakeUpdate() *Update {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.finals) > 0 {
		v := p.finals[0]
		p.finals = p.finals[1:]
		return v
	}
	v := p.latest
	p.latest = nil
	return v
}
func (p *Peer) TakeControls() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	v := p.controls
	p.controls = nil
	return v
}

func (p *Peer) SendInput(in game.Input) {
	if !p.host {
		p.replace(frame{Kind: "input", Input: &in})
	}
}
func (p *Peer) SendState(run int, s game.Snapshot) {
	if p.host {
		f := frame{Kind: "state", State: &s, Run: run}
		if s.State == game.Won || s.State == game.GameOver {
			p.reliable(f)
		} else {
			p.replace(f)
		}
	}
}
func (p *Peer) Control(action string) {
	p.reliable(frame{Kind: "control", Control: action})
}
func (p *Peer) reliable(f frame) {
	select {
	case <-p.done:
		return
	default:
	}
	select {
	case p.commands <- f:
	case <-p.done:
	default:
		p.fail(fmt.Errorf("connection is too slow"))
	}
}

// Retain only the freshest state or input while Bluetooth is busy. Simulation
// never waits for a socket and a stalled peer cannot accumulate an old backlog.
func (p *Peer) replace(f frame) {
	select {
	case <-p.done:
		return
	default:
	}
	select {
	case p.states <- f:
	default:
		select {
		case <-p.states:
		default:
		}
		select {
		case p.states <- f:
		case <-p.done:
		default:
		}
	}
}

func (p *Peer) write(hello frame) {
	defer p.wg.Done()
	if err := p.send(hello); err != nil {
		p.fail(err)
		return
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		var f frame
		select {
		case f = <-p.commands:
		default:
			select {
			case <-p.done:
				return
			case f = <-p.commands:
			case f = <-p.states:
			case <-ticker.C:
				f = frame{Kind: "ping"}
			}
		}
		if err := p.send(f); err != nil {
			p.fail(err)
			return
		}
	}
}

func (p *Peer) send(f frame) error {
	if err := p.conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return err
	}
	return writeFrame(p.conn, f)
}

func (p *Peer) read(content string) {
	defer p.wg.Done()
	p.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	hello, err := readFrame(p.conn)
	if err != nil {
		p.fail(err)
		return
	}
	if hello.Kind != "hello" || hello.Protocol != Protocol || hello.Content != content || hello.Host == p.host || len(hello.Name) > 16 {
		p.fail(fmt.Errorf("the two games have incompatible versions"))
		return
	}
	p.mu.Lock()
	p.ready = true
	p.name = hello.Name
	p.mu.Unlock()
	for {
		p.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		f, err := readFrame(p.conn)
		if err != nil {
			p.fail(err)
			return
		}
		if err = p.accept(f); err != nil {
			p.fail(err)
			return
		}
	}
}

func (p *Peer) accept(f frame) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	switch f.Kind {
	case "ping":
		return nil
	case "input":
		if !p.host || f.Input == nil || f.Input.Move < -1 || f.Input.Move > 1 {
			return fmt.Errorf("invalid remote input")
		}
		p.input = *f.Input
		p.inputAt = time.Now()
	case "state":
		if p.host || f.State == nil || f.Run < 1 {
			return fmt.Errorf("invalid host frame")
		}
		if err := f.State.Validate(); err != nil {
			return err
		}
		if f.Run < p.lastRun || f.Run == p.lastRun && f.State.Tick < p.lastTick {
			return nil
		}
		p.lastRun, p.lastTick = f.Run, f.State.Tick
		update := &Update{Run: f.Run, Snapshot: *f.State}
		if f.State.State == game.Won || f.State.State == game.GameOver {
			if f.Run > p.lastFinal {
				if len(p.finals) >= 4 {
					return fmt.Errorf("too many unfinished rounds")
				}
				p.finals = append(p.finals, update)
				p.lastFinal = f.Run
				p.latest = nil
			}
		} else if f.Run > p.lastFinal {
			p.latest = update
		}
	case "control":
		if f.Control != "pause" && f.Control != "resume" {
			return fmt.Errorf("invalid remote command")
		}
		if len(p.controls) >= 16 {
			return fmt.Errorf("too many remote commands")
		}
		p.controls = append(p.controls, f.Control)
	default:
		return fmt.Errorf("unknown multiplayer message")
	}
	return nil
}

func writeFrame(w io.Writer, f frame) error {
	data, err := json.Marshal(f)
	if err != nil {
		return err
	}
	if len(data) > maxFrameSize {
		return fmt.Errorf("multiplayer frame too large")
	}
	c := compressors.Get().(*compressor)
	defer compressors.Put(c)
	c.buffer.Reset()
	c.writer.Reset(&c.buffer)
	packed, z := &c.buffer, c.writer
	if _, err = z.Write(data); err != nil {
		return err
	}
	if err = z.Close(); err != nil {
		return err
	}
	if packed.Len() > maxWireSize {
		return fmt.Errorf("multiplayer packet too large")
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(packed.Len()))
	if err = writeAll(w, header[:]); err != nil {
		return err
	}
	return writeAll(w, packed.Bytes())
}

func writeAll(w io.Writer, b []byte) error {
	for len(b) > 0 {
		n, err := w.Write(b)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		b = b[n:]
	}
	return nil
}

func readFrame(r io.Reader) (frame, error) {
	var f frame
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return f, err
	}
	n := binary.BigEndian.Uint32(header[:])
	if n == 0 || n > maxWireSize {
		return f, fmt.Errorf("invalid multiplayer packet size")
	}
	packed := make([]byte, n)
	if _, err := io.ReadFull(r, packed); err != nil {
		return f, err
	}
	z, err := zlib.NewReader(bytes.NewReader(packed))
	if err != nil {
		return f, err
	}
	defer z.Close()
	data, err := io.ReadAll(io.LimitReader(z, maxFrameSize+1))
	if err != nil {
		return f, err
	}
	if len(data) > maxFrameSize {
		return f, fmt.Errorf("expanded frame too large")
	}
	err = json.Unmarshal(data, &f)
	return f, err
}
