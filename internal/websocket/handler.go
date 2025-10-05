package websocket

import (
	"log"

	http "net/http"

	"github.com/Noooste/azuretls-api/internal/common"
	"github.com/Noooste/azuretls-api/internal/controller"
	"github.com/Noooste/azuretls-api/internal/protocol"
	"github.com/gorilla/websocket"
)

type WSHandler struct {
	controller  *controller.SessionController
	connManager *ConnectionManager
	connHandler *ConnectionHandler
	upgrader    websocket.Upgrader
	jsonEncoder protocol.MessageEncoder
}

func NewWSHandler(server common.Server) *WSHandler {
	connManager := NewConnectionManager()

	handler := &WSHandler{
		controller:  controller.NewSessionController(server.GetSessionManager()),
		connManager: connManager,
		jsonEncoder: protocol.GetJSONEncoder(),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}

	handler.connHandler = NewConnectionHandler(connManager, handler.handleMessage)
	return handler
}

func (h *WSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	wsConn := NewWSConnection(conn, "")

	ctx := r.Context()
	go func() {
		defer func() {
			if sessionID := wsConn.SessionID(); sessionID != "" {
				_ = h.controller.DeleteSession(sessionID)
			}
		}()

		h.connHandler.HandleConnection(ctx, wsConn)
	}()
}

func (h *WSHandler) handleMessage(conn *WSConnection, message *WSMessage) error {
	switch message.Type {
	case RequestMessage:
		return h.handleRequestMessage(conn, message)
	case PingMessage:
		return h.handlePingMessage(conn, message)
	case CreateSessionMsg:
		return h.handleCreateSession(conn, message)
	case DeleteSessionMsg:
		return h.handleDeleteSession(conn, message)
	case ApplyJA3Msg:
		return h.handleApplyJA3(conn, message)
	case ApplyHTTP2Msg:
		return h.handleApplyHTTP2(conn, message)
	case ApplyHTTP3Msg:
		return h.handleApplyHTTP3(conn, message)
	case SetProxyMsg:
		return h.handleSetProxy(conn, message)
	case ClearProxyMsg:
		return h.handleClearProxy(conn, message)
	case AddPinsMsg:
		return h.handleAddPins(conn, message)
	case ClearPinsMsg:
		return h.handleClearPins(conn, message)
	case GetIPMsg:
		return h.handleGetIP(conn, message)
	case HealthMsg:
		return h.handleHealth(conn, message)
	default:
		return conn.SendError(message.ID, "Unknown message type")
	}
}

func (h *WSHandler) handleRequestMessage(conn *WSConnection, message *WSMessage) error {
	var serverReq common.ServerRequest
	if err := h.jsonEncoder.Decode(message.Payload, &serverReq); err != nil {
		return conn.SendError(message.ID, "Invalid request payload: "+err.Error())
	}

	if message.ID != "" {
		serverReq.ID = message.ID
	}

	serverResp := h.controller.ExecuteRequest(conn.SessionID(), &serverReq)

	// If the response contains an error, send it as an error message
	if serverResp.Error != "" {
		return conn.SendError(message.ID, serverResp.Error)
	}

	return conn.SendResponse(message.ID, serverResp)
}

func (h *WSHandler) handlePingMessage(conn *WSConnection, message *WSMessage) error {
	pongMessage := WSMessage{
		Type: PongMessage,
		ID:   message.ID,
	}
	return conn.WriteJSON(pongMessage)
}

func (h *WSHandler) GetConnectionManager() *ConnectionManager {
	return h.connManager
}

func (h *WSHandler) CloseAllConnections() {
	h.connManager.CloseAll()
}

func (h *WSHandler) handleCreateSession(conn *WSConnection, message *WSMessage) error {
	var config common.SessionConfig
	if len(message.Payload) > 0 {
		if err := h.jsonEncoder.Decode(message.Payload, &config); err != nil {
			return conn.SendError(message.ID, "Invalid session config: "+err.Error())
		}
	}

	sessionID, _, err := h.controller.CreateSession(&config)
	if err != nil {
		return conn.SendError(message.ID, "Failed to create session: "+err.Error())
	}

	oldSessionID := conn.SessionID()
	conn.SetSessionID(sessionID)
	h.connManager.UpdateSessionMapping(conn, oldSessionID, sessionID)

	response := map[string]string{
		"session_id": sessionID,
		"status":     "created",
	}

	return conn.SendResponse(message.ID, response)
}

func (h *WSHandler) handleDeleteSession(conn *WSConnection, message *WSMessage) error {
	sessionID := conn.SessionID()
	if sessionID == "" {
		return conn.SendError(message.ID, "No active session")
	}

	if err := h.controller.DeleteSession(sessionID); err != nil {
		return conn.SendError(message.ID, "Failed to delete session: "+err.Error())
	}

	oldSessionID := conn.SessionID()
	conn.SetSessionID("")
	h.connManager.UpdateSessionMapping(conn, oldSessionID, "")

	return conn.SendSuccess(message.ID)
}

func (h *WSHandler) handleApplyJA3(conn *WSConnection, message *WSMessage) error {
	sessionID := conn.SessionID()
	if sessionID == "" {
		return conn.SendError(message.ID, "No active session")
	}

	var payload struct {
		JA3       string `json:"ja3"`
		Navigator string `json:"navigator,omitempty"`
	}

	if err := h.jsonEncoder.Decode(message.Payload, &payload); err != nil {
		return conn.SendError(message.ID, "Invalid JA3 payload: "+err.Error())
	}

	if err := h.controller.ApplyJA3(sessionID, payload.JA3, payload.Navigator); err != nil {
		return conn.SendError(message.ID, "Failed to apply JA3: "+err.Error())
	}

	return conn.SendSuccess(message.ID)
}

func (h *WSHandler) handleApplyHTTP2(conn *WSConnection, message *WSMessage) error {
	sessionID := conn.SessionID()
	if sessionID == "" {
		return conn.SendError(message.ID, "No active session")
	}

	var payload struct {
		Fingerprint string `json:"fingerprint"`
	}

	if err := h.jsonEncoder.Decode(message.Payload, &payload); err != nil {
		return conn.SendError(message.ID, "Invalid HTTP2 payload: "+err.Error())
	}

	if err := h.controller.ApplyHTTP2(sessionID, payload.Fingerprint); err != nil {
		return conn.SendError(message.ID, "Failed to apply HTTP2: "+err.Error())
	}

	return conn.SendSuccess(message.ID)
}

func (h *WSHandler) handleApplyHTTP3(conn *WSConnection, message *WSMessage) error {
	sessionID := conn.SessionID()
	if sessionID == "" {
		return conn.SendError(message.ID, "No active session")
	}

	var payload struct {
		Fingerprint string `json:"fingerprint"`
	}

	if err := h.jsonEncoder.Decode(message.Payload, &payload); err != nil {
		return conn.SendError(message.ID, "Invalid HTTP3 payload: "+err.Error())
	}

	if err := h.controller.ApplyHTTP3(sessionID, payload.Fingerprint); err != nil {
		return conn.SendError(message.ID, "Failed to apply HTTP3: "+err.Error())
	}

	return conn.SendSuccess(message.ID)
}

func (h *WSHandler) handleSetProxy(conn *WSConnection, message *WSMessage) error {
	sessionID := conn.SessionID()
	if sessionID == "" {
		return conn.SendError(message.ID, "No active session")
	}

	var payload struct {
		Proxy string `json:"proxy"`
	}

	if err := h.jsonEncoder.Decode(message.Payload, &payload); err != nil {
		return conn.SendError(message.ID, "Invalid proxy payload: "+err.Error())
	}

	if err := h.controller.SetProxy(sessionID, payload.Proxy); err != nil {
		return conn.SendError(message.ID, "Failed to set proxy: "+err.Error())
	}

	return conn.SendSuccess(message.ID)
}

func (h *WSHandler) handleClearProxy(conn *WSConnection, message *WSMessage) error {
	sessionID := conn.SessionID()
	if sessionID == "" {
		return conn.SendError(message.ID, "No active session")
	}

	if err := h.controller.ClearProxy(sessionID); err != nil {
		return conn.SendError(message.ID, "Failed to clear proxy: "+err.Error())
	}

	return conn.SendSuccess(message.ID)
}

func (h *WSHandler) handleAddPins(conn *WSConnection, message *WSMessage) error {
	sessionID := conn.SessionID()
	if sessionID == "" {
		return conn.SendError(message.ID, "No active session")
	}

	var payload struct {
		URL  string   `json:"url"`
		Pins []string `json:"pins"`
	}

	if err := h.jsonEncoder.Decode(message.Payload, &payload); err != nil {
		return conn.SendError(message.ID, "Invalid pins payload: "+err.Error())
	}

	if err := h.controller.AddPins(sessionID, payload.URL, payload.Pins); err != nil {
		return conn.SendError(message.ID, "Failed to add pins: "+err.Error())
	}

	return conn.SendSuccess(message.ID)
}

func (h *WSHandler) handleClearPins(conn *WSConnection, message *WSMessage) error {
	sessionID := conn.SessionID()
	if sessionID == "" {
		return conn.SendError(message.ID, "No active session")
	}

	var payload struct {
		URL string `json:"url"`
	}

	if err := h.jsonEncoder.Decode(message.Payload, &payload); err != nil {
		return conn.SendError(message.ID, "Invalid clear pins payload: "+err.Error())
	}

	if err := h.controller.ClearPins(sessionID, payload.URL); err != nil {
		return conn.SendError(message.ID, "Failed to clear pins: "+err.Error())
	}

	return conn.SendSuccess(message.ID)
}

func (h *WSHandler) handleGetIP(conn *WSConnection, message *WSMessage) error {
	sessionID := conn.SessionID()
	if sessionID == "" {
		return conn.SendError(message.ID, "No active session")
	}

	ip, err := h.controller.GetIP(sessionID)
	if err != nil {
		return conn.SendError(message.ID, "Failed to get IP: "+err.Error())
	}

	response := map[string]string{
		"ip": ip,
	}

	return conn.SendResponse(message.ID, response)
}

func (h *WSHandler) handleHealth(conn *WSConnection, message *WSMessage) error {
	response := h.controller.GetHealthInfo()
	return conn.SendResponse(message.ID, response)
}
