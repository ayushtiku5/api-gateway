package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

type Item struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Stock int    `json:"stock"`
}

var inventory = map[string]*Item{
	"prod-1": {ID: "prod-1", Name: "Widget", Stock: 100},
	"prod-2": {ID: "prod-2", Name: "Gadget", Stock: 50},
	"prod-3": {ID: "prod-3", Name: "Doohickey", Stock: 0},
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/inventory/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/inventory/")
		parts := strings.SplitN(path, "/", 2)
		id := parts[0]

		item, ok := inventory[id]
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"item not found"}`, http.StatusNotFound)
			return
		}

		if r.Method == http.MethodPut && len(parts) == 2 && parts[1] == "reserve" {
			if item.Stock == 0 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]any{"error": "out of stock", "id": id})
				return
			}
			item.Stock--
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"reserved": true, "remaining_stock": item.Stock})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(item)
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"service": "inventory-service", "status": "ok"})
	})

	log.Println("[inventory-service] listening on :8083")
	log.Fatal(http.ListenAndServe(":8083", mux))
}
