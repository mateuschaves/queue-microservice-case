#!/bin/bash

# Script para build das imagens Docker
# Deve ser executado a partir da raiz do projeto

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "Building from: $(pwd)"
echo ""

# Verificar se estamos na raiz do projeto
if [ ! -d "shared" ] || [ ! -d "message-processor" ]; then
    echo "Error: This script must be run from the project root directory"
    echo "Current directory: $(pwd)"
    exit 1
fi

echo "Building API Gateway..."
docker build -f api-gateway/Dockerfile -t api-gateway:latest .

echo ""
echo "Building Message Processor..."
docker build -f message-processor/Dockerfile -t message-processor:latest .

echo ""
echo "Building Notification Service..."
docker build -f notification-service/Dockerfile -t notification-service:latest .

echo ""
echo "Build completed successfully!"

