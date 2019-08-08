package smis

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

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

/*
func TestService_RegisterEndpoint(t *testing.T) {
	testcases := []struct {
		name   string
		path   string
		method string
		err    error
	}{
		{
			name: "not allowed method",
			err:  fmt.Errorf("method  is not allowed"),
		},
		{
			name:   "method connect",
			path:   "/health",
			method: http.MethodConnect,
		},
		{
			name:   "method delete",
			path:   "/ping",
			method: http.MethodDelete,
		},
		{
			name:   "method get",
			path:   "/route",
			method: http.MethodGet,
		},
		{
			name:   "method head",
			path:   "/route/sub",
			method: http.MethodHead,
		},
		{
			name:   "method options",
			path:   "/route/:param",
			method: http.MethodOptions,
		},
		{
			name:   "method patch",
			path:   "/something",
			method: http.MethodPatch,
		},
		{
			name:   "method post",
			path:   "/else",
			method: http.MethodPost,
		},
		{
			name:   "method put",
			path:   "/to",
			method: http.MethodPut,
		},
		{
			name:   "method trace",
			path:   "/test",
			method: http.MethodTrace,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			service := &Service{Router: mux.NewRouter()}
			route, err := service.RegisterEndpoint(
				testcase.path,
				testcase.method,
				func(_ http.ResponseWriter, _ *http.Request) {},
			)

			if err != nil && testcase.err != nil {
				if err.Error() != testcase.err.Error() {
					t.Errorf("Expected the error '%s' but got '%s'", testcase.err, err)
				}
				return
			}

			// check methods
			var methods slice.StringSlice
			methods, err = route.GetMethods()
			if err != nil {
				t.Fatal("cannot retrieve from created route")
			}
			if methods.IsNotIn(testcase.method) {
				t.Errorf("method '%s' is not set on route", testcase.method)
			}

			// check registered endpoints
			//path := extractPath(testcase.path)
			//if service.registeredEndpoints.KeyNotExists(path) {
			//	t.Errorf("path '%s' is not added to registred endpoints", path)
			//}
			//
			//if service.registeredEndpoints.GetValuesForKey(path).IsNotIn(testcase.method) {
			//	t.Errorf("method '%s' is not set added to registered endpoints: %v", testcase.method, service.registeredEndpoints)
			//}
		})
	}
}*/

func TestService_ServeHTTP(t *testing.T) {
	// TODO: mock logger
	service, err := NewService(&http.Server{}, logrus.New())
	if err != nil {
		t.Fatalf("failed to create service: %s", err)
	}

	// setup endpoints
	endpoints := []struct {
		path    string
		method  string
		handler func(writer http.ResponseWriter, request *http.Request)
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

/*
func Test_NotFound(t *testing.T) {
	service, err := NewService(&http.Server{}, logrus.New())
	if err != nil {
		t.Fatalf("failed to create service: %s", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/something", nil)
	service.Router.ServeHTTP(w, req)
	t.Log(w.Result())
}

func Test_NotAllowed(t *testing.T) {
	service, err := NewService(&http.Server{}, logrus.New())
	if err != nil {
		t.Fatalf("failed to create service: %s", err)
	}

	_, err = service.RegisterEndpoint("/path", http.MethodGet, func(writer http.ResponseWriter, request *http.Request) {
		_, err := io.WriteString(writer, "We should not get this response")
		if err != nil {
			t.Fatalf("failed to writs response: %s", err)
		}
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/path", nil)
	service.Router.ServeHTTP(w, req)
	t.Log(w.Result())
}
*/
/*
func TestExtractPath(t *testing.T) {
	tests := []struct {
		name  string
		given string
		want  string
	}{
		{
			name:  "no slashes",
			given: "ping",
			want:  "",
		},
		{
			name:  "slash only",
			given: "/",
			want:  "",
		},
		{
			name:  "one slash at beginning",
			given: "/ping",
			want:  "/ping",
		},
		{
			name:  "two slashes",
			given: "/ping/something",
			want:  "/ping/something",
		},
		{
			name:  "two slashes with parameter",
			given: "/ping/:id",
			want:  "/ping",
		},
		{
			name:  "two slashes with parameter 2",
			given: "/route/:param",
			want:  "/route",
		},
		{
			name:  "two slashes with ending slash",
			given: "/Pong/",
			want:  "/Pong",
		},
		{
			name:  "mixed cases",
			given: "/pingPong",
			want:  "/pingPong",
		},
		{
			name:  "with dash",
			given: "/ping-pong",
			want:  "/ping-pong",
		},
		{
			name:  "with underscore",
			given: "/ping_pong",
			want:  "/ping_pong",
		},
		{
			name:  "with digits",
			given: "/ping123",
			want:  "/ping123",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := extractPath(test.given)
			if test.want != actual {
				t.Errorf("expected %s but got %s", test.want, actual)
			}
		})
	}
}
*/
