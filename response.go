package smis

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
)

//Response provides functions to write http responses
type Response struct {
	Log logrus.FieldLogger
}

func (r *Response) logError(msg string) {
	if r == nil || r.Log == nil {
		return
	}

	r.Log.Error(msg)
}

//WriteJSON sends a JSON response with given code and payload
func (r *Response) WriteJSON(writer http.ResponseWriter, code int, payload interface{}) {
	if r == nil {
		return
	}

	response, err := json.Marshal(payload)
	if err != nil {
		msg := errorJSON{Error: fmt.Sprintf("failed to encode response payload: %v", err)}
		r.logError(msg.Error)
		writer.WriteHeader(http.StatusInternalServerError)

		response, _ = json.Marshal(msg)
		if _, err := writer.Write(response); err != nil {
			r.logError(fmt.Sprintf("failed to write response: %v", err))
		}

		return
	}

	writer.WriteHeader(code)

	if _, err := writer.Write(response); err != nil {
		r.logError(fmt.Sprintf("failed to write response: %v", err))
	}
}

type errorJSON struct {
	Error string `json:"error"`
}
