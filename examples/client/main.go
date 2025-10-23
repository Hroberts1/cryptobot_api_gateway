package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	User      UserInfo  `json:"user"`
}

type UserInfo struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
}

func main() {
	// API Gateway base URL
	baseURL := "http://localhost:8080"
	if env := os.Getenv("API_GATEWAY_URL"); env != "" {
		baseURL = env
	}

	fmt.Println("CryptoBot API Gateway Test Client")
	fmt.Printf("Connecting to: %s\n", baseURL)

	// Test authentication
	token, err := login(baseURL, "admin", "admin123")
	if err != nil {
		log.Fatalf("Login failed: %v", err)
	}

	fmt.Printf("Login successful! Token: %.50s...\n", token)

	// Test health check
	if err := testHealthCheck(baseURL); err != nil {
		log.Printf("Health check failed: %v", err)
	}

	// Test WebSocket connection
	go testWebSocket(baseURL, token)

	// Test API endpoints (these will fail until microservices are deployed)
	testAPIEndpoints(baseURL, token)

	// Keep the client running to maintain WebSocket connection
	fmt.Println("\nPress Ctrl+C to exit...")
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt
	fmt.Println("Client stopped.")
}

func login(baseURL, username, password string) (string, error) {
	loginReq := LoginRequest{
		Username: username,
		Password: password,
	}

	data, err := json.Marshal(loginReq)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(baseURL+"/auth/login", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login failed with status: %d", resp.StatusCode)
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return "", err
	}

	return loginResp.Token, nil
}

func testHealthCheck(baseURL string) error {
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}

	var health map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return err
	}

	fmt.Printf("Health check passed: %+v\n", health)
	return nil
}

func testWebSocket(baseURL, token string) {
	u, err := url.Parse(baseURL)
	if err != nil {
		log.Printf("WebSocket URL parse error: %v", err)
		return
	}

	u.Scheme = "ws"
	u.Path = "/ws"
	u.RawQuery = "user_id=test_client"

	header := http.Header{}
	header.Set("Authorization", "Bearer "+token)

	c, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		log.Printf("WebSocket dial error: %v", err)
		return
	}
	defer c.Close()

	fmt.Println("WebSocket connected!")

	// Send ping message
	ping := map[string]string{"type": "ping"}
	if err := c.WriteJSON(ping); err != nil {
		log.Printf("WebSocket write error: %v", err)
		return
	}

	// Read messages
	for {
		var msg map[string]interface{}
		err := c.ReadJSON(&msg)
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			return
		}
		fmt.Printf("WebSocket message received: %+v\n", msg)
	}
}

func testAPIEndpoints(baseURL, token string) {
	client := &http.Client{Timeout: 10 * time.Second}

	endpoints := []string{
		"/api/v1/portfolio",
		"/api/v1/orders/active",
		"/api/v1/reports",
	}

	for _, endpoint := range endpoints {
		req, err := http.NewRequest("GET", baseURL+endpoint, nil)
		if err != nil {
			log.Printf("Error creating request for %s: %v", endpoint, err)
			continue
		}

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Error calling %s: %v", endpoint, err)
			continue
		}
		resp.Body.Close()

		fmt.Printf("API endpoint %s returned status: %d\n", endpoint, resp.StatusCode)
	}
}

func testBotCommands(baseURL, token string) {
	client := &http.Client{Timeout: 10 * time.Second}

	// Test start bot command
	startCmd := map[string]interface{}{
		"botId": "test-bot-1",
		"config": map[string]interface{}{
			"symbol":    "BTC-USD",
			"strategy":  "dca",
			"amount":    100.0,
			"frequency": "hourly",
		},
	}

	data, _ := json.Marshal(startCmd)
	req, err := http.NewRequest("POST", baseURL+"/commands/start-bot", bytes.NewBuffer(data))
	if err != nil {
		log.Printf("Error creating start bot request: %v", err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error calling start bot: %v", err)
		return
	}
	resp.Body.Close()

	fmt.Printf("Start bot command returned status: %d\n", resp.StatusCode)
}
