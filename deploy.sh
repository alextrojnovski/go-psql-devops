#!/bin/bash
set -e

echo "Building Docker image..."
docker build -t go-app:latest .

echo "Loading image into minikube..."
minikube image load go-app:latest

echo "Applying Kubernetes manifests..."
kubectl apply -f k8s/postgres-deployment.yaml
kubectl apply -f k8s/app-deployment.yaml

echo "Restarting app deployment to pick up new image..."
kubectl rollout restart deployment go-app

echo "Waiting for rollout to complete..."
kubectl rollout status deployment go-app --timeout=60s

echo "Deployment successful. App URL:"
minikube service go-app --url
