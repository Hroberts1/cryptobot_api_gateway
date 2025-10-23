package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config represents the complete gateway configuration
type Config struct {
	APIGatewayConfig     APIGatewayConfig     `json:"apiGatewayConfig"`
	ServiceDependencies  ServiceDependencies  `json:"serviceDependencies"`
	ExternalDependencies ExternalDependencies `json:"externalDependencies"`
}

// APIGatewayConfig contains basic gateway settings
type APIGatewayConfig struct {
	ListenPort   int      `json:"listenPort"`
	LogLevel     string   `json:"logLevel"`
	CorsOrigins  []string `json:"corsOrigins"`
	JWTSecretKey string   `json:"jwtSecretKey"`
}

// ServiceDependencies contains information about internal services
type ServiceDependencies struct {
	MessageBroker    MessageBroker     `json:"messageBroker"`
	InternalServices []InternalService `json:"internalServices"`
	UIService        UIService         `json:"uiService"`
}

// MessageBroker configuration for ActiveMQ Artemis
type MessageBroker struct {
	URL              string   `json:"url"`
	SubscribedTopics []string `json:"subscribedTopics"`
	PublishQueues    []string `json:"publishQueues"`
}

// InternalService represents a microservice in the cluster
type InternalService struct {
	Name        string `json:"name"`
	RoutePrefix string `json:"routePrefix"`
	TargetURL   string `json:"targetUrl"`
}

// UIService represents the UI service configuration
type UIService struct {
	Name        string `json:"name"`
	InternalURL string `json:"internalUrl"`
}

// ExternalDependencies contains external API configurations
type ExternalDependencies struct {
	CoinbaseAPI CoinbaseAPI `json:"coinbaseApi"`
}

// CoinbaseAPI configuration
type CoinbaseAPI struct {
	RestURL            string `json:"restUrl"`
	APIKeySecretRef    string `json:"apiKeySecretRef"`
	APISecretSecretRef string `json:"apiSecretSecretRef"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	var config Config

	// Load base configuration from JSON file
	if configPath != "" {
		file, err := os.Open(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open config file: %w", err)
		}
		defer file.Close()

		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&config); err != nil {
			return nil, fmt.Errorf("failed to decode config file: %w", err)
		}
	}

	// Override with environment variables
	if port := os.Getenv("GATEWAY_PORT"); port != "" {
		// Convert string to int if needed
		config.APIGatewayConfig.ListenPort = 8080 // Default, could parse from env
	}

	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		config.APIGatewayConfig.LogLevel = logLevel
	}

	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		config.APIGatewayConfig.JWTSecretKey = jwtSecret
	}

	if brokerURL := os.Getenv("MESSAGE_BROKER_URL"); brokerURL != "" {
		config.ServiceDependencies.MessageBroker.URL = brokerURL
	}

	if coinbaseKey := os.Getenv("COINBASE_API_KEY"); coinbaseKey != "" {
		config.ExternalDependencies.CoinbaseAPI.APIKeySecretRef = coinbaseKey
	}

	if coinbaseSecret := os.Getenv("COINBASE_API_SECRET"); coinbaseSecret != "" {
		config.ExternalDependencies.CoinbaseAPI.APISecretSecretRef = coinbaseSecret
	}

	return &config, nil
}

// GetServiceByRoutePrefix finds an internal service by its route prefix
func (c *Config) GetServiceByRoutePrefix(prefix string) *InternalService {
	for _, service := range c.ServiceDependencies.InternalServices {
		if service.RoutePrefix == prefix {
			return &service
		}
	}
	return nil
}
