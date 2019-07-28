// Package smis provides the basic functions to run a service
package smis

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gorilla/mux"

	"github.com/rebel-l/go-utils/mapof"
	"github.com/rebel-l/go-utils/slice"
	"github.com/sirupsen/logrus"
)

var registeredEndpoints mapof.StringSliceMap

// Service represents the fields necessary for a service
type Service struct {
	Log    logrus.FieldLogger
	Router *mux.Router
	Server *http.Server
}

// RegisterEndpoint registers a handler at the router for the given method and path.
// In case the method is not known an error is return, otherwise a *Route.
func (s Service) RegisterEndpoint(path, method string, f func(http.ResponseWriter, *http.Request)) (*mux.Route, error) {
	methods := getAllowedHTTPMethods()
	if methods.IsNotIn(method) {
		return nil, fmt.Errorf("method %s is not allowed", method)
	}

	if registeredEndpoints == nil {
		registeredEndpoints = make(mapof.StringSliceMap)
	}
	registeredEndpoints.AddUniqueValue(extractPath(path), method)

	return s.Router.HandleFunc(path, f).Methods(method), nil
}

// RegisterFileServer registers a file server to provide static files
func (s Service) RegisterFileServer(path, method, filepath string) (*mux.Route, error) {
	return s.Router.PathPrefix(path).Handler(http.StripPrefix(path, http.FileServer(http.Dir(filepath)))).Methods(method), nil
}

// ListenAndServe registers the catch all route and starts the server
func (s Service) ListenAndServe() error {
	s.Router.PathPrefix("/").Handler(&s)
	err := s.Router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, err := route.GetPathTemplate()
		if err != nil {
			return err
		}
		s.Log.Infof("Available Route: %s", pathTemplate)
		return nil
	})

	if err != nil {
		return err
	}
	return s.Server.ListenAndServe()
}

// ServeHTTP is the catch all handler
func (s Service) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	var err error
	path := extractPath(request.RequestURI)

	if registeredEndpoints.KeyExists(path) {
		s.Log.Warnf("method not allowed: %s | %s", request.Method, request.RequestURI)
		writer.Header().Add("Allow", strings.Join(registeredEndpoints.GetValuesForKey(path), ","))
		writer.WriteHeader(405)
		_, err = writer.Write([]byte("method not allowed, please check response headers for allowed methods"))
	} else {
		s.Log.Warnf("endpoint not implemented: %s | %s", request.Method, request.RequestURI)
		writer.WriteHeader(404)
		_, err = writer.Write([]byte("endpoint not implemented"))
	}

	if err != nil {
		s.Log.Errorf("catchAll failed: %s", err)
	}
}

func extractPath(path string) string {
	r := regexp.MustCompile(`\/[\w-]+`)
	return strings.Join(r.FindAllString(path, -1), "")
}

func getAllowedHTTPMethods() slice.StringSlice {
	return slice.StringSlice{
		http.MethodConnect,
		http.MethodDelete,
		http.MethodGet,
		http.MethodHead,
		http.MethodOptions,
		http.MethodPatch,
		http.MethodPost,
		http.MethodPut,
		http.MethodTrace,
	}
}
