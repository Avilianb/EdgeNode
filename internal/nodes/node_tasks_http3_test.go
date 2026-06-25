package nodes

import (
	"testing"

	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
)

func TestDecodeNodeHTTP3Policies(t *testing.T) {
	policies, err := decodeNodeHTTP3Policies(&pb.FindNodeHTTP3PoliciesResponse{
		Http3Policies: []*pb.FindNodeHTTP3PoliciesResponse_HTTP3Policy{
			{
				NodeClusterId:   1,
				Http3PolicyJSON: []byte(`{"isOn":true,"port":8443,"supportMobileBrowsers":true}`),
			},
			{
				NodeClusterId:   2,
				Http3PolicyJSON: []byte(`{"isOn":false}`),
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(policies) != 2 {
		t.Fatalf("policies count = %d, want 2", len(policies))
	}
	if policy := policies[1]; policy == nil || !policy.IsOn || policy.Port != 8443 || !policy.SupportMobileBrowsers {
		t.Fatalf("cluster 1 policy = %#v, want enabled port 8443 with mobile support", policy)
	}
	if policy := policies[2]; policy == nil || policy.IsOn || policy.Port != 443 {
		t.Fatalf("cluster 2 policy = %#v, want disabled default-port policy", policy)
	}
}

func TestDecodeNodeHTTP3PoliciesRejectsInvalidJSON(t *testing.T) {
	_, err := decodeNodeHTTP3Policies(&pb.FindNodeHTTP3PoliciesResponse{
		Http3Policies: []*pb.FindNodeHTTP3PoliciesResponse_HTTP3Policy{
			{
				NodeClusterId:   1,
				Http3PolicyJSON: []byte(`{"isOn":`),
			},
		},
	})
	if err == nil {
		t.Fatal("expected invalid HTTP/3 policy JSON to fail")
	}
}
