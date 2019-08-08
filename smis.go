//go:generate mockgen -destination mocks/logrus_mock/fieldlogger.go -package logrus_mock github.com/sirupsen/logrus FieldLogger

// Package smis provides the basic functions to run a service
package smis

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/rebel-l/go-utils/slice"

	"github.com/sirupsen/logrus"
)

const (
//MiddlewareChainPublic     = "public"
//MiddlewareChainRestricted = "restricted"
)

// Server is an interface to describe how to serve endpoints
type Server interface {
	ListenAndServe() error
}

// Service represents the fields necessary for a service
type Service struct {
	Log    logrus.FieldLogger
	Router *mux.Router
	//SubRouters          map[string]*mux.Router
	Server Server
	//MiddlewareChain     map[string][]mux.MiddlewareFunc
	//registeredEndpoints mapof.StringSliceMap
}

// NewService returns an initialized service struct
func NewService(server Server, log logrus.FieldLogger) (*Service, error) {
	if server == nil {
		return nil, fmt.Errorf("server should not be nil")
	}

	if log == nil {
		return nil, fmt.Errorf("log should not be nil")
	}

	service := &Service{
		Log:    log,
		Router: mux.NewRouter(),
		Server: server,
	}
	service.Router.NotFoundHandler = http.HandlerFunc(service.notFoundHandler)
	service.Router.MethodNotAllowedHandler = http.HandlerFunc(service.methodNotAllowedHandler)

	return service, nil
}

/*
func (s *Service) AddMiddleware(chain string, middleware mux.MiddlewareFunc) {
	s.MiddlewareChain[chain] = append(s.MiddlewareChain[chain], middleware)
}

func (s *Service) AddMiddlewareForPublicChain(middleware mux.MiddlewareFunc) {
	s.AddMiddleware(MiddlewareChainPublic, middleware)
}

func (s *Service) AddMiddlewareForRestrictedChain(middleware mux.MiddlewareFunc) {
	s.AddMiddleware(MiddlewareChainRestricted, middleware)
}
*/

// RegisterEndpoint registers a handler at the router for the given method and path.
// In case the method is not known an error is return, otherwise a *Route.
func (s *Service) RegisterEndpoint(
	path, method string, f func(http.ResponseWriter, *http.Request)) (*mux.Route, error) {

	methods := getAllowedHTTPMethods()
	if methods.IsNotIn(method) {
		return nil, fmt.Errorf("method %s is not allowed", method)
	}

	return s.Router.HandleFunc(path, f).Methods(method), nil
}

// RegisterFileServer registers a file server to provide static files
func (s *Service) RegisterFileServer(path, method, filepath string) (*mux.Route, error) {
	return s.Router.
			PathPrefix(path).
			Handler(http.StripPrefix(path, http.FileServer(http.Dir(filepath)))).
			Methods(method),
		nil
}

// ListenAndServe registers the catch all route and starts the server
func (s *Service) ListenAndServe() error {
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

/*
// ServeHTTP is the catch all handler
func (s *Service) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	var err error
	path := extractPath(request.RequestURI)

	if s.registeredEndpoints.KeyExists(path) {
		//s.Log.Warnf("method not allowed: %s | %s", request.Method, request.RequestURI)
		//writer.Header().Add("Allow", strings.Join(s.registeredEndpoints.GetValuesForKey(path), ","))
		//writer.WriteHeader(405)
		//_, err = writer.Write([]byte("method not allowed, please check response headers for allowed methods"))
	} else {
		//s.Log.Warnf("endpoint not implemented: %s | %s", request.Method, request.RequestURI)
		//writer.WriteHeader(404)
		//_, err = writer.Write([]byte("endpoint not implemented"))
	}

	if err != nil {
		s.Log.Errorf("catchAll failed: %s", err)
	}
}
*/

func (s *Service) notFoundHandler(writer http.ResponseWriter, request *http.Request) {
	s.Log.Warnf("endpoint not implemented: %s | %s", request.Method, request.RequestURI)
	writer.WriteHeader(404)
	_, err := writer.Write([]byte("endpoint not implemented"))
	if err != nil {
		s.Log.Errorf("notFoundHandler failed to send response: %s", err)
	}
}

func (s *Service) methodNotAllowedHandler(writer http.ResponseWriter, request *http.Request) {
	s.Log.Warnf("method not allowed: %s | %s", request.Method, request.RequestURI)

	methods := make([]string, 0)
	for _, m := range getAllowedHTTPMethods() {
		if request.Method == m {
			continue
		}

		simReq := &http.Request{Method: m, URL: request.URL, RequestURI: request.RequestURI}
		match := &mux.RouteMatch{}
		if !s.Router.Match(simReq, match) || match.MatchErr != nil {
			continue
		}

		methods = append(methods, m)
	}
	writer.Header().Add("Allow", strings.Join(methods, ","))
	writer.WriteHeader(405)
	_, err := writer.Write([]byte("method not allowed, please check response headers for allowed methods"))
	if err != nil {
		s.Log.Errorf("notAllowedHandler failed to send response: %s", err)
	}
}

/*
func extractPath(path string) string {
	r := regexp.MustCompile(`\/[\w-]+`)
	return strings.Join(r.FindAllString(path, -1), "")
}
*/

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

// TODO: deal with OPTIONS request (CORS/ACAO/ACAM/ACAH) ==> CORS Middleware
// TODO: add possibility to configure CORS
// TODO: introduce different middleware chains ==> predefined: public / restricted
