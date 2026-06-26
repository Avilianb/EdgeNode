package nodes

import (
	"encoding/binary"
	"net"
	"sync"
	"syscall"
	"time"
)

const (
	quicVersion1                 = 1
	http3SmallInitialPacketLimit = 64
	http3InitialStateTTL         = 30 * time.Second
)

type http3PacketConn struct {
	net.PacketConn

	lock         sync.Mutex
	initialReads map[string]*http3InitialReadState
}

type http3InitialReadState struct {
	count     int
	updatedAt time.Time
}

type http3OOBPacketConn struct {
	*http3PacketConn
	conn http3OOBConn
}

type http3OOBConn interface {
	net.PacketConn
	SyscallConn() (syscall.RawConn, error)
	SetReadBuffer(int) error
	SetWriteBuffer(int) error
	ReadMsgUDP(b, oob []byte) (n, oobn, flags int, addr *net.UDPAddr, err error)
	WriteMsgUDP(b, oob []byte, addr *net.UDPAddr) (n, oobn int, err error)
}

func newHTTP3PacketConn(conn net.PacketConn) net.PacketConn {
	base := &http3PacketConn{
		PacketConn:   conn,
		initialReads: map[string]*http3InitialReadState{},
	}
	if oobConn, ok := conn.(http3OOBConn); ok {
		return &http3OOBPacketConn{
			http3PacketConn: base,
			conn:            oobConn,
		}
	}
	return base
}

func (this *http3PacketConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	n, addr, err = this.PacketConn.ReadFrom(p)
	if err == nil {
		this.noteInitialRead(p[:n], addr)
	}
	return
}

func (this *http3PacketConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	if this.shouldSuppressSmallInitial(p, addr) {
		return len(p), nil
	}
	return this.PacketConn.WriteTo(p, addr)
}

func (this *http3OOBPacketConn) SyscallConn() (syscall.RawConn, error) {
	return this.conn.SyscallConn()
}

func (this *http3OOBPacketConn) SetReadBuffer(bytes int) error {
	return this.conn.SetReadBuffer(bytes)
}

func (this *http3OOBPacketConn) SetWriteBuffer(bytes int) error {
	return this.conn.SetWriteBuffer(bytes)
}

func (this *http3OOBPacketConn) ReadMsgUDP(b, oob []byte) (n, oobn, flags int, addr *net.UDPAddr, err error) {
	n, oobn, flags, addr, err = this.conn.ReadMsgUDP(b, oob)
	if err == nil {
		this.noteInitialRead(b[:n], addr)
	}
	return
}

func (this *http3OOBPacketConn) WriteMsgUDP(b, oob []byte, addr *net.UDPAddr) (n, oobn int, err error) {
	if this.shouldSuppressSmallInitial(b, addr) {
		return len(b), 0, nil
	}
	return this.conn.WriteMsgUDP(b, oob, addr)
}

func (this *http3PacketConn) noteInitialRead(packet []byte, addr net.Addr) {
	if addr == nil || !isQUICv1InitialPacket(packet) {
		return
	}

	now := time.Now()
	key := addr.String()

	this.lock.Lock()
	defer this.lock.Unlock()

	if len(this.initialReads) > 4096 {
		this.cleanupInitialReads(now)
	}

	state := this.initialReads[key]
	if state == nil {
		state = &http3InitialReadState{}
		this.initialReads[key] = state
	}
	state.count++
	state.updatedAt = now
}

func (this *http3PacketConn) shouldSuppressSmallInitial(packet []byte, addr net.Addr) bool {
	// Chrome can split its large QUIC ClientHello over two Initial packets. If only
	// the first packet arrives, quic-go emits a tiny Initial ACK-only packet; some
	// middleboxes turn that into an immediate CONNECTION_CLOSE. Let the client
	// retransmit instead, but don't interfere once both Initial packets arrived.
	if addr == nil || !isSmallQUICv1InitialPacket(packet) {
		return false
	}

	now := time.Now()
	key := addr.String()

	this.lock.Lock()
	defer this.lock.Unlock()

	state := this.initialReads[key]
	if state == nil || now.Sub(state.updatedAt) > http3InitialStateTTL {
		return false
	}
	return state.count < 2
}

func (this *http3PacketConn) cleanupInitialReads(now time.Time) {
	for key, state := range this.initialReads {
		if now.Sub(state.updatedAt) > http3InitialStateTTL {
			delete(this.initialReads, key)
		}
	}
}

func isSmallQUICv1InitialPacket(packet []byte) bool {
	return len(packet) <= http3SmallInitialPacketLimit && isQUICv1InitialPacket(packet)
}

func isQUICv1InitialPacket(packet []byte) bool {
	if len(packet) < 6 {
		return false
	}
	if packet[0]&0xc0 != 0xc0 {
		return false
	}
	if packet[0]&0x30 != 0 {
		return false
	}
	return binary.BigEndian.Uint32(packet[1:5]) == quicVersion1
}
