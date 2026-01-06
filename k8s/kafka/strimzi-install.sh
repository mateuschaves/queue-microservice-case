#!/bin/bash

# Script para instalar Strimzi Operator e criar cluster Kafka

set -e

echo "Instalando Strimzi Operator..."

# Criar namespace
kubectl create namespace kafka --dry-run=client -o yaml | kubectl apply -f -

# Instalar Strimzi
kubectl create -f 'https://strimzi.io/install/latest?namespace=kafka' -n kafka

echo "Aguardando Strimzi estar pronto..."
kubectl wait --for=condition=Ready pod -l name=strimzi-cluster-operator -n kafka --timeout=300s

echo "Criando cluster Kafka..."
kubectl apply -f k8s/kafka/strimzi-kafka.yaml -n kafka

echo "Aguardando Kafka estar pronto (pode levar alguns minutos)..."
kubectl wait kafka/my-cluster --for=condition=Ready --timeout=600s -n kafka

echo "Criando service para expor Kafka no namespace default..."
kubectl apply -f k8s/kafka/strimzi-service.yaml

echo "Kafka instalado e pronto!"

