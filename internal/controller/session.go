package controller

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/Noooste/azuretls-api/internal/common"
	"github.com/Noooste/azuretls-client"
)

type SessionController struct {
	sessionManager common.SessionManager
}

func NewSessionController(sessionManager common.SessionManager) *SessionController {
	return &SessionController{
		sessionManager: sessionManager,
	}
}

// CreateSession creates a new session with optional configuration
func (c *SessionController) CreateSession(config *common.SessionConfig) (string, *azuretls.Session, error) {
	sessionID := common.GenerateSessionID()
	var session *azuretls.Session
	var err error

	if config != nil {
		session, err = c.sessionManager.CreateSessionWithConfig(sessionID, config)
	} else {
		session, err = c.sessionManager.CreateSession(sessionID)
	}

	if err != nil {
		return "", nil, fmt.Errorf("failed to create session: %w", err)
	}

	if session == nil {
		return "", nil, fmt.Errorf("session creation returned nil")
	}

	return sessionID, session, nil
}

// GetSession retrieves a session by ID
func (c *SessionController) GetSession(sessionID string) (*azuretls.Session, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID required")
	}

	session, exists := c.sessionManager.GetSession(sessionID)
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	return session, nil
}

// DeleteSession removes a session
func (c *SessionController) DeleteSession(sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session ID required")
	}

	return c.sessionManager.DeleteSession(sessionID)
}

// ListSessions returns all active session IDs
func (c *SessionController) ListSessions() []string {
	return c.sessionManager.ListSessions()
}

// ExecuteRequest processes a request using the specified session
func (c *SessionController) ExecuteRequest(sessionID string, serverReq *common.ServerRequest) *common.ServerResponse {
	serverResp := &common.ServerResponse{
		ID: serverReq.ID,
	}

	session, err := c.GetSession(sessionID)
	if err != nil {
		serverResp.Error = err.Error()
		return serverResp
	}

	return c.executeRequestWithSession(session, serverReq)
}

// ExecuteStatelessRequest creates a temporary session and executes the request
func (c *SessionController) ExecuteStatelessRequest(serverReq *common.ServerRequest) *common.ServerResponse {
	tempSessionID := common.GenerateSessionID()
	session, err := c.sessionManager.CreateSession(tempSessionID)
	if err != nil {
		return &common.ServerResponse{
			ID:    serverReq.ID,
			Error: fmt.Sprintf("Failed to create temporary session: %v", err),
		}
	}

	defer func(sessionManager common.SessionManager, sessionID string) {
		err := sessionManager.DeleteSession(sessionID)
		if err != nil {
			fmt.Printf("Failed to delete temporary session %s: %v\n", sessionID, err)
		}
	}(c.sessionManager, tempSessionID)

	return c.executeRequestWithSession(session, serverReq)
}

// executeRequestWithSession handles the actual request execution
func (c *SessionController) executeRequestWithSession(session *azuretls.Session, serverReq *common.ServerRequest) *common.ServerResponse {
	serverResp := &common.ServerResponse{
		ID: serverReq.ID,
	}

	if serverReq.Body != "" && serverReq.BodyB64 != nil {
		serverResp.Error = "Both `body` and `body_b64` cannot be set"
		return serverResp
	}

	azureReq := &azuretls.Request{
		Method: serverReq.Method,
		Url:    serverReq.URL,
		Body:   serverReq.Body,
	}

	// Handle base64 encoded body
	if serverReq.BodyB64 != nil {
		azureReq.Body = serverReq.BodyB64
	} else if serverReq.Body != "" {
		azureReq.Body = serverReq.Body
	}

	// Handle headers
	if len(serverReq.OrderedHeaders) > 0 {
		azureReq.OrderedHeaders = make(azuretls.OrderedHeaders, len(serverReq.OrderedHeaders))
		for i, header := range serverReq.OrderedHeaders {
			azureReq.OrderedHeaders[i] = header
		}
	} else if len(serverReq.Headers.Keys) > 0 {
		azureReq.Header = make(map[string][]string)
		for _, value := range serverReq.Headers.Keys {
			if value == "Keys" || value == "Values" {
				continue
			}

			switch v := serverReq.Headers.Values[value].(type) {
			case string:
				azureReq.Header[value] = []string{v}
			case []string:
				azureReq.Header[value] = v
			default:
				serverResp.Error = fmt.Sprintf("Invalid header value type for key %s of type %T", value, v)
				return serverResp
			}
		}
	}

	if err := c.applyRequestOptions(azureReq, session, &serverReq.Options); err != nil {
		serverResp.Error = fmt.Sprintf("Failed to apply request options: %v", err)
		return serverResp
	}

	resp, err := session.Do(azureReq)
	if err != nil {
		serverResp.Error = err.Error()
		return serverResp
	}

	serverResp.StatusCode = resp.StatusCode
	serverResp.Status = resp.Status
	serverResp.URL = resp.Url

	// Handle response body
	if resp.Body != nil {
		if !common.IsBinaryContent(http.Header(resp.Header), resp.Body) {
			serverResp.Body = string(resp.Body)
			return serverResp
		}

		// For binary content, encode body as base64
		serverResp.BodyB64 = base64.StdEncoding.EncodeToString(resp.Body)
	}

	if resp.Header != nil {
		serverResp.Headers = make(map[string][]string)
		for key, values := range resp.Header {
			serverResp.Headers[key] = values
		}
	}

	if len(resp.Cookies) > 0 {
		serverResp.Cookies = make([]common.Cookie, len(resp.Cookies))
		for i, cookie := range resp.HttpResponse.Cookies() {
			serverResp.Cookies[i] = common.Cookie{
				Name:     cookie.Name,
				Value:    cookie.Value,
				Domain:   cookie.Domain,
				Path:     cookie.Path,
				Expires:  cookie.Expires,
				Secure:   cookie.Secure,
				HttpOnly: cookie.HttpOnly,
			}

			switch cookie.SameSite {
			case 1: // http.SameSiteDefaultMode
				serverResp.Cookies[i].SameSite = "Default"
			case 2: // http.SameSiteLaxMode
				serverResp.Cookies[i].SameSite = "Lax"
			case 3: // http.SameSiteStrictMode
				serverResp.Cookies[i].SameSite = "Strict"
			case 4: // http.SameSiteNoneMode
				serverResp.Cookies[i].SameSite = "None"
			}
		}
	}

	return serverResp
}

func (c *SessionController) applyRequestOptions(req *azuretls.Request, sess *azuretls.Session, options *common.RequestOptions) error {
	if options.TimeoutMs > 0 {
		req.TimeOut = time.Duration(options.TimeoutMs) * time.Millisecond
	}

	if options.Proxy != "" {
		if sess.Proxy == "" || sess.Proxy != options.Proxy {
			err := sess.SetProxy(options.Proxy)
			if err != nil {
				return err
			}
		}
	}

	req.ForceHTTP1 = options.ForceHTTP1
	req.ForceHTTP3 = options.ForceHTTP3
	req.InsecureSkipVerify = options.InsecureSkipVerify
	req.NoCookie = options.NoCookie
	req.IgnoreBody = options.IgnoreBody
	req.DisableRedirects = options.DisableRedirects

	if options.MaxRedirects > 0 {
		req.MaxRedirects = options.MaxRedirects
	}

	if options.Browser != "" {
		if sess.Browser != options.Browser {
			sess.Browser = options.Browser
		}
	}

	return nil
}

// ApplyJA3 applies JA3 fingerprint to a session
func (c *SessionController) ApplyJA3(sessionID, ja3, navigator string) error {
	if navigator == "" {
		navigator = azuretls.Chrome
	}

	return c.sessionManager.ApplyJA3(sessionID, ja3, navigator)
}

// ApplyHTTP2 applies HTTP2 fingerprint to a session
func (c *SessionController) ApplyHTTP2(sessionID, fingerprint string) error {
	return c.sessionManager.ApplyHTTP2(sessionID, fingerprint)
}

// ApplyHTTP3 applies HTTP3 fingerprint to a session
func (c *SessionController) ApplyHTTP3(sessionID, fingerprint string) error {
	return c.sessionManager.ApplyHTTP3(sessionID, fingerprint)
}

// SetProxy sets proxy for a session
func (c *SessionController) SetProxy(sessionID, proxy string) error {
	return c.sessionManager.SetProxy(sessionID, proxy)
}

// ClearProxy clears proxy for a session
func (c *SessionController) ClearProxy(sessionID string) error {
	return c.sessionManager.ClearProxy(sessionID)
}

// AddPins adds certificate pins for a URL in a session
func (c *SessionController) AddPins(sessionID, urlStr string, pins []string) error {
	return c.sessionManager.AddPins(sessionID, urlStr, pins)
}

// ClearPins clears certificate pins for a URL in a session
func (c *SessionController) ClearPins(sessionID, urlStr string) error {
	return c.sessionManager.ClearPins(sessionID, urlStr)
}

// GetIP gets the IP address used by a session
func (c *SessionController) GetIP(sessionID string) (string, error) {
	return c.sessionManager.GetIP(sessionID)
}

// GetHealthInfo returns health information including session count
func (c *SessionController) GetHealthInfo() map[string]any {
	sessions := c.ListSessions()

	azuretlsVersion := "unknown"
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
			if dep.Path == "github.com/Noooste/azuretls-client" {
				azuretlsVersion = dep.Version
				break
			}
		}
	}

	return map[string]any{
		"status":           "healthy",
		"sessions":         len(sessions),
		"timestamp":        time.Now().UTC(),
		"version":          "v0.0.0",
		"azuretls_version": azuretlsVersion,
	}
}
