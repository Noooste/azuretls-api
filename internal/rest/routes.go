package rest

import (
	"strings"

	"net/http"

	"github.com/Noooste/azuretls-api/internal/websocket"

	"github.com/Noooste/azuretls-api/internal/common"
)

func SetupRoutes(server common.Server) http.Handler {
	mux := http.NewServeMux()
	handler := NewRESTHandler(server)
	wsHandler := websocket.NewWSHandler(server)

	mux.HandleFunc("/health", handler.Health)
	mux.HandleFunc("/ws", wsHandler.ServeHTTP)
	mux.HandleFunc("/api/v1/session/create", handler.CreateSession)
	mux.HandleFunc("/api/v1/session/", sessionRouteHandler(handler))
	mux.HandleFunc("/api/v1/request", handler.StatelessRequest)

	// Advanced session management endpoints
	mux.HandleFunc("/api/v1/session/{id}/ja3", handler.ApplyJA3)
	mux.HandleFunc("/api/v1/session/{id}/http2", handler.ApplyHTTP2)
	mux.HandleFunc("/api/v1/session/{id}/http3", handler.ApplyHTTP3)
	mux.HandleFunc("/api/v1/session/{id}/proxy", handler.ManageProxy)
	mux.HandleFunc("/api/v1/session/{id}/pins", handler.ManagePins)
	mux.HandleFunc("/api/v1/session/{id}/ip", handler.GetIP)

	config := server.GetConfig()
	middleware := ChainMiddleware(
		RequestIDMiddleware,
		RecoveryMiddleware,
		LoggingMiddleware,
		JSONContentTypeMiddleware,
		ConcurrentRequestLimiter(config.MaxConcurrentRequests),
	)

	return middleware(mux)
}

func sessionRouteHandler(handler *Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Handle specific endpoints
		if strings.HasSuffix(path, "/request") {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			handler.SessionRequest(w, r)
			return
		}

		if strings.Contains(path, "/ja3") {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			handler.ApplyJA3(w, r)
			return
		}

		if strings.Contains(path, "/http2") {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			handler.ApplyHTTP2(w, r)
			return
		}

		if strings.Contains(path, "/http3") {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			handler.ApplyHTTP3(w, r)
			return
		}

		if strings.Contains(path, "/proxy") {
			handler.ManageProxy(w, r)
			return
		}

		if strings.Contains(path, "/pins") {
			handler.ManagePins(w, r)
			return
		}

		if strings.Contains(path, "/ip") {
			if r.Method != http.MethodGet {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			handler.GetIP(w, r)
			return
		}

		// Handle session deletion
		sessionID := strings.TrimPrefix(path, "/api/v1/session/")
		sessionID = strings.TrimSuffix(sessionID, "/")

		if sessionID == "" {
			http.Error(w, "Session ID required", http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodDelete:
			handler.DeleteSession(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}
