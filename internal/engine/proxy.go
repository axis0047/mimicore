package engine

import (
	"context"
	"net"
	"net/http"
	"net/http/httputil"
)

type ProxyHandler struct {
	proxy *httputil.ReverseProxy
}

func NewUnixProxy(socket string) *ProxyHandler {
	tr := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", socket)
		},
	}

	p := &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			r.URL.Scheme = "http"
			r.URL.Host = "unix"
		},
		Transport: tr,
	}

	return &ProxyHandler{proxy: p}
}

func (p *ProxyHandler) Serve(ctx *Context) {
	p.proxy.ServeHTTP(ctx.W, ctx.R)
}
