// Package smis provides the basic functions to run a service
package smis

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/rebel-l/go-utils/slice"
	"github.com/rebel-l/smis/middleware"
	"github.com/rebel-l/smis/middleware/cors"
	"github.com/rebel-l/smis/middleware/requestid"

	"github.com/sirupsen/logrus"
)

const (
	// MiddlewareChainDefault is the identifier for the default middleware chain
	MiddlewareChainDefault = "default"

	// MiddlewareChainPublic is the identifier for the public middleware chain
	MiddlewareChainPublic = "public"

	// MiddlewareChainRestricted is the identifier for the restricted middleware chain
	MiddlewareChainRestricted = "restricted"
)

// Server is an interface to describe how to serve endpoints
type Server interface {
	ListenAndServe() error
}

// Service represents the fields necessary for a service
type Service struct {
	Log        logrus.FieldLogger
	Router     *mux.Router
	Server     Server
	SubRouters map[string]*mux.Router
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

// GetRouterForMiddlewareChain returns the router (sub router) for a given chain.
// If a chain doesn't exist, it creates it.
func (s *Service) GetRouterForMiddlewareChain(chain string) *mux.Router {
	var router *mux.Router
	switch chain {
	case MiddlewareChainDefault:
		router = s.Router
	default:
		if s.SubRouters == nil {
			s.SubRouters = make(map[string]*mux.Router)
		}

		var ok bool
		router, ok = s.SubRouters[chain]
		if !ok {
			router = s.Router.PathPrefix("/" + chain).Subrouter()
			s.SubRouters[chain] = router
		}
	}
	return router
}

// AddMiddleware adds middleware to a specific chain. You can create custom chains with this method. The chain is
// also the path prefix, eg. your chain is "custom" your routes will start with "/custom"
func (s *Service) AddMiddleware(chain string, middleware mux.MiddlewareFunc) {
	router := s.GetRouterForMiddlewareChain(chain)
	router.Use(middleware)
}

// AddMiddlewareForDefaultChain adds middleware to the default chain. NOTE: The default chain is working without
// path prefix and uses the main router.
func (s *Service) AddMiddlewareForDefaultChain(middleware mux.MiddlewareFunc) {
	s.AddMiddleware(MiddlewareChainDefault, middleware)
}

// AddMiddlewareForPublicChain adds middleware to the public chain. NOTE: The path prefix is /public.
func (s *Service) AddMiddlewareForPublicChain(middleware mux.MiddlewareFunc) {
	s.AddMiddleware(MiddlewareChainPublic, middleware)
}

// AddMiddlewareForRestrictedChain adds middleware to the restricted chain. NOTE: The path prefix is /restricted.
func (s *Service) AddMiddlewareForRestrictedChain(middleware mux.MiddlewareFunc) {
	s.AddMiddleware(MiddlewareChainRestricted, middleware)
}

// RegisterEndpoint registers a handler at the router for the given method and path.
// In case the method is not known an error is returned, otherwise a *Route.
func (s *Service) RegisterEndpoint(
	path, method string, f http.HandlerFunc) (*mux.Route, error) {

	return s.RegisterEndpointToChain(MiddlewareChainDefault, path, method, f)
}

// RegisterEndpointToPublicChain registers a handler at the router for the given method and path at the public chain.
// In case the method is not known an error is returned, otherwise a *Route.
func (s *Service) RegisterEndpointToPublicChain(
	path, method string, f http.HandlerFunc) (*mux.Route, error) {

	return s.RegisterEndpointToChain(MiddlewareChainPublic, path, method, f)
}

// RegisterEndpointToRestictedChain registers a handler at the router for the given method and path at the
// restricted chain.
// In case the method is not known an error is returned, otherwise a *Route.
func (s *Service) RegisterEndpointToRestictedChain(
	path, method string, f http.HandlerFunc) (*mux.Route, error) {

	return s.RegisterEndpointToChain(MiddlewareChainRestricted, path, method, f)
}

// RegisterEndpointToChain registers a handler at the router for the given method and path at any chain. You can use
// your custom chains with this method.
// In case the method is not known an error is returned, otherwise a *Route.
func (s *Service) RegisterEndpointToChain(
	chain, path, method string, f http.HandlerFunc) (*mux.Route, error) {

	methods := getAllowedHTTPMethods()
	if methods.IsNotIn(method) {
		return nil, fmt.Errorf("method %s is not allowed", method)
	}

	router := s.GetRouterForMiddlewareChain(chain)
	return router.HandleFunc(path, f).Methods(method), nil
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
	err := s.Router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
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

// WithDefaultMiddleware initializes the recommended middleware for the default middleware chain.
func (s *Service) WithDefaultMiddleware(config cors.Config) *Service {
	mw := s.GetDefaultMiddleware(config)

	//nolint:gosec
	_ = mw.Walk(func(middleware mux.MiddlewareFunc) error {
		s.AddMiddlewareForDefaultChain(middleware)
		return nil
	})
	return s
}

// WithDefaultMiddlewareForPRChain initializes the recommended middleware for the public & restricted middleware chain.
func (s *Service) WithDefaultMiddlewareForPRChain(config cors.Config) *Service {
	mw := s.GetDefaultMiddleware(config)

	//nolint:gosec
	_ = mw.Walk(func(middleware mux.MiddlewareFunc) error {
		s.AddMiddlewareForPublicChain(middleware)
		s.AddMiddlewareForRestrictedChain(middleware)
		return nil
	})

	return s
}

// GetDefaultMiddleware returns the default middleware every chain should have.
func (s *Service) GetDefaultMiddleware(config cors.Config) middleware.Slice {
	var mw middleware.Slice

	mw = append(mw, requestid.New(s.Log))
	mw = append(mw, cors.New(s.Router, config))

	return mw
}

// NewLogForRequestID returns a new logger with field request ID for better debugging / tracing request. This works
// only if requestid middleware generated a request id before, otherwise the field for request ID would be empty
func (s *Service) NewLogForRequestID(ctx context.Context) logrus.FieldLogger {
	return requestid.NewLoggerFromContext(ctx, s.Log)
}

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
