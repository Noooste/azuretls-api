package test_test

import (
	"fmt"
	"net/url"

	"github.com/Noooste/azuretls-api/internal/common"
	"github.com/Noooste/azuretls-client"
)

type TestAPIServer struct {
	sessionManager common.SessionManager
}

func (t *TestAPIServer) GetSessionManager() common.SessionManager {
	return t.sessionManager
}

func (t *TestAPIServer) GetConfig() common.ServerConfig {
	return common.ServerConfig{
		MaxConcurrentRequests: 100,
	}
}

// MockSessionManager implements common.SessionManager for testing
type MockSessionManager struct {
	sessions map[string]*azuretls.Session
}

func (m *MockSessionManager) CreateSession(sessionID string) (*azuretls.Session, error) {
	session := azuretls.NewSession()
	m.sessions[sessionID] = session
	return session, nil
}

func (m *MockSessionManager) CreateSessionWithConfig(sessionID string, config *common.SessionConfig) (*azuretls.Session, error) {
	session := azuretls.NewSession()
	m.sessions[sessionID] = session
	return session, nil
}

func (m *MockSessionManager) GetSession(sessionID string) (*azuretls.Session, bool) {
	session, exists := m.sessions[sessionID]
	return session, exists
}

func (m *MockSessionManager) DeleteSession(sessionID string) error {
	if session, exists := m.sessions[sessionID]; exists {
		session.Close()
	}
	delete(m.sessions, sessionID)
	return nil
}

func (m *MockSessionManager) ListSessions() []string {
	sessions := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		sessions = append(sessions, id)
	}
	return sessions
}

func (m *MockSessionManager) CleanupSessions() error {
	for _, session := range m.sessions {
		session.Close()
	}
	m.sessions = make(map[string]*azuretls.Session)
	return nil
}

func (m *MockSessionManager) GetSessionCount() int {
	return len(m.sessions)
}

func (m *MockSessionManager) GetHealthInfo() map[string]interface{} {
	return map[string]interface{}{
		"status":        "healthy",
		"session_count": len(m.sessions),
		"uptime":        "test",
	}
}

func (m *MockSessionManager) ExecuteRequest(sessionID string, req *common.ServerRequest) *common.ServerResponse {
	_, exists := m.sessions[sessionID]
	if !exists || sessionID == "" {
		return &common.ServerResponse{
			StatusCode: 500,
			Error:      "session not found",
		}
	}
	return &common.ServerResponse{
		StatusCode: 200,
		Body:       `{"test": "response"}`,
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
	}
}

func (m *MockSessionManager) ExecuteStatelessRequest(req *common.ServerRequest) *common.ServerResponse {
	return &common.ServerResponse{
		StatusCode: 200,
		Body:       `{"stateless": "response"}`,
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
	}
}

func (m *MockSessionManager) ApplyJA3(sessionID, ja3, navigator string) error {
	_, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}
	// Mock implementation - don't actually apply JA3 in tests
	return nil
}

func (m *MockSessionManager) ApplyHTTP2(sessionID, fingerprint string) error {
	_, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}
	// Mock implementation - don't actually apply HTTP2 in tests
	return nil
}

func (m *MockSessionManager) ApplyHTTP3(sessionID, fingerprint string) error {
	_, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}
	// Mock implementation - don't actually apply HTTP3 in tests
	return nil
}

func (m *MockSessionManager) SetProxy(sessionID, proxy string) error {
	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}
	return session.SetProxy(proxy)
}

func (m *MockSessionManager) ClearProxy(sessionID string) error {
	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}
	session.ClearProxy()
	return nil
}

func (m *MockSessionManager) AddPins(sessionID, urlStr string, pins []string) error {
	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return err
	}
	return session.AddPins(parsedURL, pins)
}

func (m *MockSessionManager) ClearPins(sessionID, urlStr string) error {
	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return err
	}
	return session.ClearPins(parsedURL)
}

func (m *MockSessionManager) GetIP(sessionID string) (string, error) {
	_, exists := m.sessions[sessionID]
	if !exists {
		return "", fmt.Errorf("session not found")
	}
	// Mock implementation - return a fixed IP for testing
	return "192.168.1.1", nil
}
