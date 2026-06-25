// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cloud .
//go:build !plus

package nodes

import (
	"net/http"

	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/stats"
	"github.com/iwind/TeaGo/types"
)

func (this *HTTPRequest) processHTTP3Headers(respHeader http.Header) {
	if this == nil || this.nodeConfig == nil || this.ReqServer == nil {
		return
	}

	policy := this.nodeConfig.FindHTTP3PolicyWithClusterId(this.ReqServer.ClusterId)
	if policy == nil || !policy.IsOn {
		return
	}

	if !policy.SupportMobileBrowsers && this.RawReq != nil && stats.SharedUserAgentParser.Parse(this.RawReq.UserAgent()).IsMobile {
		return
	}

	port := policy.Port
	if port <= 0 {
		port = nodeconfigs.DefaultHTTP3Port
	}
	respHeader.Set("Alt-Svc", `h3=":`+types.String(port)+`"; ma=2592000`)
}
