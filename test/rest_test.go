package test_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Noooste/azuretls-api/common"
	"github.com/Noooste/azuretls-api/rest"
	"github.com/Noooste/azuretls-client"
	fhttp "github.com/Noooste/fhttp"
)

// TestServer represents a mock server for testing
type TestServer struct {
	*httptest.Server
	sessionManager common.SessionManager
}

func NewTestServer() *TestServer {
	sessionManager := &MockSessionManager{
		sessions: make(map[string]*azuretls.Session),
	}

	server := &TestAPIServer{sessionManager: sessionManager}
	fhttpRoutes := rest.SetupRoutes(server)

	// Convert fhttp.Handler to net/http.Handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		// Create an fhttp ResponseWriter wrapper
		fhttpW := &fhttpResponseWriter{ResponseWriter: w}

		fhttpRoutes.ServeHTTP(fhttpW, fhttpReq)
	})

	httpServer := httptest.NewServer(handler)

	return &TestServer{
		Server:         httpServer,
		sessionManager: sessionManager,
	}
}

// Test Functions

func TestRESTHealth(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	resp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("Failed to make health request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var health map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("Failed to decode health response: %v", err)
	}

	if health["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", health["status"])
	}
}

func TestRESTCreateSession(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	config := common.SessionConfig{
		Proxy: "http://proxy:8080",
	}
	body, _ := json.Marshal(config)

	resp, err := http.Post(server.URL+"/api/v1/session/create", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode create session response: %v", err)
	}

	if result["status"] != "created" {
		t.Errorf("Expected status 'created', got %v", result["status"])
	}

	if result["session_id"] == "" {
		t.Error("Expected session_id to be present")
	}
}

func TestRESTDeleteSession(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	// First create a session
	config := common.SessionConfig{}
	body, _ := json.Marshal(config)
	resp, err := http.Post(server.URL+"/api/v1/session/create", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	defer resp.Body.Close()

	var createResult map[string]string
	json.NewDecoder(resp.Body).Decode(&createResult)
	sessionID := createResult["session_id"]

	// Delete the session
	req, _ := http.NewRequest("DELETE", server.URL+"/api/v1/session/"+sessionID, nil)
	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Failed to delete session: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", resp.StatusCode)
	}
}

func TestRESTSessionRequest(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	// Create session first
	config := common.SessionConfig{}
	body, _ := json.Marshal(config)
	resp, err := http.Post(server.URL+"/api/v1/session/create", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	defer resp.Body.Close()

	var createResult map[string]string
	json.NewDecoder(resp.Body).Decode(&createResult)
	sessionID := createResult["session_id"]

	// Make session request
	serverReq := common.ServerRequest{
		URL:    "https://httpbin.org/get",
		Method: "GET",
	}
	body, _ = json.Marshal(serverReq)

	resp, err = http.Post(server.URL+"/api/v1/session/"+sessionID+"/request", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to make session request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestRESTStatelessRequest(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	serverReq := common.ServerRequest{
		URL:    "https://httpbin.org/get",
		Method: "GET",
	}
	body, _ := json.Marshal(serverReq)

	resp, err := http.Post(server.URL+"/api/v1/request", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to make stateless request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestRESTApplyJA3(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	// Create session first
	sessionID := createTestSession(t, server)

	payload := map[string]string{
		"ja3":       "test-ja3-string",
		"navigator": "test-navigator",
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(server.URL+"/api/v1/session/"+sessionID+"/ja3", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to apply JA3: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestRESTApplyHTTP2(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	sessionID := createTestSession(t, server)

	payload := map[string]string{
		"fingerprint": "test-http2-fingerprint",
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(server.URL+"/api/v1/session/"+sessionID+"/http2", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to apply HTTP2: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestRESTApplyHTTP3(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	sessionID := createTestSession(t, server)

	payload := map[string]string{
		"fingerprint": "test-http3-fingerprint",
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(server.URL+"/api/v1/session/"+sessionID+"/http3", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to apply HTTP3: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestRESTSetProxy(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	sessionID := createTestSession(t, server)

	payload := map[string]string{
		"proxy": "http://proxy:8080",
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(server.URL+"/api/v1/session/"+sessionID+"/proxy", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to set proxy: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestRESTClearProxy(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	sessionID := createTestSession(t, server)

	req, _ := http.NewRequest("DELETE", server.URL+"/api/v1/session/"+sessionID+"/proxy", nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to clear proxy: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestRESTAddPins(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	sessionID := createTestSession(t, server)

	payload := map[string]interface{}{
		"url":  "https://example.com",
		"pins": []string{"pin1", "pin2"},
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(server.URL+"/api/v1/session/"+sessionID+"/pins", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to add pins: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestRESTClearPins(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	sessionID := createTestSession(t, server)

	payload := map[string]string{
		"url": "https://example.com",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("DELETE", server.URL+"/api/v1/session/"+sessionID+"/pins", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to clear pins: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestRESTGetIP(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	sessionID := createTestSession(t, server)

	resp, err := http.Get(server.URL + "/api/v1/session/" + sessionID + "/ip")
	if err != nil {
		t.Fatalf("Failed to get IP: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode IP response: %v", err)
	}

	if result["ip"] != "192.168.1.1" {
		t.Errorf("Expected IP '192.168.1.1', got %v", result["ip"])
	}
}

func TestRESTInvalidSession(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	// Try to make request with invalid session ID
	serverReq := common.ServerRequest{
		URL:    "https://httpbin.org/get",
		Method: "GET",
	}
	body, _ := json.Marshal(serverReq)

	resp, err := http.Post(server.URL+"/api/v1/session/invalid-session/request", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}
}

func TestRESTMethodNotAllowed(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	// Try GET on create session endpoint (only POST allowed)
	resp, err := http.Get(server.URL + "/api/v1/session/create")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestRESTInvalidJSON(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	// Send invalid JSON
	resp, err := http.Post(server.URL+"/api/v1/session/create", "application/json", strings.NewReader("invalid json"))
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

// Helper function to create a test session
func createTestSession(t *testing.T, server *TestServer) string {
	config := common.SessionConfig{}
	body, _ := json.Marshal(config)
	resp, err := http.Post(server.URL+"/api/v1/session/create", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	return result["session_id"]
}
