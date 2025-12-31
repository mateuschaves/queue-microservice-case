# Estrutura do Projeto

```
queue-microservice-case/
│
├── api-gateway/                    # API Gateway (NestJS/TypeScript)
│   ├── src/
│   │   ├── main.ts                 # Entry point
│   │   ├── app.module.ts          # Módulo principal
│   │   ├── messages/              # Módulo de mensagens
│   │   │   ├── messages.controller.ts
│   │   │   ├── messages.service.ts
│   │   │   └── dto/
│   │   │       └── create-message.dto.ts
│   │   ├── messaging/             # Serviço de mensageria
│   │   │   ├── messaging.module.ts
│   │   │   └── messaging.service.ts
│   │   └── database/              # Serviço de banco de dados
│   │       ├── database.module.ts
│   │       └── database.service.ts
│   ├── Dockerfile
│   ├── package.json
│   ├── tsconfig.json
│   └── nest-cli.json
│
├── message-processor/              # Message Processor (Go)
│   ├── main.go                    # Entry point
│   ├── go.mod
│   └── Dockerfile
│
├── notification-service/            # Notification Service (Go)
│   ├── main.go                    # Entry point
│   ├── go.mod
│   └── Dockerfile
│
├── shared/                         # Código compartilhado
│   ├── contracts/                  # Contrato de eventos
│   │   ├── event.go               # Estrutura de evento
│   │   ├── errors.go              # Erros do contrato
│   │   ├── utils.go               # Utilitários
│   │   └── go.mod
│   │
│   ├── messaging/                  # Abstração de mensageria
│   │   ├── interface.go           # Interface MessageBroker
│   │   ├── factory.go             # Factory para criar broker
│   │   ├── kafka.go               # Implementação Kafka
│   │   ├── rabbitmq.go            # Implementação RabbitMQ
│   │   └── go.mod
│   │
│   ├── database/                   # Repositório de banco
│   │   ├── repository.go          # Repository com idempotência
│   │   ├── schema.sql              # Schema do PostgreSQL
│   │   └── go.mod
│   │
│   └── logger/                     # Logger estruturado
│       ├── logger.go               # Logger JSON
│       └── go.mod
│
├── k8s/                            # Manifests Kubernetes
│   ├── api-gateway/
│   │   └── deployment.yaml        # Deployment e Service
│   ├── message-processor/
│   │   └── deployment.yaml
│   ├── notification-service/
│   │   └── deployment.yaml
│   ├── postgresql/
│   │   └── deployment.yaml        # PostgreSQL + ConfigMap
│   ├── kafka/
│   │   └── deployment.yaml        # Kafka + Zookeeper
│   └── rabbitmq/
│       └── deployment.yaml
│
├── chaos/                          # Experimentos Chaos Mesh
│   ├── pod-kill.yaml              # Mata pods periodicamente
│   ├── pod-failure.yaml           # Falha de pods
│   ├── network-latency.yaml       # Latência de rede
│   ├── network-partition.yaml     # Partição de rede
│   ├── database-failure.yaml      # Falha do banco
│   ├── broker-failure.yaml        # Falha do broker
│   ├── chaos-monkey.yaml          # Chaos Monkey moderno
│   └── README.md
│
├── docker-compose.yml              # Desenvolvimento local
├── Makefile                        # Comandos úteis
├── .gitignore
├── .prettierrc                     # Configuração Prettier
├── .eslintrc.js                    # Configuração ESLint
│
├── README.md                       # Documentação principal
├── SETUP.md                        # Guia de setup
├── EXAMPLES.md                     # Exemplos de uso
└── PROJECT_STRUCTURE.md            # Este arquivo
```

## Componentes Principais

### API Gateway
- **Tecnologia**: NestJS + TypeScript
- **Porta**: 3000 (padrão)
- **Endpoints**:
  - `POST /messages` - Criar mensagem
  - `GET /messages/:id/status` - Consultar status
- **Responsabilidades**:
  - Gerar correlation_id e idempotency_id
  - Validar entrada
  - Publicar eventos message.created
  - Consultar status no banco

### Message Processor
- **Tecnologia**: Go
- **Responsabilidades**:
  - Consumir message.created
  - Implementar idempotência
  - Processar mensagens
  - Publicar message.status.updated
  - Atualizar banco de dados

### Notification Service
- **Tecnologia**: Go
- **Responsabilidades**:
  - Consumir message.status.updated
  - Registrar logs/notificações

### Shared Modules

#### contracts
Define o contrato único de eventos usado por todos os serviços.

#### messaging
Abstração que permite trocar entre Kafka e RabbitMQ sem alterar código core.

#### database
Repository com suporte a idempotência e histórico de mensagens.

#### logger
Logger estruturado em JSON com correlation_id e idempotency_id.

## Fluxo de Dados

```
Cliente
  │
  ├─> POST /messages
  │
  ▼
API Gateway
  │
  ├─> Gera correlation_id e idempotency_id
  ├─> Salva no PostgreSQL
  └─> Publica message.created
      │
      ▼
Message Processor
  │
  ├─> Verifica idempotência
  ├─> Processa mensagem
  ├─> Atualiza status no PostgreSQL
  └─> Publica message.status.updated
      │
      ▼
Notification Service
  │
  └─> Registra log/notificação
```

## Labels Kubernetes

Todos os recursos possuem labels padronizadas:

- `app`: Nome da aplicação
- `tier`: Camada (api, worker, database, messaging)
- `lang`: Linguagem (typescript, go, sql)

Exemplos de seleção:
```bash
# Todos os workers
kubectl get pods -l tier=worker

# Serviços Go
kubectl get pods -l lang=go

# Aplicar chaos apenas em workers
kubectl apply -f chaos/pod-kill.yaml
```

## Variáveis de Ambiente

### Comuns

- `MESSAGE_BROKER`: Tipo de broker (kafka, rabbit, rabbitmq)
- `KAFKA_BROKERS`: Lista de brokers Kafka
- `RABBITMQ_URL`: URL do RabbitMQ
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`: PostgreSQL

## Tópicos/Filas de Mensageria

### Kafka
- `message.created` - Eventos de criação
- `message.status.updated` - Atualizações de status
- `message.created.dlq` - DLQ para message.created
- `message.status.updated.dlq` - DLQ para message.status.updated

### RabbitMQ
- `message.created` - Fila de criação
- `message.status.updated` - Fila de status
- `message.created.dlq` - DLQ para message.created
- `message.status.updated.dlq` - DLQ para message.status.updated

## Banco de Dados

### Tabelas

#### messages
- `idempotency_id` (PK, UNIQUE)
- `correlation_id`
- `status`
- `payload` (JSONB)
- `created_at`, `updated_at`

#### message_history
- `id` (PK)
- `idempotency_id` (FK)
- `correlation_id`
- `status`
- `service_name`
- `event_id`
- `error_message`
- `created_at`

