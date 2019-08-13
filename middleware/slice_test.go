package middleware_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gorilla/mux"

	"github.com/rebel-l/smis/middleware"
)

func TestSlice_Walk(t *testing.T) {
	testCases := []struct {
		name          string
		slice         middleware.Slice
		walk          middleware.WalkCallback
		expectedError error
	}{
		{
			name: "success",
			slice: middleware.Slice{
				func(_ http.Handler) http.Handler {
					return nil
				},
			},
			walk: func(_ mux.MiddlewareFunc) error {
				return nil
			},
		},
		{
			name: "error",
			slice: middleware.Slice{
				func(_ http.Handler) http.Handler {
					return nil
				},
			},
			walk: func(_ mux.MiddlewareFunc) error {
				return fmt.Errorf("something happened")
			},
			expectedError: fmt.Errorf("something happened"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.slice.Walk(testCase.walk)
			if err != nil && testCase.expectedError != nil && err.Error() != testCase.expectedError.Error() {
				t.Errorf("expected error '%s' but got '%s'", testCase.expectedError, err)
			}

			if (err == nil && testCase.expectedError != nil) || (err != nil && testCase.expectedError == nil) {
				t.Error("expected error and/or got error is nil and should not be")
			}
		})
	}
}
