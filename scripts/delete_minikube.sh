#!/usr/bin/env bash
set -e

NAMESPACE="prac8"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "[*] Deleting Kubernetes manifests..."
kubectl delete -n "$NAMESPACE" -f "$ROOT_DIR/k8s" --ignore-not-found=true
kubectl delete -f "$ROOT_DIR/k8s/namespace.yaml" --ignore-not-found=true

echo "[*] Optionally remove images from Minikube docker manually if нужно."
