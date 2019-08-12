//go:generate mockgen -destination ./../tests/mocks/http_mock/handler.go -package http_mock net/http Handler

package cors_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rebel-l/go-utils/slice"

	"github.com/gorilla/mux"

	"github.com/rebel-l/smis/middleware/cors"

	"github.com/golang/mock/gomock"
	"github.com/rebel-l/smis/tests/mocks/http_mock"
)

func createHandler(ctrl *gomock.Controller) *http_mock.MockHandler {
	handler := http_mock.NewMockHandler(ctrl)
	handler.EXPECT().ServeHTTP(gomock.Any(), gomock.Any()).Times(1)
	return handler
}

func createOptionsHanlder(ctrl *gomock.Controller) *http_mock.MockHandler {
	handler := http_mock.NewMockHandler(ctrl)
	handler.EXPECT().ServeHTTP(gomock.Any(), gomock.Any()).Times(0)
	return handler
}

func TestNewCORS(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	reqOptions := httptest.NewRequest(http.MethodOptions, "/", nil)
	reqOptions.Header.Set("Origin", "http://example.com")

	reqPost := httptest.NewRequest(http.MethodPost, "/", nil)
	reqPost.Header.Set("Origin", "http://example.com")

	testCases := []struct {
		name            string
		request         *http.Request
		config          cors.Config
		nextHandler     http.Handler
		expectedOrigin  string
		expectedMethods string
		expectedHeaders string
		expectedMaxAge  string
	}{
		{
			name:    "options - allow",
			request: reqOptions,
			config: cors.Config{
				AccessControlAllowOrigins: slice.StringSlice{"http://example.com"},
				AccessContolAllowHeaders:  slice.StringSlice{"*"},
			},
			nextHandler:     createOptionsHanlder(ctrl),
			expectedOrigin:  "http://example.com",
			expectedMethods: "GET",
			expectedHeaders: "*",
			expectedMaxAge:  "86400",
		},
		{
			name:    "options - forbidden",
			request: reqOptions,
			config: cors.Config{
				AccessControlAllowOrigins: slice.StringSlice{"http://example.com:80"},
			},
			nextHandler: createOptionsHanlder(ctrl),
		},
		{
			name:    "options - allow *",
			request: reqOptions,
			config: cors.Config{
				AccessControlAllowOrigins: slice.StringSlice{"*"},
				AccessContolAllowHeaders:  slice.StringSlice{"token"},
			},
			nextHandler:     createOptionsHanlder(ctrl),
			expectedOrigin:  "http://example.com",
			expectedMethods: "GET",
			expectedHeaders: "token",
			expectedMaxAge:  "86400",
		},
		{
			name:    "post - allow",
			request: reqPost,
			config: cors.Config{
				AccessControlAllowOrigins: slice.StringSlice{"http://example.com"},
				AccessContolAllowHeaders:  slice.StringSlice{"token", "custom"},
			},
			nextHandler:     createHandler(ctrl),
			expectedOrigin:  "http://example.com",
			expectedMethods: "GET",
			expectedHeaders: "token,custom",
			expectedMaxAge:  "86400",
		},
		{
			name:        "post - forbidden",
			request:     reqPost,
			config:      cors.Config{AccessControlAllowOrigins: slice.StringSlice{"http://example.com:80"}},
			nextHandler: createHandler(ctrl),
		},
		{
			name:    "post - allow *",
			request: reqPost,
			config: cors.Config{
				AccessControlAllowOrigins: slice.StringSlice{"*"},
				AccessContolAllowHeaders:  slice.StringSlice{"Content-type"},
			},
			nextHandler:     createHandler(ctrl),
			expectedOrigin:  "http://example.com",
			expectedMethods: "GET",
			expectedHeaders: "Content-type",
			expectedMaxAge:  "86400",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			mw := cors.New(mux.NewRouter(), testCase.config)
			handler := mw.Middleware(testCase.nextHandler)

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, testCase.request)
			resp := w.Result()
			if err := resp.Body.Close(); err != nil {
				t.Fatalf("failed to close body: %s", err)
			}

			origin := resp.Header.Get(cors.HeaderACAO)
			if testCase.expectedOrigin != origin {
				t.Errorf("expected origin to be '%s' but got '%s'", testCase.expectedOrigin, origin)
			}

			methods := resp.Header.Get(cors.HeaderACAM)
			if testCase.expectedMethods != methods {
				t.Errorf("expected methods to be '%s' but got '%s'", testCase.expectedMethods, methods)
			}

			headers := resp.Header.Get(cors.HeaderACAH)
			if testCase.expectedHeaders != headers {
				t.Errorf("expected headers to be '%s' but got '%s'", testCase.expectedHeaders, headers)
			}

			maxAge := resp.Header.Get(cors.HeaderACMA)
			if testCase.expectedMaxAge != maxAge {
				t.Errorf("expected max age to be '%s' but got '%s'", testCase.expectedMaxAge, maxAge)
			}

		})
	}
}
