.PHONY: run-all stop demo-allowed demo-denied demo-post logs build

PIDS_FILE := .pids

build:
	go build -o .build/gateway ./gateway
	go build -o .build/user-service ./services/user-service
	go build -o .build/order-service ./services/order-service
	go build -o .build/inventory-service ./services/inventory-service

run-all:
	@mkdir -p .build logs
	@go build -o .build/gateway ./gateway
	@go build -o .build/user-service ./services/user-service
	@go build -o .build/order-service ./services/order-service
	@go build -o .build/inventory-service ./services/inventory-service
	@.build/gateway          > logs/gateway.log 2>&1 & echo $$! >> $(PIDS_FILE)
	@sleep 0.3
	@.build/user-service     > logs/user-service.log 2>&1 & echo $$! >> $(PIDS_FILE)
	@.build/order-service    > logs/order-service.log 2>&1 & echo $$! >> $(PIDS_FILE)
	@.build/inventory-service > logs/inventory-service.log 2>&1 & echo $$! >> $(PIDS_FILE)
	@sleep 0.5
	@echo ""
	@echo "  All services running. Gateway on :8080"
	@echo "  User-service    :8081"
	@echo "  Order-service   :8082"
	@echo "  Inventory-svc   :8083"
	@echo ""
	@echo "  Run 'make demo-allowed' or 'make demo-denied' to test."
	@echo "  Run 'make stop' to shut down."

stop:
	@if [ -f $(PIDS_FILE) ]; then \
		while read pid; do kill $$pid 2>/dev/null || true; done < $(PIDS_FILE); \
		rm -f $(PIDS_FILE); \
		echo "All services stopped."; \
	else \
		echo "No PID file found."; \
	fi

logs:
	@tail -f logs/gateway.log

# --- Demo targets ---

demo-allowed:
	@echo "=== [1] GET /orders/ord-1 (order-service -> user-service via gateway: ALLOW) ==="
	@curl -s localhost:8082/orders/ord-1 | python3 -m json.tool
	@echo ""
	@echo "=== [2] POST /orders (order-service -> inventory-service via gateway: ALLOW) ==="
	@curl -s -X POST localhost:8082/orders \
		-H "Content-Type: application/json" \
		-d '{"user_id":"1","product_id":"prod-1"}' | python3 -m json.tool
	@echo ""
	@echo "=== [3] GET /gateway/policies (inspect current policy table) ==="
	@curl -s localhost:8080/gateway/policies | python3 -m json.tool

demo-denied:
	@echo "=== [1] user-service -> inventory-service (DENY by policy) ==="
	@curl -s -w "\nHTTP status: %{http_code}\n" \
		-H "X-Service-Name: user-service" \
		-H "X-Target-Service: inventory-service" \
		localhost:8080/inventory/prod-1
	@echo ""
	@echo "=== [2] inventory-service -> user-service (DENY by policy) ==="
	@curl -s -w "\nHTTP status: %{http_code}\n" \
		-H "X-Service-Name: inventory-service" \
		-H "X-Target-Service: user-service" \
		localhost:8080/users/1
	@echo ""
	@echo "=== [3] unknown-service -> user-service (DENY — no matching rule, default=deny) ==="
	@curl -s -w "\nHTTP status: %{http_code}\n" \
		-H "X-Service-Name: unknown-service" \
		-H "X-Target-Service: user-service" \
		localhost:8080/users/1
