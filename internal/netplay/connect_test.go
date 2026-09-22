//go:build integration

package netplay

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"

	"bubblebobble/internal/game"
)

func TestHostAuthenticatesOnlyItsNativeProxy(t *testing.T) {
	token := bytes.Repeat([]byte{42}, 16)
	p, address, err := Host(token, "campaign", "BUB")
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	wrong, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	wrong.Write(bytes.Repeat([]byte{0}, 16))
	wrong.SetReadDeadline(time.Now().Add(time.Second))
	if _, err = wrong.Read(make([]byte, 1)); err == nil {
		t.Fatal("wrong proxy token was accepted")
	}
	wrong.Close()
	conn, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = conn.Write(token); err != nil {
		t.Fatal(err)
	}
	guest := New(conn, false, "campaign", "BOB")
	defer guest.Close()
	var host *Peer
	waitFor(t, func() bool { host, err = p.Ready(); return host != nil || err != nil })
	if err != nil {
		t.Fatal(err)
	}
	guest.SendInput(game.Input{Move: 1})
	waitFor(t, func() bool { return host.Input(time.Now()).Move == 1 })
	p.Close()
	host.Wait()
	guest.Wait()
}

func TestJoinAuthenticatesBeforeSendingTheGameProtocol(t *testing.T) {
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	token := bytes.Repeat([]byte{9}, 16)
	p := Join(l.Addr().String(), token, "campaign", "BOB")
	defer p.Close()
	conn, err := l.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(time.Second))
	received := make([]byte, 16)
	if _, err = io.ReadFull(conn, received); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(token, received) {
		t.Fatal("proxy did not receive its token first")
	}
	conn.SetReadDeadline(time.Time{})
	host := New(conn, true, "campaign", "BUB")
	defer host.Close()
	var guest *Peer
	waitFor(t, func() bool { guest, err = p.Ready(); return guest != nil || err != nil })
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { ready, _, _ := guest.Status(); return ready })
	p.Close()
	guest.Wait()
	host.Wait()
}
