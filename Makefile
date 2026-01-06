.PHONY: build build-all deploy deploy-all clean test logs

# Build all services (must be run from project root)
build-all:
	@if [ ! -d "shared" ] || [ ! -d "message-processor" ]; then \
		echo "Error: Makefile must be run from project root directory"; \
		echo "Current directory: $$(pwd)"; \
		exit 1; \
	fi
	@echo "Building API Gateway..."
	@docker build -f api-gateway/Dockerfile -t api-gateway:latest .
	@echo "Building Message Processor..."
	@docker build -f message-processor/Dockerfile -t message-processor:latest .
	@echo "Building Notification Service..."
	@docker build -f notification-service/Dockerfile -t notification-service:latest .

# Build individual services
build-api:
	@docker build -f api-gateway/Dockerfile -t api-gateway:latest .

build-processor:
	@docker build -f message-processor/Dockerfile -t message-processor:latest .

build-notification:
	@docker build -f notification-service/Dockerfile -t notification-service:latest .

# Check if kubectl is configured
check-k8s:
	@if ! kubectl cluster-info &>/dev/null; then \
		echo "Error: Kubernetes cluster is not accessible"; \
		echo "Please ensure you have a Kubernetes cluster running and kubectl is configured."; \
		echo "For local development, you can use:"; \
		echo "  - minikube: minikube start"; \
		echo "  - kind: kind create cluster"; \
		echo "  - Docker Desktop: Enable Kubernetes in settings"; \
		exit 1; \
	fi

# Deploy infrastructure
deploy-infra: check-k8s
	@echo "Deploying PostgreSQL..."
	@kubectl apply --validate=false -f k8s/postgresql/deployment.yaml
	@echo "Waiting for PostgreSQL..."
	@kubectl wait --for=condition=ready pod -l app=postgresql --timeout=120s || true
	@echo "Deploying Kafka..."
	@kubectl apply --validate=false -f k8s/kafka/deployment.yaml
	@echo "Waiting for Kafka..."
	@kubectl wait --for=condition=ready pod -l app=kafka --timeout=120s || true

# Deploy all services
deploy-all: deploy-infra
	@echo "Deploying API Gateway..."
	@kubectl apply --validate=false -f k8s/api-gateway/deployment.yaml
	@echo "Deploying Message Processor..."
	@kubectl apply --validate=false -f k8s/message-processor/deployment.yaml
	@echo "Deploying Notification Service..."
	@kubectl apply --validate=false -f k8s/notification-service/deployment.yaml

# Clean up
clean:
	@echo "Cleaning up deployments..."
	kubectl delete -f k8s/api-gateway/deployment.yaml || true
	kubectl delete -f k8s/message-processor/deployment.yaml || true
	kubectl delete -f k8s/notification-service/deployment.yaml || true
	kubectl delete -f k8s/postgresql/deployment.yaml || true
	kubectl delete -f k8s/kafka/deployment.yaml || true
	kubectl delete -f k8s/rabbitmq/deployment.yaml || true

# View logs
logs-api:
	kubectl logs -f deployment/api-gateway

logs-processor:
	kubectl logs -f deployment/message-processor

logs-notification:
	kubectl logs -f deployment/notification-service

logs-all:
	kubectl logs -f -l tier=worker,lang=go

# Local development with docker-compose
up:
	docker-compose up -d

down:
	docker-compose down -v

# Test endpoints
test-create:
	@curl -X POST http://localhost:8080/messages \
		-H "Content-Type: application/json" \
		-d '{"content": "Test message", "metadata": {"source": "test"}}'

test-status:
	@echo "Usage: make test-status ID=<message_id>"
	@curl http://localhost:8080/messages/$(ID)/status

