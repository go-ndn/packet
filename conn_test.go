package packet

import (
	"bytes"
	"testing"
	"time"
)

var (
	msg = []byte{1, 2, 3}
)

const (
	port = ":3000"
)

func TestConn(t *testing.T) {
	ln, err := Listen("udp", port)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	t.Log(ln.Addr())

	conn1, err := Dial("udp", port)
	if err != nil {
		t.Fatal(err)
	}
	defer conn1.Close()

	t.Log(conn1.LocalAddr())
	t.Log(conn1.RemoteAddr())
	now := time.Now()
	conn1.SetDeadline(now)
	conn1.SetReadDeadline(now)
	conn1.SetWriteDeadline(now)
	conn1.Write(msg)

	conn2, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer conn2.Close()

	time.Sleep(2 * Heartbeat)

	buf := make([]byte, len(msg))
	n, _ := conn2.Read(buf)
	if !bytes.Equal(msg, buf[:n]) {
		t.Fatalf("expected %v, got %v", msg, buf[:n])
	}
}
