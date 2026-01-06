# Como Acessar o API Gateway

## Opção 1: Port Forward (Recomendado)

```bash
# Iniciar port forward em background
kubectl port-forward svc/api-gateway 8080:80 &

# Aguardar alguns segundos para o port forward estabelecer conexão
sleep 3

# Testar criação de mensagem
curl -X POST http://localhost:8080/messages \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Test message",
    "metadata": {"source": "test"}
  }'

# Resposta esperada:
# {
#   "id": "...",
#   "correlation_id": "...",
#   "idempotency_id": "...",
#   "status": "pending"
# }

# Para parar o port forward:
pkill -f "kubectl port-forward"
```

## Opção 2: LoadBalancer (Docker Desktop)

Se você estiver usando Docker Desktop, o serviço já está exposto em `localhost`:

```bash
curl -X POST http://localhost/messages \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Test message",
    "metadata": {"source": "test"}
  }'
```

## Opção 3: NodePort

Se o LoadBalancer não funcionar, use o NodePort diretamente:

```bash
# Ver o NodePort
kubectl get svc api-gateway

# Acessar via NodePort (exemplo: 32002)
curl -X POST http://localhost:32002/messages \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Test message",
    "metadata": {"source": "test"}
  }'
```

## Verificar Status

```bash
# Ver pods
kubectl get pods -l app=api-gateway

# Ver logs
kubectl logs -l app=api-gateway --tail=20

# Ver serviço
kubectl get svc api-gateway

# Testar conectividade do pod
kubectl exec -it $(kubectl get pod -l app=api-gateway -o jsonpath='{.items[0].metadata.name}') -- curl localhost:3000
```

## Troubleshooting

### Erro: "Couldn't connect to server"

1. **Verificar se o port forward está ativo:**
   ```bash
   ps aux | grep "kubectl port-forward"
   ```

2. **Verificar se o pod está rodando:**
   ```bash
   kubectl get pods -l app=api-gateway
   ```

3. **Verificar logs do API Gateway:**
   ```bash
   kubectl logs -l app=api-gateway --tail=50
   ```

4. **Reiniciar port forward:**
   ```bash
   # Matar processo existente
   pkill -f "kubectl port-forward"
   
   # Iniciar novo
   kubectl port-forward svc/api-gateway 8080:80
   ```

### Erro: "Connection refused"

- O pod pode estar reiniciando
- Verifique os logs para erros
- Verifique se o serviço está exposto corretamente

### Testar diretamente no pod

```bash
# Obter nome do pod
POD_NAME=$(kubectl get pod -l app=api-gateway -o jsonpath='{.items[0].metadata.name}')

# Testar dentro do pod
kubectl exec $POD_NAME -- curl -X POST http://localhost:3000/messages \
  -H "Content-Type: application/json" \
  -d '{"content":"test"}'
```

