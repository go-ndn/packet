package packet

import (
	"bytes"
	"io"
	"testing"
	"time"
)

func TestUDPConn(t *testing.T) {
	testConn(t, "udp", ":6363")
}

func TestMulticastUDP(t *testing.T) {
	testConn(t, "udp", "224.0.23.170:56363")
}

func testConn(t *testing.T, network, address string) {
	ln, err := Listen(network, address)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	// start client
	client, err := Dial(network, address)
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

	var (
		clientMsg = []byte("FROM_CLIENT")
		serverMsg = []byte("FROM_SERVER")
	)

	// write something
	client.Write(clientMsg)
	server.Write(serverMsg)

	buf := make([]byte, 32)
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
