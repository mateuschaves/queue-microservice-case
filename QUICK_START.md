# Quick Start - Acesso Rápido

## Port Forward do API Gateway

```bash
# Iniciar port forward
kubectl port-forward svc/api-gateway 8080:80 &

# Aguardar conexão estabelecer
sleep 3
```

## Testar Criação de Mensagem

```bash
curl -X POST http://localhost:8080/messages \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Test message",
    "metadata": {"source": "test"}
  }'
```

**Resposta esperada:**
```json
{
  "id": "uuid",
  "correlation_id": "uuid",
  "idempotency_id": "uuid",
  "status": "pending"
}
```

## Consultar Status da Mensagem

```bash
# Use o ID retornado na criação
curl http://localhost:8080/messages/{id}/status
```

## Verificar Logs

```bash
# API Gateway
kubectl logs -l app=api-gateway --tail=20

# Message Processor
kubectl logs -l app=message-processor --tail=20

# Notification Service
kubectl logs -l app=notification-service --tail=20
```

## Verificar Fluxo Completo

```bash
# 1. Criar mensagem
RESPONSE=$(curl -s -X POST http://localhost:8080/messages \
  -H "Content-Type: application/json" \
  -d '{"content":"Test","metadata":{"test":"true"}}')

ID=$(echo $RESPONSE | jq -r '.id')
echo "Message ID: $ID"

# 2. Aguardar processamento
sleep 5

# 3. Consultar status
curl -s http://localhost:8080/messages/$ID/status | jq .

# 4. Verificar logs de todos os serviços
echo "=== API Gateway ==="
kubectl logs -l app=api-gateway --tail=3 | grep -i "published\|created"

echo "=== Message Processor ==="
kubectl logs -l app=message-processor --tail=3 | grep -i "processed\|received"

echo "=== Notification Service ==="
kubectl logs -l app=notification-service --tail=3 | grep -i "notification\|received"
```

## Parar Port Forward

```bash
pkill -f "kubectl port-forward"
```

