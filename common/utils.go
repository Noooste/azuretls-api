package common

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	mathRand "math/rand"
	"strings"
	"time"

	"github.com/Noooste/azuretls-api/protocol"
)

type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

var currentLogLevel = LogLevelInfo

// SetLogLevel sets the global log level
func SetLogLevel(level string) {
	switch strings.ToLower(level) {
	case "debug":
		currentLogLevel = LogLevelDebug
	case "info":
		currentLogLevel = LogLevelInfo
	case "warn", "warning":
		currentLogLevel = LogLevelWarn
	case "error":
		currentLogLevel = LogLevelError
	default:
		log.Printf("Unknown log level '%s', defaulting to 'info'", level)
		currentLogLevel = LogLevelInfo
	}
}

// LogDebug logs a debug message
func LogDebug(format string, v ...interface{}) {
	if currentLogLevel <= LogLevelDebug {
		log.Printf("[DEBUG] "+format, v...)
	}
}

// LogInfo logs an info message
func LogInfo(format string, v ...interface{}) {
	if currentLogLevel <= LogLevelInfo {
		log.Printf("[INFO] "+format, v...)
	}
}

// LogWarn logs a warning message
func LogWarn(format string, v ...interface{}) {
	if currentLogLevel <= LogLevelWarn {
		log.Printf("[WARN] "+format, v...)
	}
}

// LogError logs an error message
func LogError(format string, v ...interface{}) {
	if currentLogLevel <= LogLevelError {
		log.Printf("[ERROR] "+format, v...)
	}
}

func GenerateSessionID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to a timestamp + random number based ID if crypto/rand fails
		r := mathRand.New(mathRand.NewSource(time.Now().UnixNano()))
		return fmt.Sprintf("session-%d-%d", time.Now().UnixNano(), r.Int63())
	}
	return hex.EncodeToString(bytes)
}

// ParseRequestBody reads and parses request body with protocol detection
func ParseRequestBody(body io.Reader, contentType string, target any) (protocol.MessageEncoder, error) {
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}

	encoder, err := protocol.DetectProtocol(contentType, bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("unsupported media type: %w", err)
	}

	if len(bodyBytes) > 0 {
		if err = encoder.Decode(bodyBytes, target); err != nil {
			return encoder, fmt.Errorf("invalid request body: %w", err)
		}
	}

	return encoder, nil
}

// ExtractSessionIDFromPath extracts session ID from URL path
func ExtractSessionIDFromPath(path, endpoint string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "session" && i+1 < len(parts) {
			if endpoint == "" {
				// For simple session extraction
				sessionPart := parts[i+1]
				if requestIndex := strings.Index(sessionPart, "/request"); requestIndex != -1 {
					return sessionPart[:requestIndex]
				}
				return sessionPart
			} else {
				// For endpoint-specific extraction
				if i+2 < len(parts) && parts[i+2] == endpoint {
					return parts[i+1]
				}
			}
		}
	}
	return ""
}

// OrderedMap preserves the order of JSON keys during unmarshaling
type OrderedMap struct {
	Keys   []string
	Values map[string]any
}

// UnmarshalJSON implements custom unmarshaling to preserve key order
func (om *OrderedMap) UnmarshalJSON(data []byte) error {
	// Initialize the map
	om.Values = make(map[string]any)
	om.Keys = []string{}

	// Use json.RawMessage to parse without losing order
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// We need a different approach - parse manually to preserve order
	decoder := json.NewDecoder(strings.NewReader(string(data)))

	// Expect opening brace
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if token != json.Delim('{') {
		return fmt.Errorf("expected {, got %v", token)
	}

	// Parse key-value pairs in order
	for decoder.More() {
		// Get key
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		key, ok := token.(string)
		if !ok {
			return fmt.Errorf("expected string key, got %T", token)
		}

		// Store key in order
		om.Keys = append(om.Keys, key)

		// Get value
		var value any
		if err := decoder.Decode(&value); err != nil {
			return err
		}
		om.Values[key] = value
	}

	// Expect closing brace
	token, err = decoder.Token()
	if err != nil {
		return err
	}
	if token != json.Delim('}') {
		return fmt.Errorf("expected }, got %v", token)
	}

	return nil
}
