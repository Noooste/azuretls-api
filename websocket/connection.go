package websocket

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	http "github.com/Noooste/fhttp"
	"github.com/Noooste/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024 // 512KB
)

type MessageHandler func(*WSConnection, *WSMessage) error

type ConnectionHandler struct {
	connManager    *ConnectionManager
	messageHandler MessageHandler
	upgrader       websocket.Upgrader
}

func NewConnectionHandler(connManager *ConnectionManager, messageHandler MessageHandler) *ConnectionHandler {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	return &ConnectionHandler{
		connManager:    connManager,
		messageHandler: messageHandler,
		upgrader:       upgrader,
	}
}

func (h *ConnectionHandler) HandleConnection(ctx context.Context, conn *WSConnection) {
	connID := generateConnectionID()
	h.connManager.AddConnection(connID, conn)

	defer func() {
		h.connManager.RemoveConnection(connID)
		log.Printf("WebSocket connection %s closed (session: %s)", connID, conn.SessionID())
	}()

	log.Printf("WebSocket connection %s established (session: %s)", connID, conn.SessionID())

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go h.writePump(ctx, conn)
	h.readPump(ctx, conn)
}

func (h *ConnectionHandler) readPump(ctx context.Context, conn *WSConnection) {
	defer func(conn *WSConnection) {
		_ = conn.Close()
	}(conn)

	conn.conn.SetReadLimit(maxMessageSize)
	_ = conn.conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.conn.SetPongHandler(func(string) error {
		_ = conn.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		select {
		case <-ctx.Done():
			return
		case <-conn.CloseChan():
			return
		default:
		}

		var message WSMessage
		err := conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error (session: %s): %v", conn.SessionID(), err)
			}
			break
		}

		if message.Type == PongMessage {
			continue
		}

		if h.messageHandler != nil {
			if err := h.messageHandler(conn, &message); err != nil {
				log.Printf("Message handler error (session: %s): %v", conn.SessionID(), err)

				errorMsg := WSMessage{
					Type:    ErrorMessage,
					ID:      message.ID,
					Payload: json.RawMessage(`{"error":"` + err.Error() + `"}`),
				}

				if writeErr := conn.WriteJSON(errorMsg); writeErr != nil {
					log.Printf("Error writing error message (session: %s): %v", conn.SessionID(), writeErr)
					break
				}
			}
		}
	}
}

func (h *ConnectionHandler) writePump(ctx context.Context, conn *WSConnection) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			_ = conn.conn.SetWriteDeadline(time.Now().Add(writeWait))
			_ = conn.conn.WriteMessage(websocket.CloseMessage, nil)
			return

		case <-conn.CloseChan():
			return

		case <-ticker.C:
			_ = conn.conn.SetWriteDeadline(time.Now().Add(writeWait))

			pingMsg := WSMessage{
				Type: PingMessage,
			}

			if err := conn.WriteJSON(pingMsg); err != nil {
				log.Printf("Error sending ping (session: %s): %v", conn.SessionID(), err)
				return
			}
		}
	}
}

func (c *WSConnection) SendMessage(msgType WSMessageType, id string, payload any) error {
	var payloadBytes json.RawMessage
	var err error

	if payload != nil {
		payloadBytes, err = json.Marshal(payload)
		if err != nil {
			return err
		}
	}

	message := WSMessage{
		Type:    msgType,
		ID:      id,
		Payload: payloadBytes,
	}

	return c.WriteJSON(message)
}

func (c *WSConnection) SendResponse(id string, payload any) error {
	return c.SendMessage(ResponseMessage, id, payload)
}

func (c *WSConnection) SendError(id string, errorMsg string) error {
	errorPayload := map[string]string{
		"error": errorMsg,
	}
	return c.SendMessage(ErrorMessage, id, errorPayload)
}

func (c *WSConnection) SendSessionInfo(sessionID string) error {
	sessionPayload := map[string]string{
		"session_id": sessionID,
	}
	return c.SendMessage(SessionMessage, "", sessionPayload)
}

func (c *WSConnection) SendSuccess(id string) error {
	successPayload := map[string]string{
		"status": "success",
	}
	return c.SendMessage(ResponseMessage, id, successPayload)
}

func generateConnectionID() string {
	bytes := make([]byte, 8) // 8 bytes = 16 hex characters
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to a simple timestamp-based ID
		return fmt.Sprintf("conn-%d", time.Now().UnixNano())
	}
	return "conn-" + hex.EncodeToString(bytes)
}
