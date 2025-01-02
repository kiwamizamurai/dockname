package handler

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"

	"github.com/rs/zerolog"
)

func TestProxyHandler_ServeHTTP(t *testing.T) {
	// Prepare test server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate X-Forwarded-* headers
		if r.Header.Get("X-Forwarded-Host") == "" {
			t.Error("X-Forwarded-Host header not set")
		}
		if r.Header.Get("X-Forwarded-Proto") == "" {
			t.Error("X-Forwarded-Proto header not set")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	tests := []struct {
		name           string
		host           string
		setupRoutes    map[string]*httputil.ReverseProxy
		wantStatusCode int
	}{
		{
			name: "Success: Request to registered host",
			host: "example.com",
			setupRoutes: map[string]*httputil.ReverseProxy{
				"example.com": createTestProxy(t, backend.URL),
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "Error: Request to unregistered host",
			host:           "unknown.com",
			setupRoutes:    map[string]*httputil.ReverseProxy{},
			wantStatusCode: http.StatusNotFound,
		},
		{
			name: "Success: Request to host with port number",
			host: "example.com:8000",
			setupRoutes: map[string]*httputil.ReverseProxy{
				"example.com": createTestProxy(t, backend.URL),
			},
			wantStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create proxy handler
			h := NewProxyHandler(zerolog.Nop())

			// Set up routes
			for host, proxy := range tt.setupRoutes {
				h.AddRoute(host, proxy)
			}

			// Create test request
			req := httptest.NewRequest("GET", "http://"+tt.host, nil)
			req.Host = tt.host
			w := httptest.NewRecorder()

			// Execute request
			h.ServeHTTP(w, req)

			// Validate response
			if w.Code != tt.wantStatusCode {
				t.Errorf("ServeHTTP() status code = %v, want %v", w.Code, tt.wantStatusCode)
			}
		})
	}
}

func TestProxyHandler_RouteManagement(t *testing.T) {
	h := NewProxyHandler(zerolog.Nop())

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	proxy := createTestProxy(t, backend.URL)

	h.AddRoute("example.com", proxy)

	routes := h.GetRoutes()
	if len(routes) != 1 || routes[0] != "example.com" {
		t.Errorf("GetRoutes() = %v, want [example.com]", routes)
	}

	h.RemoveRoute("example.com")

	routes = h.GetRoutes()
	if len(routes) != 0 {
		t.Errorf("GetRoutes() after removal = %v, want []", routes)
	}
}

func createTestProxy(t *testing.T, targetURL string) *httputil.ReverseProxy {
	t.Helper()
	target, err := url.Parse(targetURL)
	if err != nil {
		t.Fatalf("Failed to parse target URL: %v", err)
	}
	return httputil.NewSingleHostReverseProxy(target)
}
