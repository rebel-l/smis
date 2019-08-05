package smis

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/rebel-l/go-utils/slice"

	"github.com/gorilla/mux"
)

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
			path:   "/health",
			method: http.MethodDelete,
		},
		{
			name:   "method get",
			path:   "/health",
			method: http.MethodGet,
		},
		{
			name:   "method head",
			path:   "/health",
			method: http.MethodHead,
		},
		{
			name:   "method options",
			path:   "/health",
			method: http.MethodOptions,
		},
		{
			name:   "method patch",
			path:   "/health",
			method: http.MethodPatch,
		},
		{
			name:   "method post",
			path:   "/health",
			method: http.MethodPost,
		},
		{
			name:   "method put",
			path:   "/health",
			method: http.MethodPut,
		},
		{
			name:   "method trace",
			path:   "/health",
			method: http.MethodTrace,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			service := Service{Router: mux.NewRouter()}
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
			if registeredEndpoints.KeyNotExists(testcase.path) {
				t.Errorf("path '%s' is not added to registred endpoints", testcase.path)
			}

			if registeredEndpoints.GetValuesForKey(testcase.path).IsNotIn(testcase.method) {
				t.Errorf("method '%s' is not set added to registered endpoints", testcase.method)
			}
		})
	}
}

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
