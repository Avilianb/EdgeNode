package nodes

import (
	"encoding/hex"
	"net"
	"testing"
)

var (
	chromeClientInitial1 = mustHex("c8000000010842cbd43cb4ec9385000044d056f41aadb4df111ac1bd88943255")
	chromeClientInitial2 = mustHex("ce000000010842cbd43cb4ec9385000044d00cb3d32a54e33054715deee2a4aa")
	serverInitialAck     = mustHex("cc0000000100047c304a92004017807cbeb504f74a57e1e321cb4bcf1aadf718")
	serverHandshake      = mustHex("c80000000100047c304a92004075fa164a0f52fa628291ff56131e3b3c95c875")
	shortHeaderPacket    = mustHex("5ae92aea5df81bb688a03ce887907d27418ab10c872a98ce91c4")
)

func TestHTTP3PacketConnSuppressesSmallInitialAfterSingleClientInitial(t *testing.T) {
	conn := newTestHTTP3PacketConn()
	addr := &net.UDPAddr{IP: net.ParseIP("203.0.113.10"), Port: 44321}

	conn.noteInitialRead(chromeClientInitial1, addr)

	if !conn.shouldSuppressSmallInitial(serverInitialAck, addr) {
		t.Fatal("expected small server Initial to be suppressed after one client Initial")
	}
}

func TestHTTP3PacketConnAllowsSmallInitialAfterSecondClientInitial(t *testing.T) {
	conn := newTestHTTP3PacketConn()
	addr := &net.UDPAddr{IP: net.ParseIP("203.0.113.10"), Port: 44321}

	conn.noteInitialRead(chromeClientInitial1, addr)
	conn.noteInitialRead(chromeClientInitial2, addr)

	if conn.shouldSuppressSmallInitial(serverInitialAck, addr) {
		t.Fatal("expected small server Initial to be allowed after two client Initial packets")
	}
}

func TestHTTP3PacketConnDoesNotSuppressUnrelatedPackets(t *testing.T) {
	conn := newTestHTTP3PacketConn()
	addr := &net.UDPAddr{IP: net.ParseIP("203.0.113.10"), Port: 44321}

	conn.noteInitialRead(chromeClientInitial1, addr)

	largeServerHandshake := append(append([]byte(nil), serverHandshake...), make([]byte, http3SmallInitialPacketLimit)...)
	if conn.shouldSuppressSmallInitial(largeServerHandshake, addr) {
		t.Fatal("expected large server Initial handshake packet to be allowed")
	}
	if conn.shouldSuppressSmallInitial(shortHeaderPacket, addr) {
		t.Fatal("expected short header packet to be allowed")
	}
	if conn.shouldSuppressSmallInitial(serverInitialAck, &net.UDPAddr{IP: net.ParseIP("203.0.113.11"), Port: 44321}) {
		t.Fatal("expected packet for an unknown address to be allowed")
	}
}

func newTestHTTP3PacketConn() *http3PacketConn {
	return &http3PacketConn{
		initialReads: map[string]*http3InitialReadState{},
	}
}

func mustHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}
