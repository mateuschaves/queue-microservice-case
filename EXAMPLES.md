# Exemplos de Uso

Este documento contém exemplos práticos de como usar o sistema.

## Exemplo 1: Criar e Consultar Mensagem

### 1. Criar uma mensagem

```bash
curl -X POST http://localhost:8080/messages \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Mensagem de teste",
    "metadata": {
      "source": "curl",
      "priority": "high"
    }
  }'
```

**Resposta:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "correlation_id": "660e8400-e29b-41d4-a716-446655440001",
  "idempotency_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "pending"
}
```

### 2. Consultar status

```bash
curl http://localhost:8080/messages/550e8400-e29b-41d4-a716-446655440000/status
```

**Resposta:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "correlation_id": "660e8400-e29b-41d4-a716-446655440001",
  "status": "processed",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:05Z",
  "history": [
    {
      "status": "pending",
      "service": "api-gateway",
      "event_id": "event-123",
      "error": null,
      "timestamp": "2024-01-15T10:30:00Z"
    },
    {
      "status": "processing",
      "service": "message-processor",
      "event_id": "event-124",
      "error": null,
      "timestamp": "2024-01-15T10:30:02Z"
    },
    {
      "status": "processed",
      "service": "message-processor",
      "event_id": "event-125",
      "error": null,
      "timestamp": "2024-01-15T10:30:05Z"
    }
  ]
}
```

## Exemplo 2: Rastrear por Correlation ID

### Buscar logs com correlation_id

```bash
# Kubernetes
kubectl logs -l app=api-gateway | grep "correlation_id"
kubectl logs -l app=message-processor | grep "correlation_id"
kubectl logs -l app=notification-service | grep "correlation_id"

# Docker Compose
docker-compose logs | grep "correlation_id"
```

### Filtrar por correlation_id específico

```bash
CORRELATION_ID="660e8400-e29b-41d4-a716-446655440001"

kubectl logs -l app=api-gateway | grep "$CORRELATION_ID"
kubectl logs -l app=message-processor | grep "$CORRELATION_ID"
kubectl logs -l app=notification-service | grep "$CORRELATION_ID"
```

## Exemplo 3: Testar Idempotência

### Enviar a mesma mensagem duas vezes

```bash
# Primeira vez
RESPONSE=$(curl -s -X POST http://localhost:8080/messages \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Test idempotency",
    "metadata": {"test": "idempotency"}
  }')

ID=$(echo $RESPONSE | jq -r '.id')
echo "Message ID: $ID"

# Segunda vez (deve retornar o mesmo ID ou ser ignorada)
curl -X POST http://localhost:8080/messages \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Test idempotency",
    "metadata": {"test": "idempotency"}
  }'
```

O sistema deve garantir que a mensagem seja processada apenas uma vez, mesmo que seja enviada múltiplas vezes.

## Exemplo 4: Verificar DLQ

### Kafka

```bash
# Listar tópicos DLQ
kubectl exec -it $(kubectl get pod -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- \
  kafka-topics --list --bootstrap-server localhost:9092 | grep dlq

# Consumir mensagens da DLQ
kubectl exec -it $(kubectl get pod -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- \
  kafka-console-consumer \
    --bootstrap-server localhost:9092 \
    --topic message.created.dlq \
    --from-beginning
```

### RabbitMQ

```bash
# Port forward para interface de gerenciamento
kubectl port-forward svc/rabbitmq 15672:15672

# Acessar http://localhost:15672 (guest/guest)
# Navegar até Queues e procurar por filas terminadas em .dlq
```

## Exemplo 5: Executar Chaos Engineering

### 1. Instalar Chaos Mesh

```bash
curl -sSL https://mirrors.chaos-mesh.org/latest/install.sh | bash
```

### 2. Aplicar experimento de pod kill

```bash
kubectl apply -f chaos/pod-kill.yaml
```

### 3. Monitorar o sistema

```bash
# Ver pods sendo mortos e recriados
watch kubectl get pods -l app=message-processor

# Ver logs para entender o comportamento
kubectl logs -f -l app=message-processor
```

### 4. Verificar que mensagens não são perdidas

```bash
# Criar mensagem durante o chaos
curl -X POST http://localhost:8080/messages \
  -H "Content-Type: application/json" \
  -d '{"content": "Test during chaos"}'

# Aguardar alguns segundos e verificar status
# A mensagem deve ser processada mesmo com pods sendo mortos
```

## Exemplo 6: Alternar entre Kafka e RabbitMQ

### Usar Kafka (padrão)

```bash
# Editar manifests
kubectl edit deployment api-gateway
kubectl edit deployment message-processor
kubectl edit deployment notification-service

# Alterar MESSAGE_BROKER para "kafka"
# Ou usar sed:
kubectl get deployment api-gateway -o yaml | \
  sed 's/value: "rabbit"/value: "kafka"/' | \
  kubectl apply -f -
```

### Usar RabbitMQ

```bash
# Deploy RabbitMQ
kubectl apply -f k8s/rabbitmq/deployment.yaml

# Alterar MESSAGE_BROKER para "rabbit" em todos os serviços
kubectl set env deployment/api-gateway MESSAGE_BROKER=rabbit
kubectl set env deployment/message-processor MESSAGE_BROKER=rabbit
kubectl set env deployment/notification-service MESSAGE_BROKER=rabbit

# Reiniciar pods para aplicar mudanças
kubectl rollout restart deployment/api-gateway
kubectl rollout restart deployment/message-processor
kubectl rollout restart deployment/notification-service
```

## Exemplo 7: Análise de Logs Estruturados

### Filtrar logs por nível

```bash
# Apenas erros
kubectl logs -l app=message-processor | jq 'select(.level == "ERROR")'

# Apenas informações
kubectl logs -l app=message-processor | jq 'select(.level == "INFO")'
```

### Rastrear uma requisição completa

```bash
CORRELATION_ID="660e8400-e29b-41d4-a716-446655440001"

# Todos os logs relacionados
kubectl logs -l app=api-gateway | jq "select(.correlation_id == \"$CORRELATION_ID\")"
kubectl logs -l app=message-processor | jq "select(.correlation_id == \"$CORRELATION_ID\")"
kubectl logs -l app=notification-service | jq "select(.correlation_id == \"$CORRELATION_ID\")"
```

### Estatísticas de processamento

```bash
# Contar mensagens processadas
kubectl logs -l app=message-processor | \
  jq 'select(.message | contains("Message processed successfully"))' | \
  wc -l

# Contar erros
kubectl logs -l app=message-processor | \
  jq 'select(.level == "ERROR")' | \
  wc -l
```

## Exemplo 8: Consultar Histórico no Banco

### Conectar ao PostgreSQL

```bash
# Kubernetes
kubectl exec -it $(kubectl get pod -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- \
  psql -U postgres -d queue_case

# Docker Compose
docker-compose exec postgresql psql -U postgres -d queue_case
```

### Consultas úteis

```sql
-- Ver todas as mensagens
SELECT idempotency_id, correlation_id, status, created_at, updated_at
FROM messages
ORDER BY created_at DESC
LIMIT 10;

-- Ver histórico de uma mensagem específica
SELECT status, service_name, event_id, error_message, created_at
FROM message_history
WHERE idempotency_id = '550e8400-e29b-41d4-a716-446655440000'
ORDER BY created_at ASC;

-- Contar mensagens por status
SELECT status, COUNT(*) as count
FROM messages
GROUP BY status;

-- Mensagens com erro
SELECT m.idempotency_id, m.correlation_id, h.error_message, h.created_at
FROM messages m
JOIN message_history h ON m.idempotency_id = h.idempotency_id
WHERE h.error_message IS NOT NULL
ORDER BY h.created_at DESC;
```

## Exemplo 9: Script de Teste Completo

```bash
#!/bin/bash

API_URL="http://localhost:8080"

echo "=== Criando mensagem ==="
RESPONSE=$(curl -s -X POST $API_URL/messages \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Test message",
    "metadata": {"test": "automated"}
  }')

ID=$(echo $RESPONSE | jq -r '.id')
CORRELATION_ID=$(echo $RESPONSE | jq -r '.correlation_id')

echo "Message ID: $ID"
echo "Correlation ID: $CORRELATION_ID"

echo ""
echo "=== Aguardando processamento (5 segundos) ==="
sleep 5

echo ""
echo "=== Consultando status ==="
curl -s $API_URL/messages/$ID/status | jq '.'

echo ""
echo "=== Verificando logs ==="
echo "Buscando logs com correlation_id: $CORRELATION_ID"
kubectl logs -l app=api-gateway --tail=10 | grep "$CORRELATION_ID" || echo "Nenhum log encontrado"
```

Salve como `test.sh`, torne executável (`chmod +x test.sh`) e execute: `./test.sh`

