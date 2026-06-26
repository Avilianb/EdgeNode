package nodes

import (
	"net"
	"testing"
)

func TestHTTP3TransportRequiresRetry(t *testing.T) {
	transport := newHTTP3Transport(nil)
	if transport == nil {
		t.Fatal("expected HTTP/3 transport")
	}
	if transport.VerifySourceAddress == nil {
		t.Fatal("expected HTTP/3 transport to require source address validation")
	}
	if !transport.VerifySourceAddress(&net.UDPAddr{IP: net.ParseIP("203.0.113.10"), Port: 44321}) {
		t.Fatal("expected HTTP/3 transport to request Retry for unvalidated clients")
	}
}

func TestHTTP3QUICConfigKeeps0RTTEnabled(t *testing.T) {
	config := newHTTP3QUICConfig()
	if config == nil {
		t.Fatal("expected HTTP/3 QUIC config")
	}
	if !config.Allow0RTT {
		t.Fatal("expected HTTP/3 QUIC config to keep 0-RTT enabled")
	}
}
