package httpclient

import (
	"net"
	"net/http"
	"time"
)

// External 访问外网 URL 的 HTTP 客户端（尊重 HTTP_PROXY / HTTPS_PROXY）
var External = &http.Client{
	Timeout: 60 * time.Second,
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   15 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConns:          32,
		IdleConnTimeout:       90 * time.Second,
	},
}
