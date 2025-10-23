# Kubernetes Deployment Guide

This directory contains Kubernetes manifests for deploying the CryptoBot Go microservice to your Kubernetes cluster.

## Quick Start

1. **Deploy to cluster:**
   ```bash
   # Make deployment script executable
   chmod +x deploy-k8s.sh
   
   # Deploy all components
   ./deploy-k8s.sh
   ```

2. **Verify deployment:**
   ```bash
   kubectl get pods -n cryptobot -l app=cryptobot_ui_service
   kubectl get services -n cryptobot -l app=cryptobot_ui_service
   ```

3. **Access the dashboard:**
   ```bash
   # Port forward to access locally
   kubectl port-forward -n cryptobot service/cryptobot_ui_service-service 8080:80
   
   # Then access http://localhost:8080
   ```

## Manifest Files

### Core Components

- **`namespace-rbac.yaml`**: Creates the `cryptobot` namespace, service account, and network policies
- **`configmap.yaml`**: Application configuration and environment variables
- **`deployment.yaml`**: Main application deployment with 2 replicas, health checks, and security contexts
- **`service.yaml`**: ClusterIP service and headless service for internal communication
- **`hpa.yaml`**: Horizontal Pod Autoscaler for automatic scaling based on CPU/memory usage

### Optional Components

- **`ingress.yaml`**: Ingress configuration for external access (requires ingress controller)

## Configuration

### Environment Variables

The application uses the following environment variables (configured in ConfigMap):

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `ENVIRONMENT` | `production` | Application environment |
| `SERVICE_NAME` | `cryptobot_ui_service` | Service identifier |
| `LOG_LEVEL` | `info` | Logging level |
| `DASHBOARD_TITLE` | `CryptoBot Dashboard` | Dashboard page title |

### Resource Limits

Default resource allocation per pod:
- **Requests**: 64Mi memory, 50m CPU
- **Limits**: 128Mi memory, 100m CPU

### Autoscaling

The HPA is configured to:
- Scale between 2-10 replicas
- Target 70% CPU utilization
- Target 80% memory utilization
- Gradual scale-up/down with stabilization windows

## Security Features

### Pod Security

- Runs as non-root user (UID 1000)
- Read-only root filesystem
- Drops all Linux capabilities
- No privilege escalation allowed

### Network Security

- NetworkPolicy restricts ingress/egress traffic
- Only allows traffic from ingress controller and same namespace
- Service account with minimal permissions

## Manual Deployment Steps

If you prefer to deploy manually:

1. **Create namespace:**
   ```bash
   kubectl apply -f namespace-rbac.yaml
   ```
kubectl rollout status deployment/cryptobot_ui_service -n cryptobot
2. **Deploy configuration:**
   ```bash
   kubectl apply -f configmap.yaml
   ```
kubectl logs -n cryptobot -l app=cryptobot_ui_service -f
3. **Deploy services:**
   ```bash
   kubectl apply -f service.yaml
   ```
kubectl describe pod -n cryptobot -l app=cryptobot_ui_service
4. **Deploy application:**
   ```bash
   kubectl apply -f deployment.yaml
   ```
kubectl exec -n cryptobot deployment/cryptobot_ui_service -- wget -q -O- http://localhost:8080/health
5. **Enable autoscaling:**
   ```bash
   kubectl apply -f hpa.yaml
   ```

6. **Setup ingress (optional):**
   ```bash
   # Edit ingress.yaml to set your domain
kubectl set image deployment/cryptobot_ui_service -n cryptobot cryptobot_ui_service=hroberts1/cryptobot_go:v1.x.x
   ```

## Monitoring and Troubleshooting

### Check deployment status:
```bash
kubectl rollout status deployment/cryptobot_ui_service -n cryptobot
```

### View logs:
```bash
kubectl logs -n cryptobot -l app=cryptobot_ui_service -f
```

### Debug pod issues:
```bash
kubectl describe pod -n cryptobot -l app=cryptobot_ui_service
```

### Test health endpoint:
```bash
kubectl exec -n cryptobot deployment/cryptobot_ui_service -- wget -q -O- http://localhost:8080/health
```

### Check autoscaler:
```bash
kubectl get hpa -n cryptobot
kubectl describe hpa cryptobot_ui_service-hpa -n cryptobot
```

## Customization

### Changing Resource Limits

Edit `deployment.yaml`:
```yaml
resources:
  requests:
    memory: "128Mi"  # Increase as needed
    cpu: "100m"
  limits:
    memory: "256Mi"
    cpu: "200m"
```

### Modifying Replica Count

Edit `deployment.yaml`:
```yaml
spec:
  replicas: 3  # Change from 2 to desired count
```

Or use kubectl:
```bash
kubectl scale deployment cryptobot_ui_service -n cryptobot --replicas=3
```

### Adding Environment Variables

Edit `configmap.yaml` and add your variables:
```yaml
data:
  NEW_VAR: "value"
```

Then reference in `deployment.yaml`:
```yaml
env:
- name: NEW_VAR
  valueFrom:
    configMapKeyRef:
      name: cryptobot_ui_service-config
      key: NEW_VAR
```

## Integration with CI/CD

The GitHub Actions workflow automatically:
1. Builds and pushes Docker images with Kubernetes-specific labels
2. Tags images with semantic versions
3. Uses the `hroberts1/cryptobot_go` repository

After the CI/CD pipeline completes, update the deployment:
```bash
kubectl set image deployment/cryptobot-go -n cryptobot cryptobot-go=hroberts1/cryptobot_go:v1.x.x
```

Or use the latest tag:
```bash
kubectl rollout restart deployment/cryptobot-go -n cryptobot
```

## Cleanup

To remove all components:
```bash
kubectl delete namespace cryptobot
```

Or remove individual components:
```bash
kubectl delete -f k8s/
```