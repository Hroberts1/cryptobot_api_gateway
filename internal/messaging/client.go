package messaging

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/go-stomp/stomp/v3"
	"github.com/sirupsen/logrus"
)

// MessageClient handles connections to ActiveMQ Artemis
type MessageClient struct {
	conn            *stomp.Conn
	subscriptions   map[string]*stomp.Subscription
	subscriptionsMu sync.RWMutex
	logger          *logrus.Entry
	url             string
	connected       bool
	mu              sync.RWMutex
}

// NewMessageClient creates a new message client
func NewMessageClient(brokerURL string, logger *logrus.Entry) (*MessageClient, error) {
	client := &MessageClient{
		subscriptions: make(map[string]*stomp.Subscription),
		logger:        logger,
		url:           brokerURL,
	}

	err := client.connect()
	if err != nil {
		return nil, err
	}

	return client, nil
}

// connect establishes connection to the message broker
func (mc *MessageClient) connect() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Parse the URL to extract host and port
	// For simplicity, assuming URL format: amqp://host:port
	// In production, you might want to use a proper URL parser
	host := "artemis-service"
	port := "61613" // STOMP port for Artemis

	netConn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to message broker: %w", err)
	}

	conn, err := stomp.Connect(netConn, stomp.ConnOpt.Login("admin", "admin"))
	if err != nil {
		netConn.Close()
		return fmt.Errorf("failed to establish STOMP connection: %w", err)
	}

	mc.conn = conn
	mc.connected = true
	mc.logger.Info("Connected to message broker")

	return nil
}

// IsConnected returns the connection status
func (mc *MessageClient) IsConnected() bool {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.connected && mc.conn != nil
}

// SubscribeToTopic subscribes to a topic and calls the handler for each message
func (mc *MessageClient) SubscribeToTopic(topic string, handler func([]byte) error) error {
	if !mc.IsConnected() {
		return fmt.Errorf("not connected to message broker")
	}

	mc.subscriptionsMu.Lock()
	defer mc.subscriptionsMu.Unlock()

	// Check if already subscribed
	if _, exists := mc.subscriptions[topic]; exists {
		return fmt.Errorf("already subscribed to topic: %s", topic)
	}

	sub, err := mc.conn.Subscribe(topic, stomp.AckAuto)
	if err != nil {
		return fmt.Errorf("failed to subscribe to topic %s: %w", topic, err)
	}

	mc.subscriptions[topic] = sub
	mc.logger.Infof("Subscribed to topic: %s", topic)

	// Handle messages in a goroutine
	go func() {
		for {
			msg := <-sub.C
			if msg.Err != nil {
				mc.logger.Errorf("Error receiving message from topic %s: %v", topic, msg.Err)
				break
			}

			if err := handler(msg.Body); err != nil {
				mc.logger.Errorf("Error handling message from topic %s: %v", topic, err)
			}
		}

		// Clean up subscription
		mc.subscriptionsMu.Lock()
		delete(mc.subscriptions, topic)
		mc.subscriptionsMu.Unlock()
		mc.logger.Infof("Unsubscribed from topic: %s", topic)
	}()

	return nil
}

// PublishToQueue publishes a message to a queue
func (mc *MessageClient) PublishToQueue(queue string, message interface{}) error {
	if !mc.IsConnected() {
		return fmt.Errorf("not connected to message broker")
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = mc.conn.Send(queue, "application/json", data)
	if err != nil {
		return fmt.Errorf("failed to send message to queue %s: %w", queue, err)
	}

	mc.logger.Debugf("Published message to queue: %s", queue)
	return nil
}

// PublishToTopic publishes a message to a topic
func (mc *MessageClient) PublishToTopic(topic string, message interface{}) error {
	if !mc.IsConnected() {
		return fmt.Errorf("not connected to message broker")
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = mc.conn.Send(topic, "application/json", data)
	if err != nil {
		return fmt.Errorf("failed to send message to topic %s: %w", topic, err)
	}

	mc.logger.Debugf("Published message to topic: %s", topic)
	return nil
}

// Unsubscribe unsubscribes from a topic
func (mc *MessageClient) Unsubscribe(topic string) error {
	mc.subscriptionsMu.Lock()
	defer mc.subscriptionsMu.Unlock()

	sub, exists := mc.subscriptions[topic]
	if !exists {
		return fmt.Errorf("not subscribed to topic: %s", topic)
	}

	err := sub.Unsubscribe()
	if err != nil {
		return fmt.Errorf("failed to unsubscribe from topic %s: %w", topic, err)
	}

	delete(mc.subscriptions, topic)
	mc.logger.Infof("Unsubscribed from topic: %s", topic)
	return nil
}

// Close closes the connection and all subscriptions
func (mc *MessageClient) Close() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if !mc.connected || mc.conn == nil {
		return
	}

	// Unsubscribe from all topics
	mc.subscriptionsMu.Lock()
	for topic, sub := range mc.subscriptions {
		sub.Unsubscribe()
		delete(mc.subscriptions, topic)
		mc.logger.Infof("Unsubscribed from topic: %s", topic)
	}
	mc.subscriptionsMu.Unlock()

	// Close connection
	mc.conn.Disconnect()
	mc.connected = false
	mc.logger.Info("Disconnected from message broker")
}
