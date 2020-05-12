package smis

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

var (
	// ErrParseBody is a common error used on parsing errors for the body.
	ErrParseBody = errors.New("failed to parse body")
)

// ParseJSONRequestBody takes a request and parses the JSON body to the destination value.
// If it is not parsable it returns ErrParseBody.
func ParseJSONRequestBody(request *http.Request, destination interface{}) error {
	if request.Header == nil || !strings.Contains(request.Header.Get(HeaderKeyContentType), HeaderContentTypeJSON) {
		return fmt.Errorf("%w: body is not a JSON", ErrParseBody)
	}

	return ParseJSONBody(request.Body, destination)
}

// ParseJSONResponseBody takes a response and parses the JSON body to the destination value.
// If it is not parsable it returns ErrParseBody.
func ParseJSONResponseBody(response *http.Response, destination interface{}) error {
	if response.Header == nil || !strings.Contains(response.Header.Get(HeaderKeyContentType), HeaderContentTypeJSON) {
		return fmt.Errorf("%w: body is not a JSON", ErrParseBody)
	}

	return ParseJSONBody(response.Body, destination)
}

// ParseJSONBody parses a JSON body to the destination value.
// If it is not parsable it returns ErrParseBody.
func ParseJSONBody(body io.ReadCloser, destination interface{}) error {
	data, err := ioutil.ReadAll(body)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrParseBody, err)
	}

	if err = json.Unmarshal(data, destination); err != nil {
		return fmt.Errorf("%w: %v", ErrParseBody, err)
	}

	return nil
}
