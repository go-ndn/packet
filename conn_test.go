package packet

import (
	"bytes"
	"io"
	"testing"
	"time"
)

var (
	clientMsg = []byte("client")
	serverMsg = []byte("server")
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

	// start client
	client, err := Dial("udp", port)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	// start server
	server, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	// SetDeadline does nothing
	now := time.Now()
	client.SetDeadline(now)
	client.SetReadDeadline(now)
	client.SetWriteDeadline(now)

	// write something
	client.Write(clientMsg)
	server.Write(serverMsg)

	buf := make([]byte, 8)
	for _, test := range []struct {
		io.Reader
		want []byte
	}{
		{
			Reader: client,
			want:   serverMsg,
		},
		{
			Reader: server,
			want:   clientMsg,
		},
	} {
		n, err := test.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(test.want, buf[:n]) {
			t.Fatalf("expect %s, got %s", test.want, buf[:n])
		}
	}
}
