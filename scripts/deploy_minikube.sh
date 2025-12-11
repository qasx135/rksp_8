#!/usr/bin/env bash
set -e

NAMESPACE="microservices-pr8"
MINIKUBE_PROFILE=${MINIKUBE_PROFILE:-minikube}

echo "[*] Using minikube profile: $MINIKUBE_PROFILE"

# Подключаем docker к демону minikube
echo "[*] Switching docker-env to Minikube..."
eval "$(minikube -p "$MINIKUBE_PROFILE" docker-env)"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "[*] Building Docker images..."
docker build -t pr8-gateway:latest "$ROOT_DIR/services/gateway"
docker build -t pr8-auth-service:latest "$ROOT_DIR/services/auth-service"
docker build -t pr8-user-service:latest "$ROOT_DIR/services/user-service"
docker build -t pr8-order-service:latest "$ROOT_DIR/services/anime-service"

echo "[*] Applying Kubernetes manifests..."
kubectl apply -f "$ROOT_DIR/k8s/namespace.yaml"
kubectl apply -n "$NAMESPACE" -f "$ROOT_DIR/k8s"

echo "[*] Done. To get gateway URL run:"
echo "    minikube service gateway -n $NAMESPACE --url"
