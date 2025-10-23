#!/bin/bash

# CryptoBot API Gateway Deployment Script

set -euo pipefail

# Configuration
NAMESPACE=${NAMESPACE:-cryptobot}
DOCKER_IMAGE=${DOCKER_IMAGE:-hroberts1/cryptobot-api-gateway}
VERSION=${VERSION:-latest}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Functions
log() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

check_requirements() {
    log "Checking requirements..."
    
    if ! command -v kubectl &> /dev/null; then
        error "kubectl is required but not installed"
    fi
    
    if ! command -v docker &> /dev/null; then
        error "docker is required but not installed"
    fi
    
    log "Requirements check passed"
}

build_image() {
    log "Building Docker image: ${DOCKER_IMAGE}:${VERSION}"
    docker build -t "${DOCKER_IMAGE}:${VERSION}" .
    
    if [ "${VERSION}" != "latest" ]; then
        docker tag "${DOCKER_IMAGE}:${VERSION}" "${DOCKER_IMAGE}:latest"
    fi
}

push_image() {
    log "Pushing Docker image: ${DOCKER_IMAGE}:${VERSION}"
    docker push "${DOCKER_IMAGE}:${VERSION}"
    
    if [ "${VERSION}" != "latest" ]; then
        docker push "${DOCKER_IMAGE}:latest"
    fi
}

create_namespace() {
    log "Creating namespace: ${NAMESPACE}"
    kubectl apply -f k8s/namespace-rbac.yaml
}

deploy_secrets() {
    log "Deploying secrets..."
    
    # Check if secrets already exist
    if kubectl get secret cryptobot-secrets -n "${NAMESPACE}" &> /dev/null; then
        warn "Secrets already exist, skipping creation"
    else
        kubectl apply -f k8s/secrets.yaml
    fi
}

deploy_configmaps() {
    log "Deploying ConfigMaps..."
    kubectl apply -f k8s/configmap.yaml
}

deploy_application() {
    log "Deploying application..."
    
    # Update image tag in deployment if VERSION is not latest
    if [ "${VERSION}" != "latest" ]; then
        sed -i.bak "s|image: ${DOCKER_IMAGE}:latest|image: ${DOCKER_IMAGE}:${VERSION}|g" k8s/deployment.yaml
    fi
    
    kubectl apply -f k8s/deployment.yaml
    kubectl apply -f k8s/service.yaml
    kubectl apply -f k8s/ingress.yaml
    
    # Apply HPA if it exists
    if [ -f "k8s/hpa.yaml" ]; then
        kubectl apply -f k8s/hpa.yaml
    fi
    
    # Restore original deployment file if modified
    if [ -f "k8s/deployment.yaml.bak" ]; then
        mv k8s/deployment.yaml.bak k8s/deployment.yaml
    fi
}

wait_for_deployment() {
    log "Waiting for deployment to be ready..."
    kubectl wait --for=condition=available --timeout=300s deployment/cryptobot-api-gateway -n "${NAMESPACE}"
}

check_deployment() {
    log "Checking deployment status..."
    
    kubectl get pods -n "${NAMESPACE}" -l app=cryptobot-api-gateway
    kubectl get svc -n "${NAMESPACE}" -l app=cryptobot-api-gateway
    kubectl get ingress -n "${NAMESPACE}"
    
    log "Deployment completed successfully!"
}

show_logs() {
    log "Showing recent logs..."
    kubectl logs -n "${NAMESPACE}" -l app=cryptobot-api-gateway --tail=50
}

# Main deployment function
deploy() {
    log "Starting deployment of CryptoBot API Gateway"
    
    check_requirements
    
    if [ "${1:-}" == "build" ] || [ "${1:-}" == "full" ]; then
        build_image
    fi
    
    if [ "${1:-}" == "push" ] || [ "${1:-}" == "full" ]; then
        push_image
    fi
    
    create_namespace
    deploy_secrets
    deploy_configmaps
    deploy_application
    wait_for_deployment
    check_deployment
    
    log "Deployment completed successfully!"
    
    if [ "${2:-}" == "logs" ]; then
        show_logs
    fi
}

# Clean up function
cleanup() {
    warn "Cleaning up deployment..."
    kubectl delete -f k8s/ --ignore-not-found=true -n "${NAMESPACE}"
    log "Cleanup completed"
}

# Help function
show_help() {
    cat << EOF
CryptoBot API Gateway Deployment Script

Usage: $0 [command] [options]

Commands:
    deploy [build|push|full] [logs]  Deploy the application
        build  - Build Docker image only
        push   - Push Docker image only  
        full   - Build, push, and deploy
        logs   - Show logs after deployment
    
    cleanup                          Remove all deployed resources
    status                           Show deployment status
    logs                            Show application logs
    help                            Show this help message

Environment Variables:
    NAMESPACE     - Kubernetes namespace (default: cryptobot)
    DOCKER_IMAGE  - Docker image name (default: hroberts1/cryptobot-api-gateway)
    VERSION       - Image version (default: latest)

Examples:
    $0 deploy full logs          # Full deployment with logs
    $0 deploy build              # Build image only
    NAMESPACE=test $0 deploy     # Deploy to test namespace
    VERSION=v1.2.3 $0 deploy push # Push specific version
EOF
}

# Command line argument handling
case "${1:-help}" in
    "deploy")
        deploy "${2:-}" "${3:-}"
        ;;
    "cleanup")
        cleanup
        ;;
    "status")
        check_deployment
        ;;
    "logs")
        show_logs
        ;;
    "help"|*)
        show_help
        ;;
esac