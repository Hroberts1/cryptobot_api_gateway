package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cryptobot-api-gateway/internal/config"
	"cryptobot-api-gateway/internal/gateway"
	"cryptobot-api-gateway/internal/messaging"
	"cryptobot-api-gateway/internal/websocket"

	"github.com/sirupsen/logrus"
)

func main() {
	// Load configuration
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "./config/gateway-config.json"
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup logging
	logLevel, err := logrus.ParseLevel(cfg.APIGatewayConfig.LogLevel)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logrus.SetLevel(logLevel)
	logrus.SetFormatter(&logrus.JSONFormatter{})

	logger := logrus.WithField("service", "api-gateway")
	logger.Info("Starting API Gateway")

	// Initialize message broker connection
	messageClient, err := messaging.NewMessageClient(cfg.ServiceDependencies.MessageBroker.URL, logger)
	if err != nil {
		logger.Warnf("Failed to connect to message broker: %v", err)
		// Continue without message broker for now
	}

	// Initialize WebSocket hub
	wsHub := websocket.NewHub(logger)
	go wsHub.Run()

	// Initialize gateway with all dependencies
	gatewayServer := gateway.NewGateway(cfg, messageClient, wsHub, logger)

	// Setup HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.APIGatewayConfig.ListenPort),
		Handler:      gatewayServer.SetupRoutes(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Infof("API Gateway listening on port %d", cfg.APIGatewayConfig.ListenPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down API Gateway...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Errorf("Server forced shutdown: %v", err)
	}

	// Close message broker connection
	if messageClient != nil {
		messageClient.Close()
	}

	// Close WebSocket hub
	wsHub.Close()

	logger.Info("API Gateway stopped")
}
