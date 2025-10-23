.PHONY: build run test clean docker-build docker-run lint fmt vet deps tidy help

# Variables
APP_NAME := cryptobot-api-gateway
DOCKER_IMAGE := hroberts1/cryptobot-api-gateway
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.0.0")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := gofmt -s
GOVET := $(GOCMD) vet

# Main targets
build: ## Build the application
	@echo "Building $(APP_NAME)..."
	CGO_ENABLED=0 GOOS=linux $(GOBUILD) -a -installsuffix cgo \
		-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.commitHash=$(COMMIT_HASH)" \
		-o bin/$(APP_NAME) ./cmd/gateway

build-local: ## Build for local development
	@echo "Building $(APP_NAME) for local development..."
	$(GOBUILD) -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.commitHash=$(COMMIT_HASH)" \
		-o bin/$(APP_NAME) ./cmd/gateway

run: build-local ## Build and run the application locally
	@echo "Running $(APP_NAME)..."
	./bin/$(APP_NAME)

run-dev: ## Run with development settings
	@echo "Running $(APP_NAME) in development mode..."
	LOG_LEVEL=debug CONFIG_PATH=./config/gateway-config.json ./bin/$(APP_NAME)

test: ## Run tests
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

test-coverage: test ## Run tests and show coverage
	@echo "Generating coverage report..."
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f bin/$(APP_NAME)
	rm -f coverage.out coverage.html

docker-build: ## Build Docker image
	@echo "Building Docker image: $(DOCKER_IMAGE):$(VERSION)"
	docker build -t $(DOCKER_IMAGE):$(VERSION) .
	docker tag $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest

docker-run: docker-build ## Build and run Docker container
	@echo "Running Docker container..."
	docker run --rm -p 8080:8080 \
		-e LOG_LEVEL=debug \
		-e JWT_SECRET=development-secret-key \
		$(DOCKER_IMAGE):latest

docker-push: docker-build ## Build and push Docker image
	@echo "Pushing Docker image..."
	docker push $(DOCKER_IMAGE):$(VERSION)
	docker push $(DOCKER_IMAGE):latest

lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run

fmt: ## Format Go code
	@echo "Formatting code..."
	$(GOFMT) -w .

vet: ## Run go vet
	@echo "Running go vet..."
	$(GOVET) ./...

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download

tidy: ## Tidy dependencies
	@echo "Tidying dependencies..."
	$(GOMOD) tidy

deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	$(GOGET) -u ./...
	$(GOMOD) tidy

# Kubernetes targets
k8s-deploy: ## Deploy to Kubernetes
	@echo "Deploying to Kubernetes..."
	kubectl apply -f k8s/namespace-rbac.yaml
	kubectl apply -f k8s/secrets.yaml
	kubectl apply -f k8s/configmap.yaml
	kubectl apply -f k8s/deployment.yaml
	kubectl apply -f k8s/service.yaml
	kubectl apply -f k8s/ingress.yaml
	kubectl apply -f k8s/hpa.yaml

k8s-delete: ## Delete from Kubernetes
	@echo "Deleting from Kubernetes..."
	kubectl delete -f k8s/ --ignore-not-found=true

k8s-status: ## Check Kubernetes deployment status
	@echo "Checking deployment status..."
	kubectl get pods -n cryptobot -l app=$(APP_NAME)
	kubectl get svc -n cryptobot -l app=$(APP_NAME)
	kubectl get ingress -n cryptobot

k8s-logs: ## Show Kubernetes logs
	@echo "Showing logs..."
	kubectl logs -f deployment/$(APP_NAME) -n cryptobot

# Development targets
dev-setup: deps tidy ## Setup development environment
	@echo "Setting up development environment..."
	@echo "Installing development tools..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.54.2; \
	fi

example-client: ## Build and run example client
	@echo "Building example client..."
	$(GOBUILD) -o bin/example-client ./examples/client
	@echo "Running example client..."
	API_GATEWAY_URL=http://localhost:8080 ./bin/example-client

# CI/CD targets
ci-test: deps vet test ## Run CI tests
	@echo "Running CI tests..."

ci-build: clean build ## Run CI build
	@echo "Running CI build..."

# Help
help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'