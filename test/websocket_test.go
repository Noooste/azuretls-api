package test_test

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	fhttp "net/http"

	"github.com/Noooste/azuretls-api/internal/common"
	"github.com/Noooste/azuretls-api/internal/rest"
	internal_websocket "github.com/Noooste/azuretls-api/internal/websocket"
	"github.com/Noooste/azuretls-client"
	"github.com/gorilla/websocket"
)

// WebSocketTestServer creates a test server with WebSocket support
type WebSocketTestServer struct {
	URL            string
	listener       net.Listener
	httpServer     *http.Server
	sessionManager common.SessionManager
}

func NewWebSocketTestServer() *WebSocketTestServer {
	sessionManager := &MockSessionManager{
		sessions: make(map[string]*azuretls.Session),
	}

	server := &TestAPIServer{sessionManager: sessionManager}
	fhttpRoutes := rest.SetupRoutes(server)

	// Convert fhttp.Handler to net/http.Handler using a compatibility wrapper
	handler := &fhttpHandlerAdapter{handler: fhttpRoutes}

	// Create a listener on a random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(fmt.Sprintf("failed to create listener: %v", err))
	}

	httpServer := &http.Server{
		Handler: handler,
	}

	testServer := &WebSocketTestServer{
		URL:            "http://" + listener.Addr().String(),
		listener:       listener,
		httpServer:     httpServer,
		sessionManager: sessionManager,
	}

	// Start the server in a goroutine
	go func() {
		_ = httpServer.Serve(listener)
	}()

	// Give the server a moment to start
	time.Sleep(50 * time.Millisecond)

	return testServer
}

func (s *WebSocketTestServer) Close() {
	if s.httpServer != nil {
		_ = s.httpServer.Close()
	}
	if s.listener != nil {
		_ = s.listener.Close()
	}
}

// fhttpHandlerAdapter adapts an fhttp.Handler to work as a net/http.Handler
// while preserving the underlying ResponseWriter's capabilities (like Hijacker)
type fhttpHandlerAdapter struct {
	handler fhttp.Handler
}

func (a *fhttpHandlerAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Create an fhttp request from the standard request
	fhttpReq := &fhttp.Request{
		Method:           r.Method,
		URL:              r.URL,
		Proto:            r.Proto,
		ProtoMajor:       r.ProtoMajor,
		ProtoMinor:       r.ProtoMinor,
		Header:           fhttp.Header(r.Header),
		Body:             r.Body,
		ContentLength:    r.ContentLength,
		TransferEncoding: r.TransferEncoding,
		Close:            r.Close,
		Host:             r.Host,
		Form:             r.Form,
		PostForm:         r.PostForm,
		RemoteAddr:       r.RemoteAddr,
		RequestURI:       r.RequestURI,
	}

	a.handler.ServeHTTP(w, fhttpReq)
}

// WebSocketTestClient wraps websocket.Conn for testing
type WebSocketTestClient struct {
	conn   *websocket.Conn
	mu     sync.Mutex
	closed bool
}

func NewWebSocketTestClient(serverURL string) (*WebSocketTestClient, error) {
	// Convert HTTP URL to WebSocket URL
	wsURL := strings.Replace(serverURL, "http://", "ws://", 1) + "/ws"

	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}

	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return nil, err
	}

	return &WebSocketTestClient{
		conn: conn,
	}, nil
}

func (c *WebSocketTestClient) SendMessage(msgType internal_websocket.WSMessageType, id string, payload interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return fmt.Errorf("connection closed")
	}

	var payloadBytes json.RawMessage
	var err error

	if payload != nil {
		payloadBytes, err = json.Marshal(payload)
		if err != nil {
			return err
		}
	}

	message := internal_websocket.WSMessage{
		Type:    msgType,
		ID:      id,
		Payload: payloadBytes,
	}

	c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return c.conn.WriteJSON(message)
}

func (c *WebSocketTestClient) ReadMessage() (*internal_websocket.WSMessage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, fmt.Errorf("connection closed")
	}

	var message internal_websocket.WSMessage
	_ = c.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	err := c.conn.ReadJSON(&message)
	return &message, err
}

func (c *WebSocketTestClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	return c.conn.Close()
}

// Test Functions

func TestWebSocketConnection(t *testing.T) {
	server := NewWebSocketTestServer()
	defer server.Close()

	client, err := NewWebSocketTestClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer client.Close()

	// Test successful connection
	if client.conn == nil {
		t.Error("Expected WebSocket connection to be established")
	}
}

func TestWebSocketCreateSession(t *testing.T) {
	server := NewWebSocketTestServer()
	defer server.Close()

	client, err := NewWebSocketTestClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer client.Close()

	// Send create session message
	config := common.SessionConfig{
		Proxy: "http://proxy:8080",
	}

	err = client.SendMessage(internal_websocket.CreateSessionMsg, "test-1", config)
	if err != nil {
		t.Fatalf("Failed to send create session message: %v", err)
	}

	// Read response
	response, err := client.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if response.Type != internal_websocket.ResponseMessage {
		t.Errorf("Expected response message, got %s", response.Type)
	}

	if response.ID != "test-1" {
		t.Errorf("Expected ID 'test-1', got %s", response.ID)
	}

	var result map[string]string
	if err := json.Unmarshal(response.Payload, &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result["status"] != "created" {
		t.Errorf("Expected status 'created', got %s", result["status"])
	}

	if result["session_id"] == "" {
		t.Error("Expected session_id to be present")
	}
}

func TestWebSocketDeleteSession(t *testing.T) {
	server := NewWebSocketTestServer()
	defer server.Close()

	client, err := NewWebSocketTestClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer client.Close()

	// First create a session
	sessionID := createWebSocketSession(t, client)

	// Delete the session
	err = client.SendMessage(internal_websocket.DeleteSessionMsg, "test-delete", nil)
	if err != nil {
		t.Fatalf("Failed to send delete session message: %v", err)
	}

	// Read response
	response, err := client.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if response.Type != internal_websocket.ResponseMessage {
		t.Errorf("Expected response message, got %s", response.Type)
	}

	var result map[string]string
	if err := json.Unmarshal(response.Payload, &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result["status"] != "success" {
		t.Errorf("Expected status 'success', got %s", result["status"])
	}

	_ = sessionID // Use sessionID to avoid unused variable warning
}

func TestWebSocketRequestMessage(t *testing.T) {
	server := NewWebSocketTestServer()
	defer server.Close()

	client, err := NewWebSocketTestClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer client.Close()

	// Create session first
	sessionID := createWebSocketSession(t, client)

	// Send request message
	serverReq := common.ServerRequest{
		URL:    "https://httpbin.org/get",
		Method: "GET",
	}

	err = client.SendMessage(internal_websocket.RequestMessage, "test-request", serverReq)
	if err != nil {
		t.Fatalf("Failed to send request message: %v", err)
	}

	// Read response
	response, err := client.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if response.Type != internal_websocket.ResponseMessage {
		t.Errorf("Expected response message, got %s", response.Type)
	}

	if response.ID != "test-request" {
		t.Errorf("Expected ID 'test-request', got %s", response.ID)
	}

	_ = sessionID // Use sessionID to avoid unused variable warning
}

func TestWebSocketRequestWithoutSession(t *testing.T) {
	server := NewWebSocketTestServer()
	defer server.Close()

	client, err := NewWebSocketTestClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer client.Close()

	// Try to send request without creating session
	serverReq := common.ServerRequest{
		URL:    "https://httpbin.org/get",
		Method: "GET",
	}

	err = client.SendMessage(internal_websocket.RequestMessage, "test-no-session", serverReq)
	if err != nil {
		t.Fatalf("Failed to send request message: %v", err)
	}

	// Should get error response
	response, err := client.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if response.Type != internal_websocket.ErrorMessage {
		t.Errorf("Expected error message, got %s", response.Type)
	}
}

func TestWebSocketApplyJA3(t *testing.T) {
	server := NewWebSocketTestServer()
	defer server.Close()

	client, err := NewWebSocketTestClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer client.Close()

	sessionID := createWebSocketSession(t, client)

	// Apply JA3
	payload := map[string]string{
		"ja3":       "test-ja3-string",
		"navigator": "test-navigator",
	}

	err = client.SendMessage(internal_websocket.ApplyJA3Msg, "test-ja3", payload)
	if err != nil {
		t.Fatalf("Failed to send JA3 message: %v", err)
	}

	// Read response
	response, err := client.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if response.Type != internal_websocket.ResponseMessage {
		t.Errorf("Expected response message, got %s", response.Type)
	}

	var result map[string]string
	if err := json.Unmarshal(response.Payload, &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result["status"] != "success" {
		t.Errorf("Expected status 'success', got %s", result["status"])
	}

	_ = sessionID
}

func TestWebSocketApplyHTTP2(t *testing.T) {
	server := NewWebSocketTestServer()
	defer server.Close()

	client, err := NewWebSocketTestClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer client.Close()

	sessionID := createWebSocketSession(t, client)

	payload := map[string]string{
		"fingerprint": "test-http2-fingerprint",
	}

	err = client.SendMessage(internal_websocket.ApplyHTTP2Msg, "test-http2", payload)
	if err != nil {
		t.Fatalf("Failed to send HTTP2 message: %v", err)
	}

	response, err := client.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if response.Type != internal_websocket.ResponseMessage {
		t.Errorf("Expected response message, got %s", response.Type)
	}

	var result map[string]string
	json.Unmarshal(response.Payload, &result)
	if result["status"] != "success" {
		t.Errorf("Expected status 'success', got %s", result["status"])
	}

	_ = sessionID
}

func TestWebSocketApplyHTTP3(t *testing.T) {
	server := NewWebSocketTestServer()
	defer server.Close()

	client, err := NewWebSocketTestClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer client.Close()

	sessionID := createWebSocketSession(t, client)

	payload := map[string]string{
		"fingerprint": "test-http3-fingerprint",
	}

	err = client.SendMessage(internal_websocket.ApplyHTTP3Msg, "test-http3", payload)
	if err != nil {
		t.Fatalf("Failed to send HTTP3 message: %v", err)
	}

	response, err := client.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if response.Type != internal_websocket.ResponseMessage {
		t.Errorf("Expected response message, got %s", response.Type)
	}

	var result map[string]string
	json.Unmarshal(response.Payload, &result)
	if result["status"] != "success" {
		t.Errorf("Expected status 'success', got %s", result["status"])
	}

	_ = sessionID
}

func TestWebSocketSetProxy(t *testing.T) {
	server := NewWebSocketTestServer()
	defer server.Close()

	client, err := NewWebSocketTestClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer client.Close()

	sessionID := createWebSocketSession(t, client)

	payload := map[string]string{
		"proxy": "http://proxy:8080",
	}

	err = client.SendMessage(internal_websocket.SetProxyMsg, "test-proxy", payload)
	if err != nil {
		t.Fatalf("Failed to send set proxy message: %v", err)
	}

	response, err := client.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if response.Type != internal_websocket.ResponseMessage {
		t.Errorf("Expected response message, got %s", response.Type)
	}

	var result map[string]string
	json.Unmarshal(response.Payload, &result)
	if result["status"] != "success" {
		t.Errorf("Expected status 'success', got %s", result["status"])
	}

	_ = sessionID
}

func TestWebSocketClearProxy(t *testing.T) {
	server := NewWebSocketTestServer()
	defer server.Close()

	client, err := NewWebSocketTestClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer client.Close()

	sessionID := createWebSocketSession(t, client)

	err = client.SendMessage(internal_websocket.ClearProxyMsg, "test-clear-proxy", nil)
	if err != nil {
		t.Fatalf("Failed to send clear proxy message: %v", err)
	}

	response, err := client.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if response.Type != internal_websocket.ResponseMessage {
		t.Errorf("Expected response message, got %s", response.Type)
	}

	var result map[string]string
	json.Unmarshal(response.Payload, &result)
	if result["status"] != "success" {
		t.Errorf("Expected status 'success', got %s", result["status"])
	}

	_ = sessionID
}

func TestWebSocketAddPins(t *testing.T) {
	server := NewWebSocketTestServer()
	defer server.Close()

	client, err := NewWebSocketTestClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer client.Close()

	sessionID := createWebSocketSession(t, client)

	payload := map[string]interface{}{
		"url":  "https://example.com",
		"pins": []string{"pin1", "pin2"},
	}

	err = client.SendMessage(internal_websocket.AddPinsMsg, "test-add-pins", payload)
	if err != nil {
		t.Fatalf("Failed to send add pins message: %v", err)
	}

	response, err := client.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if response.Type != internal_websocket.ResponseMessage {
		t.Errorf("Expected response message, got %s", response.Type)
	}

	var result map[string]string
	json.Unmarshal(response.Payload, &result)
	if result["status"] != "success" {
		t.Errorf("Expected status 'success', got %s", result["status"])
	}

	_ = sessionID
}

func TestWebSocketClearPins(t *testing.T) {
	server := NewWebSocketTestServer()
	defer server.Close()

	client, err := NewWebSocketTestClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer client.Close()

	sessionID := createWebSocketSession(t, client)

	payload := map[string]string{
		"url": "https://example.com",
	}

	err = client.SendMessage(internal_websocket.ClearPinsMsg, "test-clear-pins", payload)
	if err != nil {
		t.Fatalf("Failed to send clear pins message: %v", err)
	}

	response, err := client.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if response.Type != internal_websocket.ResponseMessage {
		t.Errorf("Expected response message, got %s", response.Type)
	}

	var result map[string]string
	json.Unmarshal(response.Payload, &result)
	if result["status"] != "success" {
		t.Errorf("Expected status 'success', got %s", result["status"])
	}

	_ = sessionID
}

func TestWebSocketGetIP(t *testing.T) {
	server := NewWebSocketTestServer()
	defer server.Close()

	client, err := NewWebSocketTestClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer client.Close()

	sessionID := createWebSocketSession(t, client)

	err = client.SendMessage(internal_websocket.GetIPMsg, "test-get-ip", nil)
	if err != nil {
		t.Fatalf("Failed to send get IP message: %v", err)
	}

	response, err := client.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if response.Type != internal_websocket.ResponseMessage {
		t.Errorf("Expected response message, got %s", response.Type)
	}

	var result map[string]string
	json.Unmarshal(response.Payload, &result)
	if result["ip"] != "192.168.1.1" {
		t.Errorf("Expected IP '192.168.1.1', got %s", result["ip"])
	}

	_ = sessionID
}

func TestWebSocketHealth(t *testing.T) {
	server := NewWebSocketTestServer()
	defer server.Close()

	client, err := NewWebSocketTestClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer client.Close()

	err = client.SendMessage(internal_websocket.HealthMsg, "test-health", nil)
	if err != nil {
		t.Fatalf("Failed to send health message: %v", err)
	}

	response, err := client.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if response.Type != internal_websocket.ResponseMessage {
		t.Errorf("Expected response message, got %s", response.Type)
	}

	var result map[string]interface{}
	json.Unmarshal(response.Payload, &result)
	if result["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", result["status"])
	}
}

func TestWebSocketPingPong(t *testing.T) {
	server := NewWebSocketTestServer()
	defer server.Close()

	client, err := NewWebSocketTestClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer client.Close()

	// Send ping
	err = client.SendMessage(internal_websocket.PingMessage, "test-ping", nil)
	if err != nil {
		t.Fatalf("Failed to send ping message: %v", err)
	}

	// Read pong response
	response, err := client.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if response.Type != internal_websocket.PongMessage {
		t.Errorf("Expected pong message, got %s", response.Type)
	}

	if response.ID != "test-ping" {
		t.Errorf("Expected ID 'test-ping', got %s", response.ID)
	}
}

func TestWebSocketInvalidMessageType(t *testing.T) {
	server := NewWebSocketTestServer()
	defer server.Close()

	client, err := NewWebSocketTestClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer client.Close()

	// Send invalid message type
	err = client.SendMessage("invalid_type", "test-invalid", nil)
	if err != nil {
		t.Fatalf("Failed to send invalid message: %v", err)
	}

	// Should get error response
	response, err := client.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if response.Type != internal_websocket.ErrorMessage {
		t.Errorf("Expected error message, got %s", response.Type)
	}
}

func TestWebSocketInvalidJSON(t *testing.T) {
	server := NewWebSocketTestServer()
	defer server.Close()

	client, err := NewWebSocketTestClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer client.Close()

	sessionID := createWebSocketSession(t, client)

	// Send invalid JSON for JA3 - valid JSON but wrong structure
	// The handler expects {ja3: string, navigator: string} but we send an array
	invalidPayload := json.RawMessage(`["not", "an", "object"]`)
	message := internal_websocket.WSMessage{
		Type:    internal_websocket.ApplyJA3Msg,
		ID:      "test-invalid-json",
		Payload: invalidPayload,
	}

	client.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	err = client.conn.WriteJSON(message)
	if err != nil {
		t.Fatalf("Failed to send invalid JSON message: %v", err)
	}

	// Should get error response
	response, err := client.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if response.Type != internal_websocket.ErrorMessage {
		t.Errorf("Expected error message, got %s", response.Type)
	}

	_ = sessionID
}

func TestWebSocketConcurrentConnections(t *testing.T) {
	server := NewWebSocketTestServer()
	defer server.Close()

	numClients := 5
	var wg sync.WaitGroup
	errors := make(chan error, numClients)

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			client, err := NewWebSocketTestClient(server.URL)
			if err != nil {
				errors <- fmt.Errorf("client %d failed to connect: %v", clientID, err)
				return
			}
			defer client.Close()

			// Create session
			sessionID := createWebSocketSession(t, client)

			// Send health check
			err = client.SendMessage(internal_websocket.HealthMsg, fmt.Sprintf("health-%d", clientID), nil)
			if err != nil {
				errors <- fmt.Errorf("client %d failed to send health: %v", clientID, err)
				return
			}

			// Read response
			_, err = client.ReadMessage()
			if err != nil {
				errors <- fmt.Errorf("client %d failed to read response: %v", clientID, err)
				return
			}

			_ = sessionID
		}(i)
	}

	// Wait for all clients to complete
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// Check for errors or timeout
	select {
	case err := <-errors:
		t.Fatalf("Concurrent connection test failed: %v", err)
	case <-done:
		// All clients completed successfully
	case <-time.After(30 * time.Second):
		t.Fatal("Concurrent connection test timed out")
	}
}

func TestWebSocketSessionCleanupOnDisconnect(t *testing.T) {
	server := NewWebSocketTestServer()
	defer server.Close()

	client, err := NewWebSocketTestClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}

	// Create session
	sessionID := createWebSocketSession(t, client)

	// Verify session exists
	mockManager := server.sessionManager.(*MockSessionManager)
	initialCount := mockManager.GetSessionCount()
	if initialCount == 0 {
		t.Error("Expected at least one session to exist")
	}

	// Close connection
	client.Close()

	// Give some time for cleanup
	time.Sleep(100 * time.Millisecond)

	// Verify session was cleaned up
	finalCount := mockManager.GetSessionCount()
	if finalCount >= initialCount {
		t.Errorf("Expected session count to decrease after disconnect, initial: %d, final: %d", initialCount, finalCount)
	}

	_ = sessionID
}

// Helper function to create a WebSocket session
func createWebSocketSession(t *testing.T, client *WebSocketTestClient) string {
	config := common.SessionConfig{
		Proxy: "http://test:8080",
	}

	err := client.SendMessage(internal_websocket.CreateSessionMsg, "create-session", config)
	if err != nil {
		t.Fatalf("Failed to send create session message: %v", err)
	}

	// Read the create session response first
	createResponse, err := client.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read create session response: %v", err)
	}

	if createResponse.Type != internal_websocket.ResponseMessage {
		t.Fatalf("Expected response message, got %s", createResponse.Type)
	}

	var createResult map[string]string
	if err := json.Unmarshal(createResponse.Payload, &createResult); err != nil {
		t.Fatalf("Failed to unmarshal create session response: %v", err)
	}

	return createResult["session_id"]
}
