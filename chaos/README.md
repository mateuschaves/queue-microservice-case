# Chaos Engineering Experiments

This directory contains Chaos Mesh experiments for testing system resilience.

## Available Experiments

### Pod Kill Experiments

- **pod-kill.yaml**: Kills one pod of message-processor every 2 minutes
- **pod-failure.yaml**: Fails 50% of worker pods for 30 seconds
- **chaos-monkey.yaml**: Modern Chaos Monkey - randomly kills up to 10% of worker pods every minute

### Network Experiments

- **network-latency.yaml**: Adds 100ms latency to message-processor network traffic for 1 minute
- **network-partition.yaml**: Partitions 30% of worker pods for 2 minutes

### Infrastructure Experiments

- **database-failure.yaml**: Kills PostgreSQL pod every 5 minutes
- **broker-failure.yaml**: Kills messaging broker (Kafka/RabbitMQ) every 3 minutes

## Usage

### Apply an experiment:
```bash
kubectl apply -f chaos/<experiment-name>.yaml
```

### Check experiment status:
```bash
kubectl get podchaos
kubectl get networkchaos
```

### Delete an experiment:
```bash
kubectl delete -f chaos/<experiment-name>.yaml
```

## Prerequisites

Chaos Mesh must be installed in your Kubernetes cluster:
```bash
curl -sSL https://mirrors.chaos-mesh.org/latest/install.sh | bash
```

## Safety

All experiments are designed for testing environments. Always test in non-production environments first.

