package nodes

import (
	"net/http"
	"testing"

	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
)

func TestHTTPRequestProcessHTTP3HeadersAddsAltSvcForEnabledPolicy(t *testing.T) {
	header := http.Header{}
	req := &HTTPRequest{
		RawReq: &http.Request{Header: http.Header{}},
		ReqServer: &serverconfigs.ServerConfig{
			ClusterId: 123,
		},
		nodeConfig: &nodeconfigs.NodeConfig{
			HTTP3Policies: map[int64]*nodeconfigs.HTTP3Policy{
				123: {
					IsOn: true,
					Port: 8443,
				},
			},
		},
	}

	req.processHTTP3Headers(header)

	if got := header.Get("Alt-Svc"); got != `h3=":8443"; ma=2592000` {
		t.Fatalf("Alt-Svc = %q, want HTTP/3 advertisement on configured port", got)
	}
}

func TestHTTPRequestProcessHTTP3HeadersSkipsDisabledPolicy(t *testing.T) {
	header := http.Header{}
	req := &HTTPRequest{
		RawReq: &http.Request{Header: http.Header{}},
		ReqServer: &serverconfigs.ServerConfig{
			ClusterId: 123,
		},
		nodeConfig: &nodeconfigs.NodeConfig{
			HTTP3Policies: map[int64]*nodeconfigs.HTTP3Policy{
				123: {
					IsOn: false,
					Port: 8443,
				},
			},
		},
	}

	req.processHTTP3Headers(header)

	if got := header.Get("Alt-Svc"); got != "" {
		t.Fatalf("Alt-Svc = %q, want no HTTP/3 advertisement for disabled policy", got)
	}
}

func TestHTTPRequestProcessHTTP3HeadersHonorsMobileBrowserPolicy(t *testing.T) {
	mobileReq := &http.Request{Header: http.Header{}}
	mobileReq.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 12_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148")

	req := &HTTPRequest{
		RawReq: mobileReq,
		ReqServer: &serverconfigs.ServerConfig{
			ClusterId: 123,
		},
		nodeConfig: &nodeconfigs.NodeConfig{
			HTTP3Policies: map[int64]*nodeconfigs.HTTP3Policy{
				123: {
					IsOn:                  true,
					Port:                  443,
					SupportMobileBrowsers: false,
				},
			},
		},
	}

	header := http.Header{}
	req.processHTTP3Headers(header)
	if got := header.Get("Alt-Svc"); got != "" {
		t.Fatalf("Alt-Svc = %q, want no HTTP/3 advertisement for mobile browsers by default", got)
	}

	req.nodeConfig.HTTP3Policies[123].SupportMobileBrowsers = true
	req.processHTTP3Headers(header)
	if got := header.Get("Alt-Svc"); got != `h3=":443"; ma=2592000` {
		t.Fatalf("Alt-Svc = %q, want HTTP/3 advertisement when mobile browsers are enabled", got)
	}
}
