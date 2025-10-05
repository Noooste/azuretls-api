package rest

import (
	"net/http"

	"github.com/Noooste/azuretls-api/internal/common"
	"github.com/Noooste/azuretls-api/internal/websocket"
	"github.com/gorilla/mux"
)

func SetupRoutes(server common.Server) http.Handler {
	r := mux.NewRouter()
	handler := NewRESTHandler(server)
	wsHandler := websocket.NewWSHandler(server)

	// Health check
	r.HandleFunc("/health", handler.Health).Methods(http.MethodGet)

	// WebSocket endpoint
	r.HandleFunc("/ws", wsHandler.ServeHTTP)

	// Session management
	r.HandleFunc("/api/v1/session/create", handler.CreateSession).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/session/{id}", handler.DeleteSession).Methods(http.MethodDelete)

	// Session request
	r.HandleFunc("/api/v1/session/{id}/request", handler.SessionRequest).Methods(http.MethodPost)

	// Stateless request
	r.HandleFunc("/api/v1/request", handler.StatelessRequest).Methods(http.MethodPost)

	// Advanced session management endpoints
	r.HandleFunc("/api/v1/session/{id}/ja3", handler.ApplyJA3).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/session/{id}/http2", handler.ApplyHTTP2).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/session/{id}/http3", handler.ApplyHTTP3).Methods(http.MethodPost)

	// Proxy management
	r.HandleFunc("/api/v1/session/{id}/proxy", handler.ManageProxy).Methods(http.MethodPost, http.MethodDelete)

	// Pin management
	r.HandleFunc("/api/v1/session/{id}/pins", handler.ManagePins).Methods(http.MethodPost, http.MethodDelete)

	// Get IP
	r.HandleFunc("/api/v1/session/{id}/ip", handler.GetIP).Methods(http.MethodGet)

	config := server.GetConfig()
	middleware := ChainMiddleware(
		RequestIDMiddleware,
		RecoveryMiddleware,
		LoggingMiddleware,
		JSONContentTypeMiddleware,
		ConcurrentRequestLimiter(config.MaxConcurrentRequests),
	)

	return middleware(r)
}
