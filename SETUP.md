# Guia de Setup Rápido

Este guia fornece instruções passo a passo para configurar e executar o projeto localmente.

## Pré-requisitos

- Docker e Docker Compose
- Kubernetes (minikube, kind, ou Docker Desktop com Kubernetes)
- kubectl configurado
- Go 1.21+ (para desenvolvimento local)
- Node.js 20+ e npm (para desenvolvimento local)

## Opção 1: Desenvolvimento Local com Docker Compose

### 1. Subir Infraestrutura

```bash
docker-compose up -d
```

Isso sobe:
- PostgreSQL na porta 5432
- Kafka + Zookeeper na porta 9092
- RabbitMQ nas portas 5672 (AMQP) e 15672 (Management UI)

### 2. Executar Serviços Localmente

#### API Gateway (NestJS)

```bash
cd api-gateway
npm install
npm run start:dev
```

Configure as variáveis de ambiente:
```bash
export MESSAGE_BROKER=kafka
export KAFKA_BROKERS=localhost:9092
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=queue_case
```

#### Message Processor (Go)

```bash
cd message-processor
go mod download
go run main.go
```

Configure as variáveis de ambiente:
```bash
export MESSAGE_BROKER=kafka
export KAFKA_BROKERS=localhost:9092
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=queue_case
```

#### Notification Service (Go)

```bash
cd notification-service
go mod download
go run main.go
```

Configure as variáveis de ambiente:
```bash
export MESSAGE_BROKER=kafka
export KAFKA_BROKERS=localhost:9092
```

## Opção 2: Kubernetes Completo

### 1. Build das Imagens

```bash
make build-all
```

Ou manualmente:

```bash
# API Gateway
cd api-gateway
docker build -t api-gateway:latest .

# Message Processor
cd ../message-processor
docker build -t message-processor:latest .

# Notification Service
cd ../notification-service
docker build -t notification-service:latest .
```

### 2. Carregar Imagens no Kubernetes

Se estiver usando minikube:
```bash
minikube image load api-gateway:latest
minikube image load message-processor:latest
minikube image load notification-service:latest
```

Se estiver usando kind:
```bash
kind load docker-image api-gateway:latest
kind load docker-image message-processor:latest
kind load docker-image notification-service:latest
```

### 3. Deploy

```bash
make deploy-all
```

Ou manualmente:
```bash
# Infraestrutura
kubectl apply -f k8s/postgresql/deployment.yaml
kubectl apply -f k8s/kafka/deployment.yaml

# Aguardar
kubectl wait --for=condition=ready pod -l app=postgresql --timeout=120s
kubectl wait --for=condition=ready pod -l app=kafka --timeout=120s

# Serviços
kubectl apply -f k8s/api-gateway/deployment.yaml
kubectl apply -f k8s/message-processor/deployment.yaml
kubectl apply -f k8s/notification-service/deployment.yaml
```

### 4. Acessar API Gateway

```bash
# Port forward
kubectl port-forward svc/api-gateway 8080:80

# Ou obter LoadBalancer IP
kubectl get svc api-gateway
```

## Alternando entre Kafka e RabbitMQ

### Docker Compose

Edite `docker-compose.yml` e use apenas o broker desejado. Configure as variáveis de ambiente nos serviços.

### Kubernetes

Edite os manifests em `k8s/*/deployment.yaml` e altere:

```yaml
env:
- name: MESSAGE_BROKER
  value: "rabbit"  # ou "kafka"
```

Depois aplique:
```bash
kubectl apply -f k8s/
```

## Verificar Funcionamento

### 1. Criar Mensagem

```bash
curl -X POST http://localhost:8080/messages \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Test message",
    "metadata": {"source": "test"}
  }'
```

### 2. Verificar Logs

```bash
# Kubernetes
kubectl logs -f deployment/api-gateway
kubectl logs -f deployment/message-processor
kubectl logs -f deployment/notification-service

# Docker Compose
docker-compose logs -f api-gateway
docker-compose logs -f message-processor
docker-compose logs -f notification-service
```

### 3. Consultar Status

Use o `id` retornado na criação:

```bash
curl http://localhost:8080/messages/{id}/status
```

## Troubleshooting

### Problemas com Go Modules

Se houver problemas com módulos Go:

```bash
cd message-processor
go mod tidy
go mod vendor  # opcional
```

### Problemas com Dependências Node

```bash
cd api-gateway
rm -rf node_modules package-lock.json
npm install
```

### Kafka não está respondendo

```bash
# Verificar se está rodando
docker-compose ps kafka

# Ver logs
docker-compose logs kafka

# Testar conexão
docker-compose exec kafka kafka-broker-api-versions --bootstrap-server localhost:9092
```

### PostgreSQL não está acessível

```bash
# Verificar se está rodando
docker-compose ps postgresql

# Testar conexão
docker-compose exec postgresql psql -U postgres -d queue_case -c "SELECT 1;"
```

## Próximos Passos

- Leia o [README.md](README.md) completo para entender a arquitetura
- Explore os [experimentos de Chaos](chaos/README.md)
- Teste a alternância entre Kafka e RabbitMQ
- Execute experimentos de chaos engineering

