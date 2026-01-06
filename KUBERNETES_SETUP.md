# Configuração do Kubernetes

Este projeto requer um cluster Kubernetes para execução. Existem várias opções para configurar um cluster local.

## Opções de Cluster Local

### 1. Minikube (Recomendado)

```bash
# Instalar minikube (macOS)
brew install minikube

# Iniciar cluster
minikube start

# Verificar status
minikube status

# Configurar Docker para usar minikube
eval $(minikube docker-env)

# Carregar imagens no minikube
minikube image load api-gateway:latest
minikube image load message-processor:latest
minikube image load notification-service:latest
```

### 2. Kind (Kubernetes in Docker)

```bash
# Instalar kind
brew install kind

# Criar cluster
kind create cluster --name queue-case

# Verificar
kubectl cluster-info --context kind-queue-case

# Carregar imagens no kind
kind load docker-image api-gateway:latest --name queue-case
kind load docker-image message-processor:latest --name queue-case
kind load docker-image notification-service:latest --name queue-case
```

### 3. Docker Desktop

1. Abra Docker Desktop
2. Vá em Settings > Kubernetes
3. Marque "Enable Kubernetes"
4. Clique em "Apply & Restart"

As imagens Docker locais estarão disponíveis automaticamente.

## Verificar Configuração

```bash
# Verificar contexto atual
kubectl config current-context

# Verificar se o cluster está acessível
kubectl cluster-info

# Verificar nodes
kubectl get nodes
```

## Deploy sem Validação

Se você quiser aplicar os manifests sem validação (útil quando o cluster não está acessível para validação):

```bash
kubectl apply --validate=false -f k8s/postgresql/deployment.yaml
kubectl apply --validate=false -f k8s/kafka/deployment.yaml
kubectl apply --validate=false -f k8s/api-gateway/deployment.yaml
kubectl apply --validate=false -f k8s/message-processor/deployment.yaml
kubectl apply --validate=false -f k8s/notification-service/deployment.yaml
```

Ou use o Makefile que já inclui `--validate=false`:

```bash
make deploy-all
```

## Troubleshooting

### Erro: "connection refused"

Isso significa que o kubectl não consegue se conectar ao cluster. Verifique:

1. O cluster está rodando?
   ```bash
   minikube status  # para minikube
   kind get clusters  # para kind
   ```

2. O contexto está configurado?
   ```bash
   kubectl config get-contexts
   kubectl config use-context <context-name>
   ```

### Erro: "ImagePullBackOff"

As imagens precisam estar disponíveis no cluster. Para clusters locais:

- **Minikube**: Use `minikube image load <image-name>`
- **Kind**: Use `kind load docker-image <image-name>`
- **Docker Desktop**: Imagens locais estão disponíveis automaticamente

### Erro: "no space left on device"

Limpe recursos não utilizados:

```bash
kubectl delete all --all
docker system prune -a
```

