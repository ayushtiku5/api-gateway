package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

type ProxyHandler struct {
	engine   *PolicyEngine
	services map[string]string
}

func NewProxyHandler(engine *PolicyEngine, services map[string]string) *ProxyHandler {
	return &ProxyHandler{engine: engine, services: services}
}

func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	source := r.Header.Get("X-Service-Name")
	target := r.Header.Get("X-Target-Service")
	start := time.Now()

	if source == "" || target == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "missing X-Service-Name or X-Target-Service header",
		})
		log.Printf("[GATEWAY] 400 missing headers path=%s", r.URL.Path)
		return
	}

	allowed := h.engine.Check(source, target)
	if !allowed {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"error":  "policy denied",
			"source": source,
			"target": target,
		})
		log.Printf("[GATEWAY] DENY  %s -> %s %s %s", source, target, r.Method, r.URL.Path)
		return
	}

	targetBase, ok := h.services[target]
	if !ok {
		writeJSON(w, http.StatusBadGateway, map[string]string{
			"error":  "unknown target service",
			"target": target,
		})
		log.Printf("[GATEWAY] 502  unknown target service=%s", target)
		return
	}

	targetURL, err := url.Parse(targetBase)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "invalid target URL"})
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.ModifyResponse = func(resp *http.Response) error {
		elapsed := time.Since(start)
		log.Printf("[GATEWAY] ALLOW %s -> %s %s %s %d (%s)",
			source, target, r.Method, r.URL.Path, resp.StatusCode, elapsed.Round(time.Millisecond))
		return nil
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("[GATEWAY] ERROR %s -> %s: %v", source, target, err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": fmt.Sprintf("upstream error: %v", err)})
	}

	// Strip internal routing headers before forwarding.
	r.Header.Del("X-Service-Name")
	r.Header.Del("X-Target-Service")
	r.Host = targetURL.Host

	proxy.ServeHTTP(w, r)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
