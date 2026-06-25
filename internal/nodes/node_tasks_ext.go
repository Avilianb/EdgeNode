// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cloud .
//go:build !plus

package nodes

import (
	"encoding/json"
	"fmt"

	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
)

func (this *Node) execScriptsChangedTask() error {
	// stub
	return nil
}

func (this *Node) execUAMPolicyChangedTask(rpcClient *rpc.RPCClient) error {
	// stub
	return nil
}

func (this *Node) execHTTPCCPolicyChangedTask(rpcClient *rpc.RPCClient) error {
	// stub
	return nil
}

func (this *Node) execHTTP3PolicyChangedTask(rpcClient *rpc.RPCClient) error {
	remotelogs.Println("NODE", "updating http3 policies ...")
	resp, err := rpcClient.NodeRPC.FindNodeHTTP3Policies(rpcClient.Context(), &pb.FindNodeHTTP3PoliciesRequest{})
	if err != nil {
		return err
	}

	policyMap, err := decodeNodeHTTP3Policies(resp)
	if err != nil {
		return err
	}
	if sharedNodeConfig != nil {
		sharedNodeConfig.UpdateHTTP3Policies(policyMap)
		if sharedListenerManager != nil {
			err = sharedListenerManager.Start(sharedNodeConfig)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (this *Node) execHTTPPagesPolicyChangedTask(rpcClient *rpc.RPCClient) error {
	// stub
	return nil
}

func (this *Node) execNetworkSecurityPolicyChangedTask(rpcClient *rpc.RPCClient) error {
	// stub
	return nil
}

func (this *Node) execPlanChangedTask(rpcClient *rpc.RPCClient) error {
	return nil
}

func decodeNodeHTTP3Policies(resp *pb.FindNodeHTTP3PoliciesResponse) (map[int64]*nodeconfigs.HTTP3Policy, error) {
	policyMap := map[int64]*nodeconfigs.HTTP3Policy{}
	if resp == nil {
		return policyMap, nil
	}

	for _, policy := range resp.Http3Policies {
		if policy == nil || len(policy.Http3PolicyJSON) == 0 {
			continue
		}

		http3Policy := nodeconfigs.NewHTTP3Policy()
		err := json.Unmarshal(policy.Http3PolicyJSON, http3Policy)
		if err != nil {
			return nil, fmt.Errorf("decode http3 policy for cluster %d failed: %w", policy.NodeClusterId, err)
		}
		err = http3Policy.Init()
		if err != nil {
			return nil, fmt.Errorf("initialize http3 policy for cluster %d failed: %w", policy.NodeClusterId, err)
		}
		policyMap[policy.NodeClusterId] = http3Policy
	}
	return policyMap, nil
}
