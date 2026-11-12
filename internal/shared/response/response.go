// Package response provides consistent JSON and error responses.
package response

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"backend/internal/shared/apperr"
)

// ErrorBody is the standard error envelope.
type ErrorBody struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Details   map[string]string `json:"details,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
}

// ErrorResponse is the top-level error payload.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// JSON writes a JSON response with the given status.
func JSON(w http.ResponseWriter, requestID string, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Request-ID", requestID)
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		// The response headers are already written; nothing safe to do here.
		return
	}
}

// WriteError maps an error to a safe JSON response.
func WriteError(w http.ResponseWriter, requestID string, err error) {
	appErr, ok := apperr.As(err)
	if !ok {
		appErr = apperr.Internal(err)
	}
	JSON(w, requestID, appErr.Status, ErrorResponse{
		Error: ErrorBody{
			Code:      appErr.Code,
			Message:   appErr.Message,
			Details:   appErr.Details,
			RequestID: requestID,
		},
	})
}

// Decode strictly decodes a JSON request body with a size limit.
func Decode(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return apperr.BadRequest(apperr.CodeValidation, "The request body is invalid JSON or contains unsupported fields.")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return apperr.BadRequest(apperr.CodeValidation, "The request body must contain a single JSON object.")
	}
	return nil
}
