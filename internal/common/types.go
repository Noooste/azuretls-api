package common

import (
	"time"

	"github.com/Noooste/azuretls-client"
)

type ServerRequest struct {
	ID             string            `json:"id"`
	Method         string            `json:"method"`
	URL            string            `json:"url"`
	Headers        map[string]string `json:"headers,omitempty"`
	OrderedHeaders [][]string        `json:"ordered_headers,omitempty"`
	Body           []byte            `json:"body,omitempty"`
	BodyB64        string            `json:"body_b64,omitempty"` // Base64 encoded binary body
	Options        RequestOptions    `json:"options,omitempty"`
}

type RequestOptions struct {
	TimeoutMs          int    `json:"timeout_ms,omitempty"`
	FollowRedirects    bool   `json:"follow_redirects,omitempty"`
	DisableRedirects   bool   `json:"disable_redirects,omitempty"`
	MaxRedirects       uint   `json:"max_redirects,omitempty"`
	Proxy              string `json:"proxy,omitempty"`
	NoCookie           bool   `json:"no_cookie,omitempty"`
	Browser            string `json:"browser,omitempty"`
	ForceHTTP1         bool   `json:"force_http1,omitempty"`
	ForceHTTP3         bool   `json:"force_http3,omitempty"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify,omitempty"`
	IgnoreBody         bool   `json:"ignore_body,omitempty"`
}

type ServerResponse struct {
	ID         string              `json:"id"`
	StatusCode int                 `json:"status_code"`
	Status     string              `json:"status"`
	Headers    map[string][]string `json:"headers"`
	Body       string              `json:"body"`
	Cookies    []Cookie            `json:"cookies,omitempty"`
	Error      string              `json:"error,omitempty"`
	URL        string              `json:"url"`
}

type Cookie struct {
	Name     string    `json:"name"`
	Value    string    `json:"value"`
	Domain   string    `json:"domain,omitempty"`
	Path     string    `json:"path,omitempty"`
	Expires  time.Time `json:"expires,omitempty"`
	Secure   bool      `json:"secure,omitempty"`
	HttpOnly bool      `json:"http_only,omitempty"`
	SameSite string    `json:"same_site,omitempty"`
}

type ServerConfig struct {
	Host                  string        `json:"host"`
	Port                  int           `json:"port"`
	MaxSessions           int           `json:"max_sessions"`
	MaxConcurrentRequests int           `json:"max_concurrent_requests"`
	ReadTimeout           time.Duration `json:"read_timeout"`
	WriteTimeout          time.Duration `json:"write_timeout"`
	LogLevel              string        `json:"log_level"`
}

type SessionConfig struct {
	Browser            string            `json:"browser,omitempty"`
	UserAgent          string            `json:"user_agent,omitempty"`
	Proxy              string            `json:"proxy,omitempty"`
	TimeoutMs          int               `json:"timeout_ms,omitempty"`
	MaxRedirects       uint              `json:"max_redirects,omitempty"`
	InsecureSkipVerify bool              `json:"insecure_skip_verify,omitempty"`
	OrderedHeaders     [][]string        `json:"ordered_headers,omitempty"`
	Headers            map[string]string `json:"headers,omitempty"`
}

type SessionManager interface {
	CreateSession(sessionID string) (*azuretls.Session, error)
	CreateSessionWithConfig(sessionID string, config *SessionConfig) (*azuretls.Session, error)
	GetSession(sessionID string) (*azuretls.Session, bool)
	DeleteSession(sessionID string) error
	ListSessions() []string
	CleanupSessions() error
	ApplyJA3(sessionID, ja3, navigator string) error
	ApplyHTTP2(sessionID, fingerprint string) error
	ApplyHTTP3(sessionID, fingerprint string) error
	SetProxy(sessionID, proxy string) error
	ClearProxy(sessionID string) error
	AddPins(sessionID, urlStr string, pins []string) error
	ClearPins(sessionID, urlStr string) error
	GetIP(sessionID string) (string, error)
}

type Server interface {
	GetConfig() ServerConfig
	GetSessionManager() SessionManager
}
