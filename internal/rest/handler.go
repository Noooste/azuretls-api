package rest

import (
	http "net/http"

	"github.com/Noooste/azuretls-api/internal/common"
	"github.com/Noooste/azuretls-api/internal/controller"
	"github.com/Noooste/azuretls-api/internal/view"
	"github.com/gorilla/mux"
)

type Handler struct {
	controller *controller.SessionController
	writer     *view.ResponseWriter
}

func NewRESTHandler(server common.Server) *Handler {
	return &Handler{
		controller: controller.NewSessionController(server.GetSessionManager()),
		writer:     view.NewResponseWriter(),
	}
}

func (h *Handler) CreateSession(w http.ResponseWriter, r *http.Request) {
	var config common.SessionConfig
	encoder, err := common.ParseRequestBody(r.Body, r.Header.Get("Content-Type"), &config)
	if err != nil {
		h.writer.WriteErrorResponse(w, err.Error(), http.StatusBadRequest, nil)
		return
	}

	sessionID, _, err := h.controller.CreateSession(&config)
	if err != nil {
		h.writer.WriteErrorResponse(w, err.Error(), http.StatusInternalServerError, encoder)
		return
	}

	response := map[string]string{
		"session_id": sessionID,
		"status":     "created",
	}

	h.writer.WriteCreatedResponse(w, response, encoder)
}

func (h *Handler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	if err := h.controller.DeleteSession(sessionID); err != nil {
		h.writer.WriteErrorResponse(w, err.Error(), http.StatusNotFound, nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) SessionRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	var serverReq common.ServerRequest
	encoder, err := common.ParseRequestBody(r.Body, r.Header.Get("Content-Type"), &serverReq)
	if err != nil {
		h.writer.WriteErrorResponse(w, err.Error(), http.StatusBadRequest, nil)
		return
	}

	serverResp := h.controller.ExecuteRequest(sessionID, &serverReq)

	statusCode := http.StatusOK
	if serverResp.Error != "" {
		statusCode = http.StatusInternalServerError
	}

	h.writer.WriteResponse(w, serverResp, statusCode, encoder)
}

func (h *Handler) StatelessRequest(w http.ResponseWriter, r *http.Request) {
	var serverReq common.ServerRequest
	encoder, err := common.ParseRequestBody(r.Body, r.Header.Get("Content-Type"), &serverReq)
	if err != nil {
		h.writer.WriteErrorResponse(w, err.Error(), http.StatusBadRequest, nil)
		return
	}

	serverResp := h.controller.ExecuteStatelessRequest(&serverReq)

	statusCode := http.StatusOK
	if serverResp.Error != "" {
		statusCode = http.StatusInternalServerError
	}

	h.writer.WriteResponse(w, serverResp, statusCode, encoder)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	response := h.controller.GetHealthInfo()
	h.writer.WriteJSONResponse(w, response, http.StatusOK)
}

// Advanced session management endpoints

func (h *Handler) ApplyJA3(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	var payload struct {
		JA3       string `json:"ja3"`
		Navigator string `json:"navigator,omitempty"`
	}

	_, err := common.ParseRequestBody(r.Body, r.Header.Get("Content-Type"), &payload)
	if err != nil {
		h.writer.WriteErrorResponse(w, err.Error(), http.StatusBadRequest, nil)
		return
	}

	if err := h.controller.ApplyJA3(sessionID, payload.JA3, payload.Navigator); err != nil {
		h.writer.WriteErrorResponse(w, err.Error(), http.StatusInternalServerError, nil)
		return
	}

	h.writer.WriteSuccessResponse(w)
}

func (h *Handler) ApplyHTTP2(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	var payload struct {
		Fingerprint string `json:"fingerprint"`
	}

	_, err := common.ParseRequestBody(r.Body, r.Header.Get("Content-Type"), &payload)
	if err != nil {
		h.writer.WriteErrorResponse(w, err.Error(), http.StatusBadRequest, nil)
		return
	}

	if err := h.controller.ApplyHTTP2(sessionID, payload.Fingerprint); err != nil {
		h.writer.WriteErrorResponse(w, err.Error(), http.StatusInternalServerError, nil)
		return
	}

	h.writer.WriteSuccessResponse(w)
}

func (h *Handler) ApplyHTTP3(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	var payload struct {
		Fingerprint string `json:"fingerprint"`
	}

	_, err := common.ParseRequestBody(r.Body, r.Header.Get("Content-Type"), &payload)
	if err != nil {
		h.writer.WriteErrorResponse(w, err.Error(), http.StatusBadRequest, nil)
		return
	}

	if err := h.controller.ApplyHTTP3(sessionID, payload.Fingerprint); err != nil {
		h.writer.WriteErrorResponse(w, err.Error(), http.StatusInternalServerError, nil)
		return
	}

	h.writer.WriteSuccessResponse(w)
}

func (h *Handler) ManageProxy(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	switch r.Method {
	case http.MethodPost:
		var payload struct {
			Proxy string `json:"proxy"`
		}

		_, err := common.ParseRequestBody(r.Body, r.Header.Get("Content-Type"), &payload)
		if err != nil {
			h.writer.WriteErrorResponse(w, err.Error(), http.StatusBadRequest, nil)
			return
		}

		if err := h.controller.SetProxy(sessionID, payload.Proxy); err != nil {
			h.writer.WriteErrorResponse(w, err.Error(), http.StatusInternalServerError, nil)
			return
		}

		h.writer.WriteSuccessResponse(w)

	case http.MethodDelete:
		if err := h.controller.ClearProxy(sessionID); err != nil {
			h.writer.WriteErrorResponse(w, err.Error(), http.StatusInternalServerError, nil)
			return
		}

		h.writer.WriteSuccessResponse(w)

	default:
		h.writer.WriteErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed, nil)
	}
}

func (h *Handler) ManagePins(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	switch r.Method {
	case http.MethodPost:
		var payload struct {
			URL  string   `json:"url"`
			Pins []string `json:"pins"`
		}

		_, err := common.ParseRequestBody(r.Body, r.Header.Get("Content-Type"), &payload)
		if err != nil {
			h.writer.WriteErrorResponse(w, err.Error(), http.StatusBadRequest, nil)
			return
		}

		if err := h.controller.AddPins(sessionID, payload.URL, payload.Pins); err != nil {
			h.writer.WriteErrorResponse(w, err.Error(), http.StatusInternalServerError, nil)
			return
		}

		h.writer.WriteSuccessResponse(w)

	case http.MethodDelete:
		var payload struct {
			URL string `json:"url"`
		}

		_, err := common.ParseRequestBody(r.Body, r.Header.Get("Content-Type"), &payload)
		if err != nil {
			h.writer.WriteErrorResponse(w, err.Error(), http.StatusBadRequest, nil)
			return
		}

		if err := h.controller.ClearPins(sessionID, payload.URL); err != nil {
			h.writer.WriteErrorResponse(w, err.Error(), http.StatusInternalServerError, nil)
			return
		}

		h.writer.WriteSuccessResponse(w)

	default:
		h.writer.WriteErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed, nil)
	}
}

func (h *Handler) GetIP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	ip, err := h.controller.GetIP(sessionID)
	if err != nil {
		h.writer.WriteErrorResponse(w, err.Error(), http.StatusInternalServerError, nil)
		return
	}

	response := map[string]string{
		"ip": ip,
	}

	h.writer.WriteJSONResponse(w, response, http.StatusOK)
}
