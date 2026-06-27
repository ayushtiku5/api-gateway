package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	gatewayURL  = "http://localhost:8080"
	serviceName = "order-service"
)

type Order struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	ProductID string    `json:"product_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

var orders = map[string]Order{
	"ord-1": {ID: "ord-1", UserID: "1", ProductID: "prod-1", Status: "shipped", CreatedAt: time.Now().Add(-48 * time.Hour)},
	"ord-2": {ID: "ord-2", UserID: "2", ProductID: "prod-2", Status: "pending", CreatedAt: time.Now().Add(-2 * time.Hour)},
}

// callViaGateway makes an inter-service call through the gateway.
func callViaGateway(method, targetService, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, gatewayURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Service-Name", serviceName)
	req.Header.Set("X-Target-Service", targetService)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return http.DefaultClient.Do(req)
}

func main() {
	mux := http.NewServeMux()

	// GET /orders/{id} — fetches order and enriches it with user info via gateway.
	mux.HandleFunc("/orders/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/orders/")
		order, ok := orders[id]
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "order not found"})
			return
		}

		// Call user-service through gateway (allowed by policy).
		resp, err := callViaGateway(http.MethodGet, "user-service", fmt.Sprintf("/users/%s", order.UserID), nil)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			json.NewEncoder(w).Encode(map[string]string{"error": "could not reach user-service"})
			return
		}
		defer resp.Body.Close()

		var user map[string]any
		json.NewDecoder(resp.Body).Decode(&user)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"order": order,
			"user":  user,
		})
	})

	// POST /orders — creates order and reserves inventory via gateway.
	mux.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			UserID    string `json:"user_id"`
			ProductID string `json:"product_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
			return
		}

		// Reserve inventory through gateway (allowed by policy).
		resp, err := callViaGateway(http.MethodPut, "inventory-service",
			fmt.Sprintf("/inventory/%s/reserve", req.ProductID), nil)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			json.NewEncoder(w).Encode(map[string]string{"error": "could not reach inventory-service"})
			return
		}
		defer resp.Body.Close()

		var invResp map[string]any
		json.NewDecoder(resp.Body).Decode(&invResp)

		if resp.StatusCode != http.StatusOK {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(resp.StatusCode)
			json.NewEncoder(w).Encode(map[string]any{
				"error":             "inventory reservation failed",
				"inventory_service": invResp,
			})
			return
		}

		id := fmt.Sprintf("ord-%d", len(orders)+1)
		order := Order{
			ID:        id,
			UserID:    req.UserID,
			ProductID: req.ProductID,
			Status:    "created",
			CreatedAt: time.Now(),
		}
		orders[id] = order

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"order":     order,
			"inventory": invResp,
		})
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"service": "order-service", "status": "ok"})
	})

	log.Println("[order-service] listening on :8082")
	log.Fatal(http.ListenAndServe(":8082", mux))
}
