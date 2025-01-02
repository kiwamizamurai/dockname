package handler

import (
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"

	"github.com/rs/zerolog"
)

type ProxyHandler struct {
	routes     map[string]*httputil.ReverseProxy
	routesLock sync.RWMutex
	logger     zerolog.Logger
}

func NewProxyHandler(logger zerolog.Logger) *ProxyHandler {
	return &ProxyHandler{
		routes: make(map[string]*httputil.ReverseProxy),
		logger: logger,
	}
}

func (h *ProxyHandler) AddRoute(host string, proxy *httputil.ReverseProxy) {
	h.routesLock.Lock()
	defer h.routesLock.Unlock()
	h.routes[host] = proxy
}

func (h *ProxyHandler) RemoveRoute(host string) {
	h.routesLock.Lock()
	defer h.routesLock.Unlock()
	delete(h.routes, host)
}

func (h *ProxyHandler) GetRoutes() []string {
	h.routesLock.RLock()
	defer h.routesLock.RUnlock()
	routes := make([]string, 0, len(h.routes))
	for route := range h.routes {
		routes = append(routes, route)
	}
	return routes
}

func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := strings.Split(r.Host, ":")[0]

	h.logger.Debug().
		Str("host", host).
		Str("method", r.Method).
		Str("url", r.URL.String()).
		Str("remote_addr", r.RemoteAddr).
		Msg("Request received")

	h.routesLock.RLock()
	proxy, exists := h.routes[host]
	h.routesLock.RUnlock()

	if !exists {
		h.logger.Debug().
			Str("host", host).
			Msg("Route not found")
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	// Set X-Forwarded-* headers
	if clientIP, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		if prior, ok := r.Header["X-Forwarded-For"]; ok {
			clientIP = strings.Join(prior, ", ") + ", " + clientIP
		}
		r.Header.Set("X-Forwarded-For", clientIP)
	}
	r.Header.Set("X-Forwarded-Host", r.Host)
	r.Header.Set("X-Forwarded-Proto", "http")

	proxy.ServeHTTP(w, r)
}
