package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/Noooste/websocket"
)

type WSMessageType string

const (
	RequestMessage   WSMessageType = "request"
	ResponseMessage  WSMessageType = "response"
	ErrorMessage     WSMessageType = "error"
	PingMessage      WSMessageType = "ping"
	PongMessage      WSMessageType = "pong"
	SessionMessage   WSMessageType = "session"
	CreateSessionMsg WSMessageType = "create_session"
	DeleteSessionMsg WSMessageType = "delete_session"
	ApplyJA3Msg      WSMessageType = "apply_ja3"
	ApplyHTTP2Msg    WSMessageType = "apply_http2"
	ApplyHTTP3Msg    WSMessageType = "apply_http3"
	SetProxyMsg      WSMessageType = "set_proxy"
	ClearProxyMsg    WSMessageType = "clear_proxy"
	AddPinsMsg       WSMessageType = "add_pins"
	ClearPinsMsg     WSMessageType = "clear_pins"
	GetIPMsg         WSMessageType = "get_ip"
	HealthMsg        WSMessageType = "health"
)

type WSMessage struct {
	Type    WSMessageType   `json:"type"`
	ID      string          `json:"id,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type WSConnection struct {
	conn      *websocket.Conn
	sessionID string
	mu        sync.Mutex
	closed    bool
	closeChan chan struct{}
}

func NewWSConnection(conn *websocket.Conn, sessionID string) *WSConnection {
	return &WSConnection{
		conn:      conn,
		sessionID: sessionID,
		closeChan: make(chan struct{}),
	}
}

func (c *WSConnection) WriteJSON(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return websocket.ErrCloseSent
	}

	_ = c.conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
	return c.conn.WriteJSON(v)
}

func (c *WSConnection) ReadJSON(v any) error {
	if c.closed {
		return websocket.ErrCloseSent
	}

	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	return c.conn.ReadJSON(v)
}

func (c *WSConnection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	close(c.closeChan)
	return c.conn.Close()
}

func (c *WSConnection) IsClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

func (c *WSConnection) SessionID() string {
	return c.sessionID
}

func (c *WSConnection) SetSessionID(sessionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessionID = sessionID
}

func (c *WSConnection) CloseChan() <-chan struct{} {
	return c.closeChan
}

type ConnectionManager struct {
	connections  map[string]*WSConnection
	sessionConns map[string]*WSConnection
	mu           sync.RWMutex
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connections:  make(map[string]*WSConnection),
		sessionConns: make(map[string]*WSConnection),
	}
}

func (cm *ConnectionManager) AddConnection(connID string, conn *WSConnection) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.connections[connID] = conn
	if conn.SessionID() != "" {
		cm.sessionConns[conn.SessionID()] = conn
	}
}

func (cm *ConnectionManager) UpdateSessionMapping(conn *WSConnection, oldSessionID, newSessionID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if oldSessionID != "" {
		delete(cm.sessionConns, oldSessionID)
	}
	if newSessionID != "" {
		cm.sessionConns[newSessionID] = conn
	}
}

func (cm *ConnectionManager) RemoveConnection(connID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if conn, exists := cm.connections[connID]; exists {
		delete(cm.sessionConns, conn.SessionID())
		delete(cm.connections, connID)
		_ = conn.Close()
	}
}

func (cm *ConnectionManager) GetConnection(connID string) (*WSConnection, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	conn, exists := cm.connections[connID]
	return conn, exists
}

func (cm *ConnectionManager) GetConnectionBySession(sessionID string) (*WSConnection, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	conn, exists := cm.sessionConns[sessionID]
	return conn, exists
}

func (cm *ConnectionManager) ListConnections() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	connIDs := make([]string, 0, len(cm.connections))
	for id := range cm.connections {
		connIDs = append(connIDs, id)
	}

	return connIDs
}

func (cm *ConnectionManager) CloseAll() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for _, conn := range cm.connections {
		_ = conn.Close()
	}

	cm.connections = make(map[string]*WSConnection)
	cm.sessionConns = make(map[string]*WSConnection)
}
