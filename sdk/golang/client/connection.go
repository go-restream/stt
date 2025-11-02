package asr

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ConnectionManager manages WebSocket connection lifecycle
type ConnectionManager struct {
	conn         *websocket.Conn
	connMutex    sync.RWMutex
	url           string
	headers       http.Header
	dialer       *websocket.Dialer
	connected     bool
	ctx           context.Context
	cancel        context.CancelFunc
	pingInterval  time.Duration
	reconnect     bool
	maxRetries    int
	retryDelay    time.Duration
}

// ConnectionStatus represents the current status of the WebSocket connection
type ConnectionStatus int

const (
	ConnectionStatusDisconnected ConnectionStatus = iota
	ConnectionStatusConnecting
	ConnectionStatusConnected
	ConnectionStatusReconnecting
	ConnectionStatusFailed
)

// NewConnectionManager creates a new connection manager
func NewConnectionManager(url string) *ConnectionManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &ConnectionManager{
		url:          url,
		headers:       make(http.Header),
		dialer:       websocket.DefaultDialer,
		connected:     false,
		ctx:           ctx,
		cancel:        cancel,
		pingInterval:  30 * time.Second,
		reconnect:     true,
		maxRetries:    3,
		retryDelay:    2 * time.Second,
	}
}

// SetHeader sets a custom header for the WebSocket connection
func (cm *ConnectionManager) SetHeader(key, value string) {
	cm.headers.Set(key, value)
}

// SetPingInterval sets the interval for sending ping frames
func (cm *ConnectionManager) SetPingInterval(interval time.Duration) {
	cm.pingInterval = interval
}

// SetReconnectOptions configures automatic reconnection behavior
func (cm *ConnectionManager) SetReconnectOptions(reconnect bool, maxRetries int, retryDelay time.Duration) {
	cm.reconnect = reconnect
	cm.maxRetries = maxRetries
	cm.retryDelay = retryDelay
}

// Connect establishes a WebSocket connection
func (cm *ConnectionManager) Connect() error {
	cm.connMutex.Lock()
	defer cm.connMutex.Unlock()

	if cm.connected {
		return fmt.Errorf("already connected")
	}

	cm.dialer.HandshakeTimeout = 10 * time.Second

	log.Printf("[🔗 Connection] Connecting to WebSocket: %s", cm.url)

	conn, _, err := cm.dialer.Dial(cm.url, cm.headers)
	if err != nil {
		log.Printf("[❌ Connection] Failed to connect: %v", err)
		return fmt.Errorf("connection failed: %w", err)
	}

	cm.conn = conn
	cm.connected = true

	// Set up ping/pong handlers
	cm.conn.SetPingHandler(func(appData string) error {
		log.Printf("[💓 Heartbeat] Received ping from server")
		return cm.conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(5*time.Second))
	})

	cm.conn.SetPongHandler(func(appData string) error {
		log.Printf("[💓 Heartbeat] Received pong from server")
		return nil
	})

	// Set up close handler
	cm.conn.SetCloseHandler(func(code int, text string) error {
		log.Printf("[❌ Connection] Connection closed: %d - %s", code, text)
		cm.connected = false

		if cm.reconnect && code != websocket.CloseNormalClosure {
			go cm.attemptReconnect()
		}
		return nil
	})

	log.Printf("[✅ Connection] Successfully connected to: %s", cm.url)
	return nil
}

// Disconnect closes the WebSocket connection
func (cm *ConnectionManager) Disconnect() error {
	cm.connMutex.Lock()
	defer cm.connMutex.Unlock()

	if !cm.connected {
		return nil
	}

	log.Printf("[🔌 Connection] Disconnecting from WebSocket")

	// Cancel any ongoing operations
	cm.cancel()

	// Close the connection
	if cm.conn != nil {
		err := cm.conn.WriteControl(websocket.CloseMessage, []byte{}, time.Now().Add(5*time.Second))
		if err != nil {
			log.Printf("[⚠️ Connection] Error sending close message: %v", err)
		}

		err = cm.conn.Close()
		if err != nil {
			log.Printf("[⚠️ Connection] Error closing connection: %v", err)
		}

		cm.conn = nil
	}

	cm.connected = false
	log.Printf("[✅ Connection] Successfully disconnected")
	return nil
}

// IsConnected returns the current connection status
func (cm *ConnectionManager) IsConnected() bool {
	cm.connMutex.RLock()
	defer cm.connMutex.RUnlock()
	return cm.connected
}

// GetStatus returns the current detailed connection status
func (cm *ConnectionManager) GetStatus() ConnectionStatus {
	cm.connMutex.RLock()
	defer cm.connMutex.RUnlock()

	if cm.ctx.Err() != nil {
		return ConnectionStatusFailed
	}

	if !cm.connected {
		return ConnectionStatusDisconnected
	}

	return ConnectionStatusConnected
}

// SendMessage sends a text message over the WebSocket
func (cm *ConnectionManager) SendMessage(message []byte) error {
	if !cm.IsConnected() {
		return fmt.Errorf("not connected")
	}

	cm.connMutex.Lock()
	defer cm.connMutex.Unlock()

	if cm.conn == nil {
		return fmt.Errorf("connection is nil")
	}

	cm.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	err := cm.conn.WriteMessage(websocket.TextMessage, message)
	if err != nil {
		log.Printf("[❌ Connection] Failed to send message: %v", err)
		// Mark as disconnected on send error
		cm.connected = false
		return fmt.Errorf("send message failed: %w", err)
	}

	return nil
}

// StartPingLoop starts sending ping frames periodically
func (cm *ConnectionManager) StartPingLoop() {
	ticker := time.NewTicker(cm.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cm.ctx.Done():
			return
		case <-ticker.C:
			if cm.IsConnected() {
				if err := cm.sendPing(); err != nil {
					log.Printf("[⚠️ Heartbeat] Failed to send ping: %v", err)
				}
			}
		}
	}
}

// sendPing sends a ping frame
func (cm *ConnectionManager) sendPing() error {
	cm.connMutex.Lock()
	defer cm.connMutex.Unlock()

	if cm.conn == nil || !cm.connected {
		return fmt.Errorf("connection not available")
	}

	cm.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return cm.conn.WriteMessage(websocket.PingMessage, []byte("heartbeat"))
}

// attemptReconnect tries to reconnect with exponential backoff
func (cm *ConnectionManager) attemptReconnect() {
	log.Printf("[🔄 Connection] Starting reconnection attempt")

	for attempt := 1; attempt <= cm.maxRetries; attempt++ {
		select {
		case <-cm.ctx.Done():
			return
		default:
		}

		delay := time.Duration(attempt) * cm.retryDelay
		if delay > 30*time.Second {
			delay = 30 * time.Second
		}

		log.Printf("[🔄 Connection] Reconnection attempt %d/%d in %v", attempt, cm.maxRetries, delay)
		time.Sleep(delay)

		err := cm.Connect()
		if err == nil {
			log.Printf("[✅ Connection] Successfully reconnected on attempt %d", attempt)
			return
		}

		log.Printf("[❌ Connection] Reconnection attempt %d failed: %v", attempt, err)
	}

	log.Printf("[❌ Connection] All reconnection attempts failed")
}

// ReadMessage reads the next message from the WebSocket
func (cm *ConnectionManager) ReadMessage() (messageType int, message []byte, err error) {
	if !cm.IsConnected() {
		return 0, nil, fmt.Errorf("not connected")
	}

	cm.connMutex.RLock()
	conn := cm.conn
	cm.connMutex.RUnlock()

	if conn == nil {
		return 0, nil, fmt.Errorf("connection is nil")
	}

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	return conn.ReadMessage()
}

// Cleanup performs cleanup of connection resources
func (cm *ConnectionManager) Cleanup() {
	cm.cancel()

	if err := cm.Disconnect(); err != nil {
		log.Printf("[⚠️ Connection] Error during cleanup: %v", err)
	}
}