package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

func main() {
	cfgPath := "policies.yaml"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	engine := NewPolicyEngine(cfg)
	proxy := NewProxyHandler(engine, cfg.Services)

	mux := http.NewServeMux()

	mux.HandleFunc("/gateway/policies", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"default_action": cfg.DefaultAction,
			"rules":          engine.Rules(),
			"services":       cfg.Services,
		})
	})

	mux.Handle("/", proxy)

	log.Printf("[GATEWAY] starting on :8080 (default_action=%s, %d policies, %d services)",
		cfg.DefaultAction, len(cfg.Policies), len(cfg.Services))
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
