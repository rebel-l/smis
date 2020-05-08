//go:generate mockgen -destination ./../tests/mocks/http_mock/handler.go -package http_mock net/http Handler

package cors_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/gorilla/mux"

	"github.com/rebel-l/go-utils/slice"
	"github.com/rebel-l/smis/middleware/cors"
	"github.com/rebel-l/smis/tests/mocks/http_mock"
)

func createMockHandler(ctrl *gomock.Controller, times int) *http_mock.MockHandler {
	handler := http_mock.NewMockHandler(ctrl)
	handler.EXPECT().ServeHTTP(gomock.Any(), gomock.Any()).Times(times)

	return handler
}

func TestNew(t *testing.T) { // nolint: funlen
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	router := mux.NewRouter()
	router.HandleFunc("/", func(_ http.ResponseWriter, _ *http.Request) {}).
		Methods(http.MethodPost, http.MethodGet)

	reqOptions := httptest.NewRequest(http.MethodOptions, "/", nil)
	reqOptions.Header.Set(cors.HeaderOrigin, "http://example.com")
	reqOptions.Header.Set(cors.HeaderACRM, http.MethodPost)

	reqPost := httptest.NewRequest(http.MethodPost, "/", nil)
	reqPost.Header.Set(cors.HeaderOrigin, "http://example.com")

	testCases := []struct {
		name            string
		request         *http.Request
		config          cors.Config
		nextHandler     http.Handler
		expectedCode    int
		expectedOrigin  string
		expectedMethods string
		expectedHeaders string
		expectedMaxAge  string
		expectedBody    string
	}{
		{
			name:    "options - allow",
			request: reqOptions,
			config: cors.Config{
				AccessControlAllowOrigins: slice.StringSlice{"http://example.com"},
				AccessControlAllowHeaders: slice.StringSlice{"*"},
			},
			nextHandler:     createMockHandler(ctrl, 0),
			expectedCode:    http.StatusNoContent,
			expectedOrigin:  "http://example.com",
			expectedMethods: "GET,POST",
			expectedHeaders: "*",
			expectedMaxAge:  "86400",
		},
		{
			name:    "options - forbidden",
			request: reqOptions,
			config: cors.Config{
				AccessControlAllowOrigins: slice.StringSlice{"http://example.com:80"},
			},
			nextHandler:  createMockHandler(ctrl, 0),
			expectedCode: http.StatusForbidden,
			expectedBody: "access from origin forbidden",
		},
		{
			name:    "options - allow *",
			request: reqOptions,
			config: cors.Config{
				AccessControlAllowOrigins: slice.StringSlice{"*"},
				AccessControlAllowHeaders: slice.StringSlice{"token"},
				AccessControlMaxAge:       10,
			},
			nextHandler:     createMockHandler(ctrl, 0),
			expectedCode:    http.StatusNoContent,
			expectedOrigin:  "http://example.com",
			expectedMethods: "GET,POST",
			expectedHeaders: "token",
			expectedMaxAge:  "10",
		},
		{
			name:    "post - allow",
			request: reqPost,
			config: cors.Config{
				AccessControlAllowOrigins: slice.StringSlice{"http://example.com"},
				AccessControlAllowHeaders: slice.StringSlice{"token", "custom"},
			},
			nextHandler:     createMockHandler(ctrl, 1),
			expectedCode:    http.StatusOK,
			expectedOrigin:  "http://example.com",
			expectedMethods: "GET,POST",
			expectedHeaders: "token,custom",
			expectedMaxAge:  "86400",
		},
		{
			name:         "post - forbidden",
			request:      reqPost,
			config:       cors.Config{AccessControlAllowOrigins: slice.StringSlice{"http://example.com:80"}},
			nextHandler:  createMockHandler(ctrl, 0),
			expectedCode: http.StatusForbidden,
			expectedBody: "access from origin forbidden",
		},
		{
			name:    "post - allow *",
			request: reqPost,
			config: cors.Config{
				AccessControlAllowOrigins: slice.StringSlice{"*"},
				AccessControlAllowHeaders: slice.StringSlice{"Content-type"},
			},
			nextHandler:     createMockHandler(ctrl, 1),
			expectedCode:    http.StatusOK,
			expectedOrigin:  "http://example.com",
			expectedMethods: "GET,POST",
			expectedHeaders: "Content-type",
			expectedMaxAge:  "86400",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			mw := cors.New(router, testCase.config)
			handler := mw.Middleware(testCase.nextHandler)

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, testCase.request)
			resp := w.Result()

			// Assert Code
			if testCase.expectedCode != resp.StatusCode {
				t.Errorf("expected code %d but got %d", testCase.expectedCode, resp.StatusCode)
			}

			// Assert Body
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read body: %s", err)
			}

			if testCase.expectedBody != string(body) {
				t.Errorf("expected body to be '%s' but got '%s'", testCase.expectedBody, string(body))
			}

			if err := resp.Body.Close(); err != nil {
				t.Fatalf("failed to close body: %s", err)
			}

			// Assert Header
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
