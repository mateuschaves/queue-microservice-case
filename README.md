# Queue Microservice Case Study

Um projeto completo de estudo de arquitetura de microserviços com foco em mensageria, resiliência e chaos engineering, utilizando Kubernetes como ambiente de execução.

## 🎯 Objetivos

Este projeto foi criado para permitir a comparação prática entre diferentes sistemas de mensageria (principalmente Kafka e RabbitMQ), sem que os microsserviços fiquem acoplados diretamente a nenhuma dessas tecnologias. A troca do broker deve ser possível apenas por configuração de variáveis de ambiente, sem alterações no core da aplicação.

## 🏗️ Arquitetura

O sistema é composto por múltiplos microsserviços:

### 1. API Gateway (NestJS/TypeScript)
- **Responsabilidade**: Expor endpoints HTTP
- **Funcionalidades**:
  - Recebe requisições externas
  - Valida dados de entrada
  - Gera `correlation_id` e `idempotency_id`
  - Publica eventos `message.created`
  - Expõe endpoint de consulta de status (`GET /messages/:id/status`)

### 2. Message Processor (Go)
- **Responsabilidade**: Processar mensagens
- **Funcionalidades**:
  - Consome eventos `message.created`
  - Implementa idempotência explícita
  - Processa mensagens e atualiza status no banco
  - Publica eventos `message.status.updated`

### 3. Notification Service (Go)
- **Responsabilidade**: Notificações
- **Funcionalidades**:
  - Consome eventos `message.status.updated`
  - Registra logs/notificações

### 4. PostgreSQL
- **Responsabilidade**: Armazenar estado e histórico
- **Tabelas**:
  - `messages`: Estado atual das mensagens (idempotency_id como chave primária)
  - `message_history`: Histórico completo de mudanças de status

## 📋 Contrato de Eventos

Todos os eventos trocados entre os serviços seguem um contrato único e obrigatório:

```json
{
  "event_id": "string (único)",
  "correlation_id": "string (gerado pela API)",
  "idempotency_id": "string (gerado pela API)",
  "event_type": "string (ex: message.created, message.status.updated)",
  "source_service": "string (nome do serviço)",
  "timestamp": "string (ISO-8601)",
  "payload": {
    // Dados específicos do evento
  }
}
```

**Importante**: Todos esses campos são obrigatórios e devem ser propagados em todos os serviços, logs, mensagens de erro e eventos enviados para DLQ.

## 🔄 Dead Letter Queue (DLQ)

O sistema implementa Dead Letter Queue de forma explícita:

- **Kafka**: Tópicos específicos terminados em `.dlq` (ex: `message.created.dlq`)
- **RabbitMQ**: Dead Letter Exchange (`dlx`) com filas dedicadas (ex: `message.created.dlq`)

Quando uma mensagem falha definitivamente após tentativas de processamento, o evento original é enviado para a DLQ acompanhado do erro ocorrido e do contexto completo (incluindo `correlation_id` e `idempotency_id`).

## 🔐 Idempotência

Todos os consumidores escritos em Go implementam idempotência de forma explícita:

1. Antes de processar qualquer mensagem, o serviço verifica no banco se aquele `idempotency_id` já foi processado
2. Caso positivo, a mensagem é ignorada de forma segura
3. Caso negativo, o processamento ocorre normalmente e o estado é persistido

Isso garante que, mesmo com falhas, retries ou reentregas causadas por Kafka, RabbitMQ ou falhas induzidas por chaos engineering, o sistema não produza efeitos colaterais duplicados.

## 🚀 Como Subir o Cluster Localmente

### Pré-requisitos

- Kubernetes (minikube, kind, ou Docker Desktop com Kubernetes habilitado)
- kubectl configurado
- Docker para build das imagens

### 1. Build das Imagens

```bash
# Build da API Gateway
cd api-gateway
docker build -t api-gateway:latest .

# Build do Message Processor
cd ../message-processor
docker build -t message-processor:latest .

# Build do Notification Service
cd ../notification-service
docker build -t notification-service:latest .
```

### 2. Deploy no Kubernetes

```bash
# Deploy do PostgreSQL
kubectl apply -f k8s/postgresql/deployment.yaml

# Deploy do Kafka (ou RabbitMQ)
kubectl apply -f k8s/kafka/deployment.yaml
# OU
kubectl apply -f k8s/rabbitmq/deployment.yaml

# Aguardar serviços estarem prontos
kubectl wait --for=condition=ready pod -l app=postgresql --timeout=120s
kubectl wait --for=condition=ready pod -l app=kafka --timeout=120s

# Deploy dos microsserviços
kubectl apply -f k8s/api-gateway/deployment.yaml
kubectl apply -f k8s/message-processor/deployment.yaml
kubectl apply -f k8s/notification-service/deployment.yaml
```

### 3. Verificar Status

```bash
# Ver pods
kubectl get pods

# Ver logs do API Gateway
kubectl logs -f deployment/api-gateway

# Ver logs do Message Processor
kubectl logs -f deployment/message-processor

# Ver logs do Notification Service
kubectl logs -f deployment/notification-service
```

## 🔀 Alternando entre Kafka e RabbitMQ

A troca do broker é feita exclusivamente por variável de ambiente `MESSAGE_BROKER`:

### Para usar Kafka (padrão):
```yaml
env:
- name: MESSAGE_BROKER
  value: "kafka"
```

### Para usar RabbitMQ:
```yaml
env:
- name: MESSAGE_BROKER
  value: "rabbit"  # ou "rabbitmq"
```

**Importante**: Todos os serviços devem usar o mesmo broker. Para alternar:

1. Edite os manifests em `k8s/*/deployment.yaml`
2. Altere a variável `MESSAGE_BROKER` em todos os serviços
3. Aplique os manifests novamente: `kubectl apply -f k8s/`

## 🧪 Executando Experimentos de Chaos

### Instalar Chaos Mesh

```bash
curl -sSL https://mirrors.chaos-mesh.org/latest/install.sh | bash
```

### Experimentos Disponíveis

#### 1. Pod Kill (mata pods do message-processor a cada 2 minutos)
```bash
kubectl apply -f chaos/pod-kill.yaml
```

#### 2. Pod Failure (falha 50% dos pods workers por 30 segundos)
```bash
kubectl apply -f chaos/pod-failure.yaml
```

#### 3. Network Latency (adiciona 100ms de latência)
```bash
kubectl apply -f chaos/network-latency.yaml
```

#### 4. Network Partition (particiona 30% dos workers por 2 minutos)
```bash
kubectl apply -f chaos/network-partition.yaml
```

#### 5. Database Failure (mata PostgreSQL a cada 5 minutos)
```bash
kubectl apply -f chaos/database-failure.yaml
```

#### 6. Broker Failure (mata broker a cada 3 minutos)
```bash
kubectl apply -f chaos/broker-failure.yaml
```

#### 7. Chaos Monkey (mata aleatoriamente até 10% dos workers a cada minuto)
```bash
kubectl apply -f chaos/chaos-monkey.yaml
```

### Verificar Status dos Experimentos

```bash
kubectl get podchaos
kubectl get networkchaos
```

### Remover Experimentos

```bash
kubectl delete -f chaos/<experiment-name>.yaml
```

## 📊 Validando DLQ

### Kafka

```bash
# Listar tópicos DLQ
kubectl exec -it <kafka-pod> -- kafka-topics --list --bootstrap-server localhost:9092 | grep dlq

# Consumir mensagens da DLQ
kubectl exec -it <kafka-pod> -- kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic message.created.dlq \
  --from-beginning
```

### RabbitMQ

```bash
# Acessar interface de gerenciamento
kubectl port-forward svc/rabbitmq 15672:15672

# Acessar http://localhost:15672 (guest/guest)
# Verificar filas terminadas em .dlq
```

## 🧪 Testando o Fluxo Completo

### 1. Criar uma Mensagem

```bash
# Obter IP do LoadBalancer
API_URL=$(kubectl get svc api-gateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
# Ou use port-forward se não tiver LoadBalancer
kubectl port-forward svc/api-gateway 8080:80

# Criar mensagem
curl -X POST http://localhost:8080/messages \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Test message",
    "metadata": {
      "source": "test"
    }
  }'

# Resposta:
# {
#   "id": "idempotency_id",
#   "correlation_id": "correlation_id",
#   "idempotency_id": "idempotency_id",
#   "status": "pending"
# }
```

### 2. Consultar Status

```bash
# Usar o id retornado na criação
curl http://localhost:8080/messages/{id}/status

# Resposta:
# {
#   "id": "idempotency_id",
#   "correlation_id": "correlation_id",
#   "status": "processed",
#   "created_at": "...",
#   "updated_at": "...",
#   "history": [
#     {
#       "status": "pending",
#       "service": "api-gateway",
#       "timestamp": "..."
#     },
#     {
#       "status": "processing",
#       "service": "message-processor",
#       "timestamp": "..."
#     },
#     {
#       "status": "processed",
#       "service": "message-processor",
#       "timestamp": "..."
#     }
#   ]
# }
```

### 3. Rastrear por Correlation ID

```bash
# Consultar logs com correlation_id
kubectl logs -l app=api-gateway | grep "correlation_id"
kubectl logs -l app=message-processor | grep "correlation_id"
kubectl logs -l app=notification-service | grep "correlation_id"
```

## 📝 Logs Estruturados

Todos os serviços geram logs estruturados em JSON com os seguintes campos obrigatórios:

- `level`: Nível do log (INFO, ERROR, WARN, DEBUG)
- `service`: Nome do serviço
- `correlation_id`: ID de correlação (quando disponível)
- `idempotency_id`: ID de idempotência (quando disponível)
- `message`: Mensagem do log
- `timestamp`: Timestamp ISO-8601
- `additional_data`: Dados adicionais (opcional)

Exemplo:
```json
{
  "level": "INFO",
  "service": "message-processor",
  "correlation_id": "abc-123",
  "idempotency_id": "def-456",
  "message": "Message processed successfully",
  "timestamp": "2024-01-15T10:30:00Z",
  "additional_data": {
    "status": "processed"
  }
}
```

## 🏷️ Labels Kubernetes

Todos os recursos Kubernetes possuem labels bem definidas para facilitar seleção e aplicação de experimentos de caos:

- `app`: Nome da aplicação
- `tier`: Camada (api, worker, database, messaging)
- `lang`: Linguagem (typescript, go, sql)

Exemplos de seleção:
```bash
# Selecionar todos os workers
kubectl get pods -l tier=worker

# Selecionar serviços Go
kubectl get pods -l lang=go

# Aplicar chaos apenas em workers
kubectl apply -f chaos/pod-kill.yaml  # Configurado para tier=worker
```

## 📁 Estrutura do Projeto

```
queue-microservice-case/
├── api-gateway/              # API Gateway (NestJS/TypeScript)
│   ├── src/
│   ├── Dockerfile
│   └── package.json
├── message-processor/         # Message Processor (Go)
│   ├── main.go
│   ├── Dockerfile
│   └── go.mod
├── notification-service/      # Notification Service (Go)
│   ├── main.go
│   ├── Dockerfile
│   └── go.mod
├── shared/                    # Código compartilhado
│   ├── contracts/            # Contrato de eventos
│   ├── messaging/            # Abstração de mensageria
│   ├── database/             # Repositório de banco
│   └── logger/               # Logger estruturado
├── k8s/                      # Manifests Kubernetes
│   ├── api-gateway/
│   ├── message-processor/
│   ├── notification-service/
│   ├── postgresql/
│   ├── kafka/
│   └── rabbitmq/
├── chaos/                     # Experimentos Chaos Mesh
│   ├── pod-kill.yaml
│   ├── network-latency.yaml
│   ├── chaos-monkey.yaml
│   └── ...
└── README.md
```

## 🔧 Variáveis de Ambiente

### API Gateway
- `PORT`: Porta do servidor (padrão: 3000)
- `MESSAGE_BROKER`: Tipo de broker (kafka, rabbit, rabbitmq)
- `KAFKA_BROKERS`: Lista de brokers Kafka (ex: "localhost:9092")
- `RABBITMQ_URL`: URL do RabbitMQ (ex: "amqp://guest:guest@localhost:5672/")
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`: Configurações do PostgreSQL

### Message Processor / Notification Service
- `MESSAGE_BROKER`: Tipo de broker (kafka, rabbit, rabbitmq)
- `KAFKA_BROKERS`: Lista de brokers Kafka
- `RABBITMQ_URL`: URL do RabbitMQ
- `DB_*`: Configurações do PostgreSQL (apenas message-processor)

## 🎓 Conceitos Demonstrados

Este projeto demonstra na prática:

1. **Arquitetura de Microserviços**: Separação de responsabilidades, comunicação assíncrona
2. **Abstração de Dependências**: Troca de broker sem alterar código core
3. **Idempotência**: Garantia de processamento único mesmo com retries
4. **Rastreabilidade**: Correlation ID e Idempotency ID propagados em toda a cadeia
5. **Dead Letter Queue**: Tratamento de mensagens com falha definitiva
6. **Resiliência**: Sistema preparado para falhas e recuperação
7. **Chaos Engineering**: Testes de resiliência com Chaos Mesh
8. **Observabilidade**: Logs estruturados para rastreamento completo
9. **Kubernetes**: Deploy e orquestração de microserviços
10. **Event-Driven Architecture**: Comunicação baseada em eventos

## ⚠️ Notas Importantes

- Este é um projeto de **estudo e experimentação**, não uma aplicação de produção
- O foco está em demonstrar conceitos arquiteturais, não performance ou escalabilidade extrema
- Alguns recursos podem ser simplificados para facilitar o entendimento (ex: PostgreSQL sem HA)
- Sempre teste em ambientes não-produtivos antes de aplicar em produção

## 📚 Referências

- [NestJS Documentation](https://docs.nestjs.com/)
- [Go Documentation](https://go.dev/doc/)
- [Kafka Documentation](https://kafka.apache.org/documentation/)
- [RabbitMQ Documentation](https://www.rabbitmq.com/documentation.html)
- [Chaos Mesh Documentation](https://chaos-mesh.org/docs/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)

## 📄 Licença

MIT

