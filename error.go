package smis

import (
	"net/http"
)

const (
	separatorAfterCode     = " - "
	separatorBeforeDetails = ": "
)

var (
	// ErrResponseJSONConversion represents a standard error on JSON conversion.
	ErrResponseJSONConversion = Error{
		StatusCode: http.StatusInternalServerError,
		Code:       "SMIS-5001",
		External:   "a general issue occurred on preparing response",
		Internal:   "failed to parse JSON",
	}
)

// Error represents a struct for handling with error response from SMiS.
type Error struct {
	StatusCode int    `json:"-"`
	Code       string `json:"code"`
	External   string `json:"error"`
	Internal   string `json:"-"`
	Details    error  `json:"-"`
}

// GetInternal returns the internal error with code. If internal error is empty it takes external error.
// If external is also empty only code is returned. This is used for logging.
func (e Error) GetInternal() string {
	internal := e.getInternal()
	msg := internal

	if e.Details != nil {
		if e.Code == "" && msg == "" {
			msg += e.Details.Error()
		} else {
			msg += separatorBeforeDetails + e.Details.Error()
		}
	}

	if e.Code != "" {
		// TODO: avoid that deep nesting
		if msg != "" {
			if internal == "" {
				msg = e.Code + msg
			} else {
				msg = e.Code + separatorAfterCode + msg
			}
		} else {
			msg = e.Code
		}
	}

	return msg
}

func (e Error) getInternal() string {
	msg := e.External
	if e.Internal != "" {
		msg = e.Internal
	}

	return msg
}

// WithDetails returns a copy of the original error and extends it with a detail go error.
func (e Error) WithDetails(err error) Error {
	v := Error{
		StatusCode: e.StatusCode,
		Code:       e.Code,
		External:   e.External,
		Details:    err,
	}

	v.Internal = e.getInternal()

	return v
}

// String returns the external error with code. It's use to send plain text error message with response.
func (e Error) String() string {
	if e.Code != "" {
		if e.External != "" {
			return e.Code + separatorAfterCode + e.External
		}

		return e.Code
	}

	return e.External
}
