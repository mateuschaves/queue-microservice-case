# Troubleshooting Kafka no Kubernetes

## Problema Atual

O Kafka está crashando no Kubernetes sem mostrar o erro completo nos logs. Isso é um problema comum com a imagem Confluent Kafka no Kubernetes.

## Soluções Recomendadas

### Opção 1: Usar Strimzi Operator (Recomendado)

O Strimzi é o operador oficial do Kafka para Kubernetes e é a solução mais robusta:

```bash
# Instalar Strimzi
kubectl create namespace kafka
kubectl create -f 'https://strimzi.io/install/latest?namespace=kafka' -n kafka

# Aplicar Kafka cluster
kubectl apply -f - <<EOF
apiVersion: kafka.strimzi.io/v1beta2
kind: Kafka
metadata:
  name: my-cluster
  namespace: kafka
spec:
  kafka:
    replicas: 1
    listeners:
      - name: plain
        port: 9092
        type: internal
        tls: false
    config:
      offsets.topic.replication.factor: 1
      transaction.state.log.replication.factor: 1
      transaction.state.log.min.isr: 1
      auto.create.topics.enable: "true"
    storage:
      type: ephemeral
  zookeeper:
    replicas: 1
    storage:
      type: ephemeral
EOF

# Aguardar Kafka estar pronto
kubectl wait kafka/my-cluster --for=condition=Ready --timeout=300s -n kafka

# Criar service para expor
kubectl expose service my-cluster-kafka-bootstrap -n kafka --name=kafka --port=9092
```

### Opção 2: Usar Redpanda (Mais Simples)

Redpanda é compatível com Kafka API mas muito mais simples de configurar:

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: redpanda
spec:
  serviceName: redpanda
  replicas: 1
  selector:
    matchLabels:
      app: redpanda
  template:
    metadata:
      labels:
        app: redpanda
    spec:
      containers:
      - name: redpanda
        image: docker.redpanda.com/vectorized/redpanda:latest
        command:
        - /usr/bin/rpk
        - redpanda
        - start
        - --kafka-addr
        - internal://0.0.0.0:9092,external://0.0.0.0:19092
        - --advertise-kafka-addr
        - internal://redpanda:9092,external://localhost:19092
        - --pandaproxy-addr
        - internal://0.0.0.0:8082,external://0.0.0.0:18082
        - --advertise-pandaproxy-addr
        - internal://redpanda:8082,external://localhost:18082
        - --schema-registry-addr
        - internal://0.0.0.0:8081
        - --rpc-addr
        - redpanda:33145
        - --advertise-rpc-addr
        - redpanda:33145
        - --smp
        - "1"
        - --memory
        - "1G"
        - --mode
        - dev-container
        - --default-log-level=info
        ports:
        - containerPort: 9092
        - containerPort: 19092
        - containerPort: 8081
        - containerPort: 8082
        - containerPort: 18082
        - containerPort: 33145
---
apiVersion: v1
kind: Service
metadata:
  name: redpanda
spec:
  type: ClusterIP
  ports:
  - port: 9092
    targetPort: 9092
  selector:
    app: redpanda
```

### Opção 3: Usar Docker Compose para Desenvolvimento

Para desenvolvimento local, use docker-compose que já está configurado:

```bash
docker-compose up -d kafka zookeeper
```

### Opção 4: Debug do Problema Atual

Para debugar o problema atual:

```bash
# Ver logs completos
kubectl logs kafka-0 --previous

# Executar Kafka manualmente para ver erro
kubectl run kafka-debug --image=confluentinc/cp-kafka:7.5.0 --rm -it --restart=Never -- \
  sh -c "export KAFKA_BROKER_ID=1 && export KAFKA_ZOOKEEPER_CONNECT=zookeeper:2181 && \
  export KAFKA_LISTENERS=PLAINTEXT://0.0.0.0:9092 && \
  export KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://kafka:9092 && \
  /etc/confluent/docker/run"

# Verificar se Zookeeper está acessível
kubectl exec -it zookeeper-0 -- nc -zv localhost 2181
```

## Configuração Atual

A configuração atual está em `k8s/kafka/deployment.yaml`. O problema pode ser:

1. **Permissões**: O Kafka pode precisar de permissões especiais
2. **Inicialização**: O script de inicialização pode estar falhando silenciosamente
3. **Memória**: Pode precisar de mais memória
4. **Zookeeper**: Pode haver problema de conectividade com Zookeeper

## Próximos Passos

1. Tente usar o Strimzi Operator (Opção 1) - é a solução mais robusta
2. Ou use Redpanda (Opção 2) - mais simples e compatível
3. Para desenvolvimento, use docker-compose (Opção 3)

