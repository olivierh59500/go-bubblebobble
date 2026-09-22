package netplay

import (
	"context"
	"crypto/subtle"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// Pending owns connection resources, including the established peer, until Close.
// The game thread can use Ready while Android cancels the same attempt safely.
type Pending struct {
	mu       sync.Mutex
	listener net.Listener
	conn     net.Conn
	peer     *Peer
	err      error
	closed   bool
	cancel   context.CancelFunc
}

func Host(token []byte, content, name string) (*Pending, string, error) {
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, "", err
	}
	p := &Pending{listener: l}
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				p.finish(nil, err)
				return
			}
			p.mu.Lock()
			if p.closed {
				p.mu.Unlock()
				conn.Close()
				return
			}
			p.conn = conn
			p.mu.Unlock()
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			supplied := make([]byte, 16)
			_, err = io.ReadFull(conn, supplied)
			if err != nil || len(token) != 16 || subtle.ConstantTimeCompare(supplied, token) != 1 {
				conn.Close()
				continue
			}
			conn.SetReadDeadline(time.Time{})
			l.Close()
			p.finish(New(conn, true, content, name), nil)
			return
		}
	}()
	return p, l.Addr().String(), nil
}

func Join(address string, token []byte, content, name string) *Pending {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	p := &Pending{cancel: cancel}
	go func() {
		defer cancel()
		conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", address)
		if err != nil {
			p.finish(nil, err)
			return
		}
		p.mu.Lock()
		if p.closed {
			p.mu.Unlock()
			conn.Close()
			return
		}
		p.conn = conn
		p.mu.Unlock()
		conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if len(token) != 16 {
			err = fmt.Errorf("invalid proxy token")
		} else {
			err = writeAll(conn, token)
		}
		if err != nil {
			conn.Close()
			p.finish(nil, err)
			return
		}
		conn.SetWriteDeadline(time.Time{})
		p.finish(New(conn, false, content, name), nil)
	}()
	return p
}

func (p *Pending) finish(peer *Peer, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		if peer != nil {
			peer.Close()
		}
		return
	}
	p.peer, p.err = peer, err
}

func (p *Pending) Ready() (*Peer, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	peer, err := p.peer, p.err
	return peer, err
}
func (p *Pending) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	if p.cancel != nil {
		p.cancel()
	}
	if p.listener != nil {
		p.listener.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
	if p.peer != nil {
		p.peer.Close()
	}
}
