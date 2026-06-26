package nodes

import (
	"encoding/hex"
	"net"
	"testing"

	"golang.org/x/net/ipv4"
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

func TestHTTP3PacketConnOOBWrapperSupportsIPv4PacketConn(t *testing.T) {
	udpConn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer udpConn.Close()

	wrapped := newHTTP3PacketConn(udpConn)

	defer func() {
		if err := recover(); err != nil {
			t.Fatalf("wrapped UDP conn should support ipv4.NewPacketConn: %v", err)
		}
	}()
	if ipv4.NewPacketConn(wrapped) == nil {
		t.Fatal("expected ipv4 packet conn")
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
