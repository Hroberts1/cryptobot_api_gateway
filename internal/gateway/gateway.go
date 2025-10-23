package gateway

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"cryptobot-api-gateway/internal/config"
	"cryptobot-api-gateway/internal/messaging"
	"cryptobot-api-gateway/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Gateway represents the API Gateway server
type Gateway struct {
	config        *config.Config
	messageClient *messaging.MessageClient
	wsHub         *websocket.Hub
	logger        *logrus.Entry
}

// NewGateway creates a new gateway instance
func NewGateway(cfg *config.Config, messageClient *messaging.MessageClient, wsHub *websocket.Hub, logger *logrus.Entry) *Gateway {
	return &Gateway{
		config:        cfg,
		messageClient: messageClient,
		wsHub:         wsHub,
		logger:        logger,
	}
}

// SetupRoutes configures all routes for the gateway
func (g *Gateway) SetupRoutes() *gin.Engine {
	// Set Gin mode based on log level
	if g.config.APIGatewayConfig.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Middleware
	router.Use(g.corsMiddleware())
	router.Use(g.loggingMiddleware())
	router.Use(gin.Recovery())

	// Health check endpoint
	router.GET("/health", g.healthCheck)

	// Authentication routes (no auth required)
	g.setupAuthRoutes(router)

	// WebSocket endpoint for real-time updates
	router.GET("/ws", g.handleWebSocket)

	// API routes that proxy to microservices
	api := router.Group("/api")
	{
		api.Use(g.authMiddleware())

		// Route to UI service (if needed for API calls)
		api.Any("/ui/*path", g.proxyToUIService)

		// Route to internal microservices
		for _, service := range g.config.ServiceDependencies.InternalServices {
			g.setupServiceProxy(api, service)
		}
	}

	// Command endpoints for bot control
	commands := router.Group("/commands")
	{
		commands.Use(g.authMiddleware())
		commands.POST("/start-bot", g.handleStartBot)
		commands.POST("/stop-bot", g.handleStopBot)
		commands.POST("/fetch-history", g.handleFetchHistory)
	}

	// External API proxies (like Coinbase)
	external := router.Group("/external")
	{
		external.Use(g.authMiddleware())
		external.Any("/coinbase/*path", g.proxyToCoinbase)
	}

	return router
}

// setupServiceProxy creates a reverse proxy for a microservice
func (g *Gateway) setupServiceProxy(group *gin.RouterGroup, service config.InternalService) {
	targetURL, err := url.Parse(service.TargetURL)
	if err != nil {
		g.logger.Errorf("Invalid target URL for service %s: %v", service.Name, err)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.ErrorHandler = g.proxyErrorHandler

	// Remove the route prefix from the path before forwarding
	group.Any(service.RoutePrefix+"/*path", func(c *gin.Context) {
		// Remove the route prefix from the request path
		originalPath := c.Request.URL.Path
		newPath := strings.TrimPrefix(originalPath, service.RoutePrefix)
		if newPath == "" {
			newPath = "/"
		}
		c.Request.URL.Path = newPath

		g.logger.Debugf("Proxying request from %s to %s%s", originalPath, service.TargetURL, newPath)
		proxy.ServeHTTP(c.Writer, c.Request)
	})
}

// proxyToUIService handles proxying requests to the UI service
func (g *Gateway) proxyToUIService(c *gin.Context) {
	targetURL, err := url.Parse(g.config.ServiceDependencies.UIService.InternalURL)
	if err != nil {
		g.logger.Errorf("Invalid UI service URL: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Service configuration error"})
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.ErrorHandler = g.proxyErrorHandler

	// Remove /api/ui from path
	originalPath := c.Request.URL.Path
	newPath := strings.TrimPrefix(originalPath, "/api/ui")
	if newPath == "" {
		newPath = "/"
	}
	c.Request.URL.Path = newPath

	g.logger.Debugf("Proxying UI request from %s to %s%s", originalPath, g.config.ServiceDependencies.UIService.InternalURL, newPath)
	proxy.ServeHTTP(c.Writer, c.Request)
}

// proxyToCoinbase handles proxying requests to Coinbase API
func (g *Gateway) proxyToCoinbase(c *gin.Context) {
	targetURL, err := url.Parse(g.config.ExternalDependencies.CoinbaseAPI.RestURL)
	if err != nil {
		g.logger.Errorf("Invalid Coinbase API URL: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "External API configuration error"})
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.ErrorHandler = g.proxyErrorHandler

	// Remove /external/coinbase from path
	originalPath := c.Request.URL.Path
	newPath := strings.TrimPrefix(originalPath, "/external/coinbase")
	if newPath == "" {
		newPath = "/"
	}
	c.Request.URL.Path = newPath

	// Add authentication headers for Coinbase
	// Note: In production, you would retrieve these from Kubernetes secrets
	// c.Request.Header.Set("CB-ACCESS-KEY", coinbaseAPIKey)
	// c.Request.Header.Set("CB-ACCESS-SIGN", signature)
	// c.Request.Header.Set("CB-ACCESS-TIMESTAMP", timestamp)

	g.logger.Debugf("Proxying Coinbase request from %s to %s%s", originalPath, g.config.ExternalDependencies.CoinbaseAPI.RestURL, newPath)
	proxy.ServeHTTP(c.Writer, c.Request)
}

// proxyErrorHandler handles proxy errors
func (g *Gateway) proxyErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	g.logger.Errorf("Proxy error: %v", err)
	w.WriteHeader(http.StatusBadGateway)
	w.Write([]byte("Service temporarily unavailable"))
}

// healthCheck provides a health check endpoint
func (g *Gateway) healthCheck(c *gin.Context) {
	status := gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   "1.0.0",
		"services": gin.H{
			"message_broker": g.checkMessageBrokerHealth(),
		},
	}

	c.JSON(http.StatusOK, status)
}

// checkMessageBrokerHealth checks if the message broker is healthy
func (g *Gateway) checkMessageBrokerHealth() string {
	if g.messageClient == nil {
		return "disconnected"
	}
	if g.messageClient.IsConnected() {
		return "connected"
	}
	return "error"
}

// handleWebSocket upgrades HTTP connection to WebSocket
func (g *Gateway) handleWebSocket(c *gin.Context) {
	g.wsHub.HandleWebSocket(c.Writer, c.Request)
}

// Command handlers for bot control
func (g *Gateway) handleStartBot(c *gin.Context) {
	if g.messageClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Message broker not available"})
		return
	}

	var request struct {
		BotID  string                 `json:"botId" binding:"required"`
		Config map[string]interface{} `json:"config"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	message := map[string]interface{}{
		"command":   "start",
		"botId":     request.BotID,
		"config":    request.Config,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	err := g.messageClient.PublishToQueue("queue://commands.start_bot", message)
	if err != nil {
		g.logger.Errorf("Failed to publish start bot command: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send command"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "Start bot command sent", "botId": request.BotID})
}

func (g *Gateway) handleStopBot(c *gin.Context) {
	if g.messageClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Message broker not available"})
		return
	}

	var request struct {
		BotID string `json:"botId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	message := map[string]interface{}{
		"command":   "stop",
		"botId":     request.BotID,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	err := g.messageClient.PublishToQueue("queue://commands.stop_bot", message)
	if err != nil {
		g.logger.Errorf("Failed to publish stop bot command: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send command"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "Stop bot command sent", "botId": request.BotID})
}

func (g *Gateway) handleFetchHistory(c *gin.Context) {
	if g.messageClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Message broker not available"})
		return
	}

	var request struct {
		Symbol    string    `json:"symbol" binding:"required"`
		StartDate time.Time `json:"startDate" binding:"required"`
		EndDate   time.Time `json:"endDate" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	message := map[string]interface{}{
		"command":   "fetch_history",
		"symbol":    request.Symbol,
		"startDate": request.StartDate.Format(time.RFC3339),
		"endDate":   request.EndDate.Format(time.RFC3339),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	err := g.messageClient.PublishToQueue("queue://commands.fetch.history", message)
	if err != nil {
		g.logger.Errorf("Failed to publish fetch history command: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send command"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "Fetch history command sent", "symbol": request.Symbol})
}
