.PHONY: build build-all deploy deploy-all clean test logs

# Build all services
build-all:
	@echo "Building API Gateway..."
	cd api-gateway && docker build -t api-gateway:latest .
	@echo "Building Message Processor..."
	cd message-processor && docker build -t message-processor:latest .
	@echo "Building Notification Service..."
	cd notification-service && docker build -t notification-service:latest .

# Deploy infrastructure
deploy-infra:
	@echo "Deploying PostgreSQL..."
	kubectl apply -f k8s/postgresql/deployment.yaml
	@echo "Waiting for PostgreSQL..."
	kubectl wait --for=condition=ready pod -l app=postgresql --timeout=120s
	@echo "Deploying Kafka..."
	kubectl apply -f k8s/kafka/deployment.yaml
	@echo "Waiting for Kafka..."
	kubectl wait --for=condition=ready pod -l app=kafka --timeout=120s

# Deploy all services
deploy-all: deploy-infra
	@echo "Deploying API Gateway..."
	kubectl apply -f k8s/api-gateway/deployment.yaml
	@echo "Deploying Message Processor..."
	kubectl apply -f k8s/message-processor/deployment.yaml
	@echo "Deploying Notification Service..."
	kubectl apply -f k8s/notification-service/deployment.yaml

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

