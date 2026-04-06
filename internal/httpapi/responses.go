package httpapi

import (
	"encoding/json"
	"net/http"
)

type itemResponse struct {
	Data any `json:"data"`
}

type listMeta struct {
	Page     int32  `json:"page"`
	PageSize int32  `json:"page_size"`
	Total    int64  `json:"total"`
	Sort     string `json:"sort"`
	Order    string `json:"order"`
}

type listResponse struct {
	Data any      `json:"data"`
	Meta listMeta `json:"meta"`
}

type errorPayload struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	// Response helpers keep the handler layer focused on behavior, not repeated envelope wiring.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeData(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, itemResponse{Data: data})
}

func writeList(w http.ResponseWriter, data any, meta listMeta) {
	writeJSON(w, http.StatusOK, listResponse{Data: data, Meta: meta})
}

func writeError(w http.ResponseWriter, status int, code, message string, fields map[string]string) {
	writeJSON(w, status, errorPayload{Error: apiError{Code: code, Message: message, Fields: fields}})
}
