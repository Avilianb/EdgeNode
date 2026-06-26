package nodes

import (
	"errors"
	"net"
	"net/http"

	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils/goman"
	"github.com/iwind/TeaGo/types"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

// HTTP3Listener listens for HTTP/3 over QUIC on one UDP port.
type HTTP3Listener struct {
	HTTPListener

	port       int
	packetConn net.PacketConn
	h3Server   *http3.Server
	h3Listener http3.QUICEarlyListener
	transport  *quic.Transport
}

func NewHTTP3Listener(port int, group *serverconfigs.ServerAddressGroup) *HTTP3Listener {
	listener := &HTTP3Listener{
		HTTPListener: HTTPListener{
			BaseListener: BaseListener{Group: group},
			addr:         ":" + types.String(port),
			isHTTPS:      true,
			isHTTP3:      true,
		},
		port: port,
	}
	return listener
}

func (this *HTTP3Listener) Listen() error {
	packetConn, err := net.ListenPacket("udp", this.addr)
	if err != nil {
		return err
	}
	this.packetConn = newHTTP3PacketConn(packetConn)
	this.h3Server = &http3.Server{
		Addr:      this.addr,
		Port:      this.port,
		Handler:   this,
		TLSConfig: this.buildTLSConfig(),
	}
	this.transport = newHTTP3Transport(this.packetConn)
	h3Listener, err := this.transport.ListenEarly(http3.ConfigureTLSConfig(this.h3Server.TLSConfig), newHTTP3QUICConfig())
	if err != nil {
		_ = this.packetConn.Close()
		return err
	}
	this.h3Listener = h3Listener

	goman.New(func() {
		err := this.h3Server.ServeListener(this.h3Listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, net.ErrClosed) {
			remotelogs.Error("HTTP3_LISTENER", err.Error())
		}
	})

	return nil
}

func (this *HTTP3Listener) Close() error {
	if this.h3Server != nil {
		_ = this.h3Server.Close()
	}
	if this.h3Listener != nil {
		_ = this.h3Listener.Close()
	}
	if this.transport != nil {
		_ = this.transport.Close()
	}
	if this.packetConn != nil {
		return this.packetConn.Close()
	}
	return nil
}

func (this *HTTP3Listener) Reload(group *serverconfigs.ServerAddressGroup) {
	this.HTTPListener.Reload(group)
	this.HTTPListener.addr = ":" + types.String(this.port)
	this.HTTPListener.isHTTP = false
	this.HTTPListener.isHTTPS = true
	this.HTTPListener.isHTTP3 = true
}

func (this *ListenerManager) reloadHTTP3Listeners(nodeConfig *nodeconfigs.NodeConfig) {
	var desiredPorts = map[int]*serverconfigs.ServerAddressGroup{}
	if nodeConfig != nil && nodeConfig.IsOn {
		for _, port := range nodeConfig.FindHTTP3Ports() {
			group := this.http3GroupForPort(nodeConfig, port)
			if group != nil && len(group.Servers()) > 0 {
				desiredPorts[port] = group
			}
		}
	}

	for port, listener := range this.http3Listeners {
		if desiredPorts[port] == nil {
			remotelogs.Println("HTTP3_LISTENER", "close udp://:"+types.String(port))
			_ = listener.Close()
			delete(this.http3Listeners, port)
		}
	}

	for port, group := range desiredPorts {
		listener := this.http3Listeners[port]
		if listener != nil {
			listener.Reload(group)
			continue
		}

		remotelogs.Println("HTTP3_LISTENER", "listen udp://:"+types.String(port))
		listener = NewHTTP3Listener(port, group)
		err := listener.Listen()
		if err != nil {
			firstServer := group.FirstServer()
			if firstServer != nil {
				remotelogs.ServerError(firstServer.Id, "HTTP3_LISTENER", "listen udp://:"+types.String(port)+" failed: "+err.Error(), nodeconfigs.NodeLogTypeListenAddressFailed, nil)
			} else {
				remotelogs.Error("HTTP3_LISTENER", err.Error())
			}
			continue
		}
		this.http3Listeners[port] = listener
	}
}

func newHTTP3Transport(packetConn net.PacketConn) *quic.Transport {
	return &quic.Transport{
		Conn: packetConn,
		VerifySourceAddress: func(net.Addr) bool {
			return true
		},
	}
}

func newHTTP3QUICConfig() *quic.Config {
	return &quic.Config{Allow0RTT: true}
}

func (this *ListenerManager) http3GroupForPort(nodeConfig *nodeconfigs.NodeConfig, port int) *serverconfigs.ServerAddressGroup {
	group := serverconfigs.NewServerAddressGroup("HTTP3")
	for _, server := range nodeConfig.Servers {
		if server == nil || !server.SupportsHTTP3() {
			continue
		}

		policy := nodeConfig.FindHTTP3PolicyWithClusterId(server.ClusterId)
		if policy == nil || !policy.IsOn {
			continue
		}
		policyPort := policy.Port
		if policyPort <= 0 {
			policyPort = nodeconfigs.DefaultHTTP3Port
		}
		if policyPort == port {
			group.Add(server)
		}
	}
	return group
}
