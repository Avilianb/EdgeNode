package nodes

import (
	"context"
	"io"
	"log"
	"net"
	"strconv"
	"testing"

	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/sslconfigs"
)

func TestListenerManagerStartsAndStopsHTTP3Listeners(t *testing.T) {
	suppressStandardLogForTest(t)

	tcpPort := freeTCPPort(t)
	http3Port := freeUDPPort(t)
	manager := NewListenerManager()
	t.Cleanup(func() {
		closeListenerManagerForTest(manager)
	})

	err := manager.Start(testHTTP3NodeConfig(t, tcpPort, http3Port, true))
	if err != nil {
		t.Fatal(err)
	}

	if manager.http3Listeners[http3Port] == nil {
		t.Fatalf("HTTP/3 listener for UDP port %d was not started", http3Port)
	}

	err = manager.Start(testHTTP3NodeConfig(t, tcpPort, http3Port, false))
	if err != nil {
		t.Fatal(err)
	}

	if len(manager.http3Listeners) != 0 {
		t.Fatalf("HTTP/3 listeners = %d, want none after disabling policy", len(manager.http3Listeners))
	}
}

func TestListenerManagerRebuildsHTTP3ListenersWhenPortChanges(t *testing.T) {
	suppressStandardLogForTest(t)

	tcpPort := freeTCPPort(t)
	oldHTTP3Port := freeUDPPort(t)
	newHTTP3Port := freeUDPPort(t)
	manager := NewListenerManager()
	t.Cleanup(func() {
		closeListenerManagerForTest(manager)
	})

	err := manager.Start(testHTTP3NodeConfig(t, tcpPort, oldHTTP3Port, true))
	if err != nil {
		t.Fatal(err)
	}
	err = manager.Start(testHTTP3NodeConfig(t, tcpPort, newHTTP3Port, true))
	if err != nil {
		t.Fatal(err)
	}

	if manager.http3Listeners[oldHTTP3Port] != nil {
		t.Fatalf("HTTP/3 listener for old UDP port %d is still running", oldHTTP3Port)
	}
	if manager.http3Listeners[newHTTP3Port] == nil {
		t.Fatalf("HTTP/3 listener for new UDP port %d was not started", newHTTP3Port)
	}
}

func testHTTP3NodeConfig(t *testing.T, tcpPort int, http3Port int, http3IsOn bool) *nodeconfigs.NodeConfig {
	t.Helper()

	config := &nodeconfigs.NodeConfig{
		IsOn: true,
		Servers: []*serverconfigs.ServerConfig{
			{
				Id:        1,
				ClusterId: 77,
				IsOn:      true,
				HTTPS: &serverconfigs.HTTPSProtocolConfig{
					BaseProtocol: serverconfigs.BaseProtocol{
						IsOn: true,
						Listen: []*serverconfigs.NetworkAddressConfig{
							{
								Protocol:  serverconfigs.ProtocolHTTPS,
								Host:      "127.0.0.1",
								PortRange: strconv.Itoa(tcpPort),
							},
						},
					},
					SSLPolicy: &sslconfigs.SSLPolicy{
						IsOn:         true,
						HTTP2Enabled: true,
						HTTP3Enabled: true,
					},
				},
			},
		},
		HTTP3Policies: map[int64]*nodeconfigs.HTTP3Policy{
			77: {
				IsOn: http3IsOn,
				Port: http3Port,
			},
		},
	}
	err, serverErrors := config.Init(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(serverErrors) > 0 {
		t.Fatalf("unexpected server config errors: %#v", serverErrors)
	}
	return config
}

func freeTCPPort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	return listener.Addr().(*net.TCPAddr).Port
}

func freeUDPPort(t *testing.T) int {
	t.Helper()

	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	return conn.LocalAddr().(*net.UDPAddr).Port
}

func closeListenerManagerForTest(manager *ListenerManager) {
	if manager == nil {
		return
	}
	for _, listener := range manager.listenersMap {
		_ = listener.Close()
	}
	for _, listener := range manager.http3Listeners {
		_ = listener.Close()
	}
}

func suppressStandardLogForTest(t *testing.T) {
	t.Helper()

	oldWriter := log.Writer()
	log.SetOutput(io.Discard)
	t.Cleanup(func() {
		log.SetOutput(oldWriter)
	})
}
