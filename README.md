# CryptoBot API Gateway

This is the API Gateway service for the CryptoBot trading application. It serves as the central entry point for all client requests and routes them to appropriate microservices.

## Architecture

The API Gateway provides:

- **Request Routing**: Routes API requests to appropriate backend microservices
- **Authentication**: JWT-based authentication for all protected endpoints
- **WebSocket Support**: Real-time updates via WebSocket connections
- **Message Broker Integration**: Publishes commands to ActiveMQ Artemis
- **External API Proxy**: Secure proxy for external APIs like Coinbase
- **CORS Handling**: Cross-Origin Resource Sharing configuration

## Configuration

The gateway is configured via:

1. **JSON Configuration File**: `/app/config/gateway-config.json`
2. **Environment Variables**: For sensitive data and runtime configuration
3. **Kubernetes ConfigMaps**: For application settings
4. **Kubernetes Secrets**: For JWT secrets and API keys

### Key Configuration Sections

#### API Gateway Config
- Listen port (default: 8080)
- Log level (info, debug, warn, error)
- CORS origins
- JWT secret key

#### Service Dependencies
- **Message Broker**: ActiveMQ Artemis connection details
- **Internal Services**: Microservice routing configuration
- **UI Service**: Frontend service connection

#### External Dependencies
- **Coinbase API**: External trading API configuration

## API Endpoints

### Health Check
- `GET /health` - Service health status

### Authentication
Protected endpoints require `Authorization: Bearer <jwt-token>` header.

### API Routes (Protected)
- `/api/v1/portfolio/*` → account-service
- `/api/v1/orders/active/*` → order-monitor-service
- `/api/v1/trade/execute/*` → buy-sell-engine
- `/api/v1/reports/*` → report-engine
- `/api/ui/*` → cryptobot-ui-service

### Command Endpoints (Protected)
- `POST /commands/start-bot` - Start trading bot
- `POST /commands/stop-bot` - Stop trading bot
- `POST /commands/fetch-history` - Fetch historical data

### External API Proxy (Protected)
- `/external/coinbase/*` → Coinbase API

### WebSocket
- `GET /ws` - WebSocket connection for real-time updates

## Message Broker Integration

The gateway subscribes to these topics for real-time updates:
- `topic://trades.filled`
- `topic://orders.updated`
- `topic://pnl.update`
- `topic://portfolio.snapshot`
- `topic://system.registry.online`
- `topic://system.registry.offline`
- `topic://market.data.live`

And publishes commands to these queues:
- `queue://commands.start_bot`
- `queue://commands.stop_bot`
- `queue://commands.fetch.history`

## WebSocket Messages

Real-time updates are pushed to connected clients via WebSocket in this format:

```json
{
  "type": "message_type",
  "data": {
    // Message payload
  }
}
```

## Development

### Prerequisites
- Go 1.21+
- Docker
- Kubernetes cluster
- ActiveMQ Artemis message broker

### Local Development

1. **Clone the repository**
```bash
git clone <repository-url>
cd cryptobot_api_gateway
```

2. **Install dependencies**
```bash
go mod download
```

3. **Set environment variables**
```bash
export JWT_SECRET="your-jwt-secret-key"
export MESSAGE_BROKER_URL="stomp://localhost:61613"
export LOG_LEVEL="debug"
```

4. **Run the application**
```bash
go run cmd/gateway/main.go
```

### Building Docker Image

```bash
docker build -t hroberts1/cryptobot-api-gateway:latest .
```

### Running Tests

```bash
go test -v ./...
```

## Deployment

### Kubernetes Deployment

1. **Apply namespace and RBAC**
```bash
kubectl apply -f k8s/namespace-rbac.yaml
```

2. **Create secrets**
```bash
# Edit k8s/secrets.yaml with your actual secrets
kubectl apply -f k8s/secrets.yaml
```

3. **Apply configuration**
```bash
kubectl apply -f k8s/configmap.yaml
```

4. **Deploy the service**
```bash
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
```

5. **Configure ingress**
```bash
kubectl apply -f k8s/ingress.yaml
```

6. **Setup HPA (optional)**
```bash
kubectl apply -f k8s/hpa.yaml
```

### Verification

Check deployment status:
```bash
kubectl get pods -n cryptobot
kubectl get svc -n cryptobot
kubectl get ingress -n cryptobot
```

Check logs:
```bash
kubectl logs -f deployment/cryptobot-api-gateway -n cryptobot
```

Health check:
```bash
curl http://api.cryptobot.local/health
```

## Security Considerations

1. **JWT Secrets**: Use strong, randomly generated secrets
2. **API Keys**: Store external API keys in Kubernetes secrets
3. **Network Policies**: Restrict traffic between services
4. **RBAC**: Use minimal required permissions
5. **TLS**: Enable HTTPS in production environments
6. **Input Validation**: All inputs are validated and sanitized

## Monitoring

The gateway provides:
- Health check endpoint for readiness/liveness probes
- Structured JSON logging
- Request/response metrics
- Connection count monitoring for WebSocket clients

## Troubleshooting

### Common Issues

1. **Message broker connection failed**
   - Check Artemis service is running
   - Verify network connectivity
   - Check credentials and URL

2. **Service routing errors**
   - Verify backend services are running
   - Check service URLs in configuration
   - Review network policies

3. **Authentication failures**
   - Verify JWT secret is correctly configured
   - Check token format and expiration
   - Review token generation logic

4. **WebSocket connection issues**
   - Check ingress annotations for WebSocket support
   - Verify proxy timeout settings
   - Review client connection logic

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

[Add your license information here]