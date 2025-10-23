package gateway

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	User      UserInfo  `json:"user"`
}

// UserInfo represents user information
type UserInfo struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
}

// setupAuthRoutes adds authentication routes to the router
func (g *Gateway) setupAuthRoutes(router *gin.Engine) {
	auth := router.Group("/auth")
	{
		auth.POST("/login", g.handleLogin)
		auth.POST("/refresh", g.authMiddleware(), g.handleRefreshToken)
		auth.POST("/logout", g.authMiddleware(), g.handleLogout)
	}
}

// handleLogin processes login requests and returns JWT token
func (g *Gateway) handleLogin(c *gin.Context) {
	var request LoginRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// In production, validate against a real user database
	// This is a simple example for demonstration
	if !g.validateCredentials(request.Username, request.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate JWT token
	userID := "user123"                 // In production, get from database
	roles := []string{"user", "trader"} // In production, get from database

	token, err := g.generateJWTToken(userID, request.Username, roles)
	if err != nil {
		g.logger.Errorf("Failed to generate JWT token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	expiresAt := time.Now().Add(24 * time.Hour)

	response := LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User: UserInfo{
			ID:       userID,
			Username: request.Username,
			Roles:    roles,
		},
	}

	c.JSON(http.StatusOK, response)
}

// handleRefreshToken refreshes an existing JWT token
func (g *Gateway) handleRefreshToken(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	roles, exists := c.Get("roles")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Generate new token
	token, err := g.generateJWTToken(userID.(string), username.(string), roles.([]string))
	if err != nil {
		g.logger.Errorf("Failed to generate JWT token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	expiresAt := time.Now().Add(24 * time.Hour)

	response := LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User: UserInfo{
			ID:       userID.(string),
			Username: username.(string),
			Roles:    roles.([]string),
		},
	}

	c.JSON(http.StatusOK, response)
}

// handleLogout processes logout requests
func (g *Gateway) handleLogout(c *gin.Context) {
	// In production, you might want to blacklist the token
	// For now, we'll just return success
	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}

// validateCredentials validates user credentials
// In production, this should check against a secure user database
func (g *Gateway) validateCredentials(username, password string) bool {
	// This is a simple example - DO NOT use in production!
	// In production, use proper password hashing (bcrypt) and a database
	validUsers := map[string]string{
		"admin":  "admin123",
		"trader": "trader123",
		"demo":   "demo123",
	}

	expectedPassword, exists := validUsers[username]
	return exists && expectedPassword == password
}
