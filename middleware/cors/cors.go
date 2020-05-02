package cors

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/rebel-l/go-utils/slice"

	"github.com/gorilla/mux"
)

const (
	// AccessControlMaxAgeDefault is the default max age of ACA* headers in seconds
	AccessControlMaxAgeDefault = 86400

	// HeaderACAO is the header key for Access-Control-Allow-Origin
	HeaderACAO = "Access-Control-Allow-Origin"

	// HeaderACAM is the header key for Access-Control-Allow-Methods
	HeaderACAM = "Access-Control-Allow-Methods"

	// HeaderACAH is the header key for Access-Control-Allow-Headers
	HeaderACAH = "Access-Control-Allow-Headers"

	// HeaderACMA is the header key for Access-Control-Max-Age
	HeaderACMA = "Access-Control-Max-Age"

	// HeaderACRM is the header key for Access-Control-Request-Method
	HeaderACRM = "Access-Control-Request-Method"

	// HeaderOrigin is the header key for Origin
	HeaderOrigin = "Origin"
)

type cors struct {
	Config Config
	Router *mux.Router
}

// New returns a middleware to handle CORS requests.
func New(router *mux.Router, config Config) mux.MiddlewareFunc {
	middleware := &cors{Config: config, Router: router}
	return middleware.handler
}

func (c *cors) handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// Safari, MSIE & MS Edge doesn't follow the defined CORS flow yet and expect
		// an ACA* header for all method even if the content type is the standard form.
		// All other known browsers do correctly the OPTIONS request.

		origin := request.Header.Get(HeaderOrigin)
		if c.Config.AccessControlAllowOrigins.IsIn(origin) || c.Config.AccessControlAllowOrigins.IsIn("*") {
			// origin
			writer.Header().Set(HeaderACAO, origin)

			// methods
			writer.Header().Set(HeaderACAM, c.getMethods(request))

			// header
			writer.Header().Set(HeaderACAH, strings.Join(c.Config.AccessControlAllowHeaders, ","))

			// max age
			writer.Header().Set(HeaderACMA, c.getMaxAge())
		}

		if request.Method == http.MethodOptions {
			writer.WriteHeader(http.StatusNoContent)
		} else {
			next.ServeHTTP(writer, request)
		}
	})
}

func (c *cors) getMaxAge() string {
	maxAge := c.Config.AccessControlMaxAge
	if maxAge <= 0 {
		maxAge = AccessControlMaxAgeDefault
	}

	return fmt.Sprint(maxAge)
}

func (c *cors) getMethods(request *http.Request) string {
	var methods slice.StringSlice

	reqMethod := request.Header.Get(HeaderACRM)
	if reqMethod == "" {
		reqMethod = request.Method
	}

	simReq := &http.Request{
		Method:     reqMethod,
		URL:        request.URL,
		RequestURI: request.RequestURI,
	}

	routerMatch := &mux.RouteMatch{}
	if c.Router.Match(simReq, routerMatch) && routerMatch.MatchErr == nil {
		methods, _ = routerMatch.Route.GetMethods()
	}

	if methods == nil || methods.IsNotIn(http.MethodOptions) {
		methods = append(methods, http.MethodOptions)
	}

	return strings.Join(methods, ",")
}
