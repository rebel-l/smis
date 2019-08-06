package smis

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/rebel-l/go-utils/slice"

	"github.com/gorilla/mux"
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
}

func TestService_ServeHTTP(t *testing.T) {
	testcases := []struct {
		name    string
		path    string
		method  string
		request *http.Request
	}{
		{
			name:    "root path / get",
			path:    "/",
			method:  http.MethodGet,
			request: httptest.NewRequest(http.MethodPut, "/", nil),
		},
		{
			name:    "path with param",
			path:    "/route/:param",
			method:  http.MethodPut,
			request: httptest.NewRequest(http.MethodGet, "/route/12358", nil),
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			service := &Service{Router: mux.NewRouter(), Log: logrus.New()}
			route, err := service.RegisterEndpoint(testcase.path, testcase.method, func(writer http.ResponseWriter, request *http.Request) {
				_, err := io.WriteString(writer, "We should not get this response")
				if err != nil {
					t.Fatalf("failed to writs response: %s", err)
				}
			})

			if err != nil {
				t.Fatalf("failed to register endpoint: %s", err)
			}
			t.Log(route)
			//t.Log(service.registeredEndpoints)

			w := httptest.NewRecorder()
			service.Router.ServeHTTP(w, testcase.request)
			resp := w.Result()
			t.Log(resp)
			t.Log(route.GetPathTemplate())
			//service.Router.NotFoundHandler
		})
	}
}

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
