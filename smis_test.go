//go:generate mockgen -destination ./tests/mocks/http_mock/responseWriter.go -package http_mock net/http ResponseWriter
//go:generate mockgen -destination ./tests/mocks/logrus_mock/fieldlogger.go -package logrus_mock github.com/sirupsen/logrus FieldLogger
//go:generate mockgen -destination ./tests/mocks/smis_mock/smis.go -package smis_mock github.com/rebel-l/smis Server

package smis

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rebel-l/smis/middleware/requestid"

	"github.com/golang/mock/gomock"

	"github.com/gorilla/mux"

	"github.com/rebel-l/go-utils/slice"
	"github.com/rebel-l/smis/middleware/cors"
	"github.com/rebel-l/smis/tests/mocks/http_mock"
	"github.com/rebel-l/smis/tests/mocks/logrus_mock"
	"github.com/rebel-l/smis/tests/mocks/smis_mock"

	"github.com/sirupsen/logrus"
)

func TestNewService(t *testing.T) {
	testcases := []struct {
		name   string
		server Server
		log    logrus.FieldLogger
		err    error
	}{
		{
			name: "server and log nil",
			err:  fmt.Errorf("server should not be nil"),
		},
		{
			name: "server nil",
			log:  logrus.New(),
			err:  fmt.Errorf("server should not be nil"),
		},
		{
			name:   "log nil",
			server: &http.Server{},
			err:    fmt.Errorf("log should not be nil"),
		},
		{
			name:   "server and log given",
			log:    logrus.New(),
			server: &http.Server{},
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			_, err := NewService(testcase.server, testcase.log)

			if testcase.err == nil && err != nil {
				t.Errorf("expected no error but got '%s'", err.Error())
				return
			}

			if testcase.err != nil && err == nil {
				t.Errorf("expected '%s' but got no error", testcase.err.Error())
				return
			}

			if testcase.err != nil && err != nil && testcase.err.Error() != err.Error() {
				t.Errorf("expected error message '%s' but got '%s'", testcase.err.Error(), err.Error())
			}
		})
	}
}

func TestService_RegisterEndpoint_Error(t *testing.T) {
	testcases := []struct {
		name   string
		path   string
		method string
	}{
		{
			name: "no path / method",
		},
		{
			name:   "method only",
			method: "batman",
		},
		{
			name:   "method and path",
			method: "spiderman",
			path:   "/something",
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			service := &Service{Router: mux.NewRouter()}
			_, err := service.RegisterEndpoint(
				testcase.path,
				testcase.method,
				func(_ http.ResponseWriter, _ *http.Request) {},
			)

			msg := fmt.Errorf("method %s is not allowed", testcase.method)
			if err == nil {
				t.Error("expected an error to be thrown but got nil")
			} else if msg.Error() != err.Error() {
				t.Errorf("expected error '%s' but got '%s'", msg, err)
			}
		})
	}
}

func TestService_ServeHTTP(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := logrus_mock.NewMockFieldLogger(ctrl)
	m.EXPECT().Warnf(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	service, err := NewService(&http.Server{}, m)
	if err != nil {
		t.Fatalf("failed to create service: %s", err)
	}

	// setup endpoints
	endpoints := []struct {
		path    string
		method  string
		handler http.HandlerFunc
	}{
		{
			path:   "/health",
			method: http.MethodGet,
			handler: func(writer http.ResponseWriter, _ *http.Request) {
				_, err := io.WriteString(writer, "health endpoint")
				if err != nil {
					t.Fatalf("failed to write response: %s", err)
				}
			},
		},
		{
			path:   "/user/{id}",
			method: http.MethodPut,
			handler: func(writer http.ResponseWriter, _ *http.Request) {
				_, err := io.WriteString(writer, "user endpoint - PUT")
				if err != nil {
					t.Fatalf("failed to write response: %s", err)
				}
			},
		},
		{
			path:   "/user/{id}",
			method: http.MethodPost,
			handler: func(writer http.ResponseWriter, _ *http.Request) {
				_, err := io.WriteString(writer, "user endpoint - POST")
				if err != nil {
					t.Fatalf("failed to write response: %s", err)
				}
			},
		},
	}

	for _, e := range endpoints {
		_, err := service.RegisterEndpoint(e.path, e.method, e.handler)
		if err != nil {
			t.Fatalf("failed to register endpoint: %s", err)
		}
	}

	_, err = service.RegisterFileServer("/static", http.MethodGet, "./tests/data/staticfiles")
	if err != nil {
		t.Fatalf("failed to register file serer: %s", err)
	}

	// define test cases
	headerSkip := make(map[string]string)

	headerPlain := make(map[string]string)
	headerPlain["Content-type"] = "text/plain; charset=utf-8"

	headerNotAllowed := make(map[string]string)
	headerNotAllowed["Allow"] = "GET"

	headerNotAllowed2 := make(map[string]string)
	headerNotAllowed2["Allow"] = "POST,PUT"

	testcases := []struct {
		name           string
		request        *http.Request
		expectedStatus string
		expectedHeader map[string]string
		expectedBody   string
	}{
		{
			name:           "root path / get",
			request:        httptest.NewRequest(http.MethodGet, "/", nil),
			expectedStatus: "404 Not Found",
			expectedHeader: headerSkip,
			expectedBody:   "endpoint not implemented",
		},
		{
			name:           "health endpoint - GET",
			request:        httptest.NewRequest(http.MethodGet, "/health", nil),
			expectedStatus: "200 OK",
			expectedHeader: headerPlain,
			expectedBody:   "health endpoint",
		},
		{
			name:           "health endpoint - CONNECT",
			request:        httptest.NewRequest(http.MethodConnect, "/health", nil),
			expectedStatus: "405 Method Not Allowed",
			expectedHeader: headerNotAllowed,
			expectedBody:   "method not allowed, please check response headers for allowed methods",
		},
		{
			name:           "health endpoint - DELETE",
			request:        httptest.NewRequest(http.MethodDelete, "/health", nil),
			expectedStatus: "405 Method Not Allowed",
			expectedHeader: headerNotAllowed,
			expectedBody:   "method not allowed, please check response headers for allowed methods",
		},
		{
			name:           "health endpoint - HEAD",
			request:        httptest.NewRequest(http.MethodHead, "/health", nil),
			expectedStatus: "405 Method Not Allowed",
			expectedHeader: headerNotAllowed,
			expectedBody:   "method not allowed, please check response headers for allowed methods",
		},
		{
			name:           "health endpoint - OPTIONS",
			request:        httptest.NewRequest(http.MethodOptions, "/health", nil),
			expectedStatus: "405 Method Not Allowed",
			expectedHeader: headerNotAllowed,
			expectedBody:   "method not allowed, please check response headers for allowed methods",
		},
		{
			name:           "health endpoint - PATCH",
			request:        httptest.NewRequest(http.MethodPatch, "/health", nil),
			expectedStatus: "405 Method Not Allowed",
			expectedHeader: headerNotAllowed,
			expectedBody:   "method not allowed, please check response headers for allowed methods",
		},
		{
			name:           "health endpoint - POST",
			request:        httptest.NewRequest(http.MethodPost, "/health", nil),
			expectedStatus: "405 Method Not Allowed",
			expectedHeader: headerNotAllowed,
			expectedBody:   "method not allowed, please check response headers for allowed methods",
		},
		{
			name:           "health endpoint - PUT",
			request:        httptest.NewRequest(http.MethodPut, "/health", nil),
			expectedStatus: "405 Method Not Allowed",
			expectedHeader: headerNotAllowed,
			expectedBody:   "method not allowed, please check response headers for allowed methods",
		},
		{
			name:           "health endpoint - TRACE",
			request:        httptest.NewRequest(http.MethodTrace, "/health", nil),
			expectedStatus: "405 Method Not Allowed",
			expectedHeader: headerNotAllowed,
			expectedBody:   "method not allowed, please check response headers for allowed methods",
		},
		{
			name:           "user endpoint - POST",
			request:        httptest.NewRequest(http.MethodPost, "/user/1", nil),
			expectedStatus: "200 OK",
			expectedHeader: headerPlain,
			expectedBody:   "user endpoint - POST",
		},
		{
			name:           "user endpoint - PUT",
			request:        httptest.NewRequest(http.MethodPut, "/user/2", nil),
			expectedStatus: "200 OK",
			expectedHeader: headerPlain,
			expectedBody:   "user endpoint - PUT",
		},
		{
			name:           "user endpoint - CONNECT",
			request:        httptest.NewRequest(http.MethodConnect, "/user/3", nil),
			expectedStatus: "405 Method Not Allowed",
			expectedHeader: headerNotAllowed2,
			expectedBody:   "method not allowed, please check response headers for allowed methods",
		},
		{
			name:           "user endpoint - DELETE",
			request:        httptest.NewRequest(http.MethodDelete, "/user/3", nil),
			expectedStatus: "405 Method Not Allowed",
			expectedHeader: headerNotAllowed2,
			expectedBody:   "method not allowed, please check response headers for allowed methods",
		},
		{
			name:           "user endpoint - HEAD",
			request:        httptest.NewRequest(http.MethodHead, "/user/3", nil),
			expectedStatus: "405 Method Not Allowed",
			expectedHeader: headerNotAllowed2,
			expectedBody:   "method not allowed, please check response headers for allowed methods",
		},
		{
			name:           "user endpoint - GET",
			request:        httptest.NewRequest(http.MethodGet, "/user/3", nil),
			expectedStatus: "405 Method Not Allowed",
			expectedHeader: headerNotAllowed2,
			expectedBody:   "method not allowed, please check response headers for allowed methods",
		},
		{
			name:           "user endpoint - HEAF",
			request:        httptest.NewRequest(http.MethodHead, "/user/3", nil),
			expectedStatus: "405 Method Not Allowed",
			expectedHeader: headerNotAllowed2,
			expectedBody:   "method not allowed, please check response headers for allowed methods",
		},
		{
			name:           "user endpoint - OPTIONS",
			request:        httptest.NewRequest(http.MethodOptions, "/user/3", nil),
			expectedStatus: "405 Method Not Allowed",
			expectedHeader: headerNotAllowed2,
			expectedBody:   "method not allowed, please check response headers for allowed methods",
		},
		{
			name:           "user endpoint - PATCH",
			request:        httptest.NewRequest(http.MethodPatch, "/user/3", nil),
			expectedStatus: "405 Method Not Allowed",
			expectedHeader: headerNotAllowed2,
			expectedBody:   "method not allowed, please check response headers for allowed methods",
		},
		{
			name:           "user endpoint - TRACE",
			request:        httptest.NewRequest(http.MethodTrace, "/user/3", nil),
			expectedStatus: "405 Method Not Allowed",
			expectedHeader: headerNotAllowed2,
			expectedBody:   "method not allowed, please check response headers for allowed methods",
		},
		{
			name:           "user endpoint - GET - not implemented",
			request:        httptest.NewRequest(http.MethodDelete, "/user/", nil),
			expectedStatus: "404 Not Found",
			expectedHeader: headerSkip,
			expectedBody:   "endpoint not implemented",
		},
		{
			name:           "static endpoint - file exists",
			request:        httptest.NewRequest(http.MethodGet, "/static/something", nil),
			expectedStatus: "200 OK",
			expectedHeader: headerSkip,
			expectedBody:   "This is just some content.",
		},
		{
			name:           "static endpoint - file exists",
			request:        httptest.NewRequest(http.MethodGet, "/static/somethingelse", nil),
			expectedStatus: "404 Not Found",
			expectedHeader: headerSkip,
			expectedBody:   "404 page not found\n",
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			service.Router.ServeHTTP(w, testcase.request)
			resp := w.Result()

			// check status
			if testcase.expectedStatus != resp.Status {
				t.Errorf("expected status '%s' but got '%s'", testcase.expectedStatus, resp.Status)
			}

			// check header
			for key, expectedHeader := range testcase.expectedHeader {
				header := resp.Header.Get(key)
				if expectedHeader != header {
					t.Errorf("expected haeder for key '%s' to be '%s' but got '%s'", key, expectedHeader, header)
				}
			}

			// check body
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read body: %s", err)
			}
			if err = resp.Body.Close(); err != nil {
				t.Fatalf("failed to close body: %s", err)
			}

			if testcase.expectedBody != string(body) {
				t.Errorf("expected body to be '%s' but got '%s'", testcase.expectedBody, string(body))
			}
		})
	}
}

func TestService_ServeHTTP_WithMiddleware(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := logrus_mock.NewMockFieldLogger(ctrl)

	serviceDefault, err := NewService(&http.Server{}, m)
	if err != nil {
		t.Fatalf("failed to create service: %s", err)
	}

	serviceSubRouters, err := NewService(&http.Server{}, m)
	if err != nil {
		t.Fatalf("failed to create service: %s", err)
	}

	endpoint := func(_ http.ResponseWriter, _ *http.Request) {}

	// default
	middlewareDefault := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			_, err := writer.Write([]byte("default middleware"))
			if err != nil {
				t.Fatalf("failed to send response from default middleware: %s", err)
			}
		})
	}
	serviceDefault.AddMiddlewareForDefaultChain(middlewareDefault)
	_, err = serviceDefault.RegisterEndpoint("/main", http.MethodGet, endpoint)
	if err != nil {
		t.Errorf("failed to register endpoint: %s", err)
	}

	// public
	middlewarePublic := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			_, err := writer.Write([]byte("public middleware"))
			if err != nil {
				t.Fatalf("failed to send response from public middleware: %s", err)
			}
		})
	}
	serviceSubRouters.AddMiddlewareForPublicChain(middlewarePublic)
	_, err = serviceSubRouters.RegisterEndpointToPublicChain("/weather", http.MethodGet, endpoint)
	if err != nil {
		t.Errorf("failed to register endpoint: %s", err)
	}

	// restricted
	middlewareRestricted := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			_, err := writer.Write([]byte("restricted middleware"))
			if err != nil {
				t.Fatalf("failed to send response from restricted middleware: %s", err)
			}
		})
	}
	serviceSubRouters.AddMiddlewareForRestrictedChain(middlewareRestricted)
	_, err = serviceSubRouters.RegisterEndpointToRestictedChain("/admin", http.MethodGet, endpoint)
	if err != nil {
		t.Errorf("failed to register endpoint: %s", err)
	}

	testcases := []struct {
		name         string
		request      *http.Request
		expectedBody string
		service      *Service
	}{
		{
			name:         "default",
			request:      httptest.NewRequest(http.MethodGet, "/main", nil),
			expectedBody: "default middleware",
			service:      serviceDefault,
		},
		{
			name:         "public",
			request:      httptest.NewRequest(http.MethodGet, "/public/weather", nil),
			expectedBody: "public middleware",
			service:      serviceSubRouters,
		},
		{
			name:         "restricted",
			request:      httptest.NewRequest(http.MethodGet, "/restricted/admin", nil),
			expectedBody: "restricted middleware",
			service:      serviceSubRouters,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			testcase.service.Router.ServeHTTP(w, testcase.request)
			resp := w.Result()

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read body: %s", err)
			}
			if err = resp.Body.Close(); err != nil {
				t.Fatalf("failed to close body: %s", err)
			}

			if testcase.expectedBody != string(body) {
				t.Errorf("expected body to be '%s' but got '%s'", testcase.expectedBody, string(body))
			}
		})
	}
}

func TestService_ListenAndServe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// success
	serverMockSuccess := smis_mock.NewMockServer(ctrl)
	serverMockSuccess.EXPECT().ListenAndServe().Times(1)

	logMockSuccess := logrus_mock.NewMockFieldLogger(ctrl)
	logMockSuccess.EXPECT().Infof(gomock.Eq("Available Route: %s"), gomock.Eq("/ping")).Times(1)

	serviceSuccess, err := NewService(serverMockSuccess, logMockSuccess)
	if err != nil {
		t.Fatalf("failed to create server: %s", err)
	}

	routeSuccess := serviceSuccess.Router.NewRoute()
	routeSuccess.Path("/ping")

	// error
	serverMockError := smis_mock.NewMockServer(ctrl)
	serverMockError.EXPECT().ListenAndServe().Times(0)

	logMockError := logrus_mock.NewMockFieldLogger(ctrl)
	logMockError.EXPECT().Infof(gomock.Any(), gomock.Any()).Times(0)

	serviceError, err := NewService(serverMockError, logMockError)
	if err != nil {
		t.Fatalf("failed to create server: %s", err)
	}

	routeError := serviceError.Router.NewRoute()
	routeError.Name("health")
	routeError.Path("/health")
	routeError.Name("health new") // this causes an error

	// tests
	testcases := []struct {
		name    string
		service *Service
		err     error
	}{
		{
			name:    "success",
			service: serviceSuccess,
		},
		{
			name:    "error",
			service: serviceError,
			err:     fmt.Errorf(`mux: route already has name "health", can't set "health new"`),
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			err := testcase.service.ListenAndServe()
			if testcase.err == nil && err != nil {
				t.Errorf("ListenAndServe should NOT throw an error, but got: %s", err)
			}

			if testcase.err != nil && (err == nil || testcase.err.Error() != err.Error()) {
				t.Errorf("ListenAndServe should throw an error '%s', but got '%v'", testcase.err, err)
			}
		})
	}
}

func Test_NotFound_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	errMsg := fmt.Errorf("something happened")

	logMock := logrus_mock.NewMockFieldLogger(ctrl)
	logMock.EXPECT().Warnf(gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
	logMock.EXPECT().
		Errorf(gomock.Eq("notFoundHandler failed to send response: %s"), gomock.Eq(errMsg)).
		Times(1)

	service, err := NewService(&http.Server{}, logMock)
	if err != nil {
		t.Fatalf("failed to create service: %s", err)
	}

	writerMock := http_mock.NewMockResponseWriter(ctrl)
	writerMock.EXPECT().WriteHeader(404).Times(1)
	writerMock.EXPECT().Write(gomock.Any()).Return(0, errMsg)
	req := httptest.NewRequest(http.MethodPut, "/something", nil)
	service.notFoundHandler(writerMock, req)
}

func Test_NotAllowed_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	errMsg := fmt.Errorf("response writer closed")

	logMock := logrus_mock.NewMockFieldLogger(ctrl)
	logMock.EXPECT().Warnf(gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
	logMock.EXPECT().
		Errorf(gomock.Eq("notAllowedHandler failed to send response: %s"), gomock.Eq(errMsg)).
		Times(1)

	service, err := NewService(&http.Server{}, logMock)
	if err != nil {
		t.Fatalf("failed to create service: %s", err)
	}

	writerMock := http_mock.NewMockResponseWriter(ctrl)
	writerMock.EXPECT().WriteHeader(405).Times(1)
	writerMock.EXPECT().Header().Times(1).Return(http.Header{})
	writerMock.EXPECT().Write(gomock.Any()).Return(0, errMsg)
	req := httptest.NewRequest(http.MethodPut, "/notAllowed", nil)
	service.methodNotAllowedHandler(writerMock, req)
}

func TestService_WithDefaultMiddleware(t *testing.T) {
	expectedOrigin := "http://example.com"
	expectedMethods := "POST,OPTIONS"
	expectedHeaders := ""
	expectedMaxAge := "86400"
	expectedBody := "created"

	cfg := cors.Config{
		AccessControlAllowOrigins: slice.StringSlice{expectedOrigin},
	}

	req := httptest.NewRequest(http.MethodPost, "/new", nil)
	req.Header.Set(cors.HeaderOrigin, expectedOrigin)

	service, err := NewService(&http.Server{}, logrus.New())
	if err != nil {
		t.Errorf("expect no error but got: %s", err)
	}
	_, err = service.WithDefaultMiddleware(cfg).
		RegisterEndpoint("/new", http.MethodPost, func(writer http.ResponseWriter, _ *http.Request) {

			if _, err = writer.Write([]byte(expectedBody)); err != nil {
				t.Fatalf("failed to send response: %s", err)
			}
		})
	if err != nil {
		t.Errorf("failed to register endpoint: %s", err)
	}
	w := httptest.NewRecorder()
	service.Router.ServeHTTP(w, req)
	resp := w.Result()

	// check header
	gotRequestID := w.Header().Get(requestid.HeaderRID)
	if gotRequestID == "" {
		t.Error("request ID in header should not be empty")
	}

	gotOrigin := w.Header().Get(cors.HeaderACAO)
	if expectedOrigin != gotOrigin {
		t.Errorf("expected origin '%s' but got '%s'", expectedOrigin, gotOrigin)
	}

	gotHeaders := w.Header().Get(cors.HeaderACAH)
	if expectedHeaders != gotHeaders {
		t.Errorf("expected header '%s' but got '%s'", expectedHeaders, gotHeaders)
	}

	gotMethods := w.Header().Get(cors.HeaderACAM)
	if expectedMethods != gotMethods {
		t.Errorf("expected methods '%s' but got '%s'", expectedMethods, gotMethods)
	}

	gotMaxAge := w.Header().Get(cors.HeaderACMA)
	if expectedMaxAge != gotMaxAge {
		t.Errorf("expected max age '%s' but got '%s'", expectedMaxAge, gotMaxAge)
	}

	// check body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %s", err)
	}

	if err = resp.Body.Close(); err != nil {
		t.Fatalf("failed to close body: %s", err)
	}

	if expectedBody != string(body) {
		t.Errorf("expectedf body to be '%s' but got '%s'", expectedBody, string(body))
	}
}

func TestService_WithDefaultMiddlewareForPRChain(t *testing.T) {
	origin := "http://www.example.com"
	cfg := cors.Config{
		AccessControlAllowOrigins: slice.StringSlice{origin},
	}

	service, err := NewService(&http.Server{}, logrus.New())
	if err != nil {
		t.Errorf("expect no error but got: %s", err)
	}
	service.WithDefaultMiddlewareForPRChain(cfg)
	_, err = service.RegisterEndpointToPublicChain(
		"/new", http.MethodPost, func(writer http.ResponseWriter, _ *http.Request) {

			if _, err = writer.Write([]byte("public")); err != nil {
				t.Fatalf("failed to send response: %s", err)
			}
		})
	if err != nil {
		t.Errorf("failed to register endpoint: %s", err)
	}

	_, err = service.RegisterEndpointToRestictedChain(
		"/new", http.MethodPost, func(writer http.ResponseWriter, _ *http.Request) {
			if _, err = writer.Write([]byte("restricted")); err != nil {
				t.Fatalf("failed to send response: %s", err)
			}
		})
	if err != nil {
		t.Errorf("failed to register endpoint: %s", err)
	}

	reqPublic := httptest.NewRequest(http.MethodPost, "/public/new", nil)
	reqPublic.Header.Set(cors.HeaderOrigin, origin)

	reqRestricted := httptest.NewRequest(http.MethodPost, "/restricted/new", nil)
	reqRestricted.Header.Set(cors.HeaderOrigin, origin)

	testCases := []struct {
		name               string
		expectedACAOrigin  string
		expectedACAMethods string
		expectedACAHeaders string
		expectedACAMaxAge  string
		expectedBody       string
		request            *http.Request
	}{
		{
			name:               "public",
			request:            reqPublic,
			expectedACAOrigin:  origin,
			expectedACAMethods: "POST,OPTIONS",
			expectedACAHeaders: "",
			expectedACAMaxAge:  "86400",
			expectedBody:       "public",
		},
		{
			name:               "restricted",
			request:            reqRestricted,
			expectedACAOrigin:  origin,
			expectedACAMethods: "POST,OPTIONS",
			expectedACAHeaders: "",
			expectedACAMaxAge:  "86400",
			expectedBody:       "restricted",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			service.Router.ServeHTTP(w, testCase.request)
			resp := w.Result()

			// check header
			gotRequestID := w.Header().Get(requestid.HeaderRID)
			if gotRequestID == "" {
				t.Error("request ID in header should not be empty")
			}

			gotOrigin := w.Header().Get(cors.HeaderACAO)
			if testCase.expectedACAOrigin != gotOrigin {
				t.Errorf("expected origin '%s' but got '%s'", testCase.expectedACAOrigin, gotOrigin)
			}

			gotHeaders := w.Header().Get(cors.HeaderACAH)
			if testCase.expectedACAHeaders != gotHeaders {
				t.Errorf("expected header '%s' but got '%s'", testCase.expectedACAHeaders, gotHeaders)
			}

			gotMethods := w.Header().Get(cors.HeaderACAM)
			if testCase.expectedACAMethods != gotMethods {
				t.Errorf("expected methods '%s' but got '%s'", testCase.expectedACAMethods, gotMethods)
			}

			gotMaxAge := w.Header().Get(cors.HeaderACMA)
			if testCase.expectedACAMaxAge != gotMaxAge {
				t.Errorf("expected max age '%s' but got '%s'", testCase.expectedACAMaxAge, gotMaxAge)
			}

			// check body
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read body: %s", err)
			}

			if err = resp.Body.Close(); err != nil {
				t.Fatalf("failed to close body: %s", err)
			}

			if testCase.expectedBody != string(body) {
				t.Errorf("expectedf body to be '%s' but got '%s'", testCase.expectedBody, string(body))
			}
		})
	}
}
