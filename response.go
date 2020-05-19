package smis

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
)

const (
	// HeaderKeyContentType represents the key in the header for the content type
	HeaderKeyContentType = "Content-Type"

	// HeaderContentTypeJSON represent the value for content type JSON in the header
	HeaderContentTypeJSON = "application/json"

	// HeaderContentTypePlain represent the value for content type text/plain in the header
	HeaderContentTypePlain = "text/plain; charset=utf-8"
)

//Response provides functions to write http responses.
type Response struct {
	Log logrus.FieldLogger
}

func (r *Response) logError(msg string) {
	if r == nil || r.Log == nil {
		return
	}

	r.Log.Error(msg)
}

func (r *Response) logWarning(msg string) {
	if r == nil || r.Log == nil {
		return
	}

	r.Log.Warn(msg)
}

//WriteJSON sends a JSON response with given code and payload.
func (r *Response) WriteJSON(writer http.ResponseWriter, code int, payload interface{}) {
	if writer == nil {
		r.logError("writer is nil")
		return
	}

	writer.Header().Set(HeaderKeyContentType, HeaderContentTypeJSON)

	response, err := json.Marshal(payload)
	if err != nil {
		se := ErrResponseJSONConversion.WithDetails(err)
		r.WriteJSONError(writer, se)

		return
	}

	writer.WriteHeader(code)

	if _, err := writer.Write(response); err != nil {
		r.logError(fmt.Sprintf("failed to write response: %v", err))
	}
}

// WriteJSONError generalize sending error responses to client.
func (r *Response) WriteJSONError(writer http.ResponseWriter, responseErr Error) {
	if responseErr.StatusCode < http.StatusBadRequest {
		r.logWarning(
			fmt.Sprintf(
				"status code for error response should be of 4xx or 5xx but used %d", responseErr.StatusCode,
			),
		)
	}

	payload, err := json.Marshal(responseErr)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		writer.Header().Set(HeaderKeyContentType, HeaderContentTypePlain)
		_, _ = writer.Write([]byte(responseErr.String()))

		r.logError(fmt.Sprintf("failed to encode response payload: %v", err))

		return
	}

	writer.WriteHeader(responseErr.StatusCode)
	writer.Header().Set(HeaderKeyContentType, HeaderContentTypeJSON)
	_, _ = writer.Write(payload)

	r.logError(responseErr.GetInternal())
}
