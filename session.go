package api

import (
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/Noooste/azuretls-api/common"
	"github.com/Noooste/azuretls-client"
)

type DefaultSessionManager struct {
	sessions map[string]*azuretls.Session
	mu       sync.RWMutex
}

func (sm *DefaultSessionManager) ApplyJA3(sessionID, ja3, navigator string) error {
	sm.mu.RLock()
	session, exists := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("session with ID %s not found", sessionID)
	}

	return session.ApplyJa3(ja3, navigator)
}

func (sm *DefaultSessionManager) ApplyHTTP2(sessionID, fingerprint string) error {
	sm.mu.RLock()
	session, exists := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("session with ID %s not found", sessionID)
	}

	return session.ApplyHTTP2(fingerprint)
}

func (sm *DefaultSessionManager) ApplyHTTP3(sessionID, fingerprint string) error {
	sm.mu.RLock()
	session, exists := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("session with ID %s not found", sessionID)
	}

	return session.ApplyHTTP3(fingerprint)
}

func (sm *DefaultSessionManager) SetProxy(sessionID, proxy string) error {
	sm.mu.RLock()
	session, exists := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("session with ID %s not found", sessionID)
	}

	return session.SetProxy(proxy)
}

func (sm *DefaultSessionManager) ClearProxy(sessionID string) error {
	sm.mu.RLock()
	session, exists := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("session with ID %s not found", sessionID)
	}

	session.ClearProxy()
	return nil
}

func (sm *DefaultSessionManager) AddPins(sessionID, urlStr string, pins []string) error {
	sm.mu.RLock()
	session, exists := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("session with ID %s not found", sessionID)
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	return session.AddPins(parsedURL, pins)
}

func (sm *DefaultSessionManager) ClearPins(sessionID, urlStr string) error {
	sm.mu.RLock()
	session, exists := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("session with ID %s not found", sessionID)
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	return session.ClearPins(parsedURL)
}

func (sm *DefaultSessionManager) GetIP(sessionID string) (string, error) {
	sm.mu.RLock()
	session, exists := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("session with ID %s not found", sessionID)
	}

	return session.Ip()
}

func NewSessionManager() *DefaultSessionManager {
	return &DefaultSessionManager{
		sessions: make(map[string]*azuretls.Session),
	}
}

func (sm *DefaultSessionManager) CreateSession(sessionID string) (*azuretls.Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sessionID == "" {
		sessionID = common.GenerateSessionID()
	}

	if _, exists := sm.sessions[sessionID]; exists {
		return nil, fmt.Errorf("session with ID %s already exists", sessionID)
	}

	session := azuretls.NewSession()
	sm.sessions[sessionID] = session

	return session, nil
}

func (sm *DefaultSessionManager) GetSession(sessionID string) (*azuretls.Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	return session, exists
}

func (sm *DefaultSessionManager) DeleteSession(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session with ID %s not found", sessionID)
	}

	session.Close()
	delete(sm.sessions, sessionID)

	return nil
}

func (sm *DefaultSessionManager) ListSessions() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessionIDs := make([]string, 0, len(sm.sessions))
	for id := range sm.sessions {
		sessionIDs = append(sessionIDs, id)
	}

	return sessionIDs
}

func (sm *DefaultSessionManager) CleanupSessions() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for id, session := range sm.sessions {
		session.Close()
		delete(sm.sessions, id)
	}

	return nil
}

func (sm *DefaultSessionManager) CreateSessionWithConfig(sessionID string, config *common.SessionConfig) (*azuretls.Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sessionID == "" {
		sessionID = common.GenerateSessionID()
	}

	if _, exists := sm.sessions[sessionID]; exists {
		return nil, fmt.Errorf("session with ID %s already exists", sessionID)
	}

	session := azuretls.NewSession()

	// Apply configuration if provided
	if config != nil {
		if config.Browser != "" {
			session.Browser = config.Browser
		}
		if config.UserAgent != "" {
			session.UserAgent = config.UserAgent
		}
		if config.Proxy != "" {
			if err := session.SetProxy(config.Proxy); err != nil {
				return nil, fmt.Errorf("failed to set proxy: %w", err)
			}
		}
		if config.TimeoutMs > 0 {
			session.SetTimeout(time.Duration(config.TimeoutMs) * time.Millisecond)
		}
		if config.MaxRedirects > 0 {
			session.MaxRedirects = config.MaxRedirects
		}
		session.InsecureSkipVerify = config.InsecureSkipVerify

		if len(config.OrderedHeaders) > 0 {
			session.OrderedHeaders = make(azuretls.OrderedHeaders, len(config.OrderedHeaders))
			for i, header := range config.OrderedHeaders {
				session.OrderedHeaders[i] = header
			}
		}

		if len(config.Headers) > 0 {
			for k, v := range config.Headers {
				session.Header.Set(k, v)
			}
		}
	}

	sm.sessions[sessionID] = session
	return session, nil
}

// GenerateSessionID is deprecated, use common.GenerateSessionID instead
func GenerateSessionID() string {
	return common.GenerateSessionID()
}
