package middleware

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

const (
	// HeaderACAO header key for Access-Control-Allow-Origin
	HeaderACAO = "Access-Control-Allow-Origin"

	// HeaderACAM header key for Access-Control-Allow-Methods
	HeaderACAM = "Access-Control-Allow-Methods"

	// HeaderACAH header key for Access-Control-Allow-Headers
	HeaderACAH = "Access-Control-Allow-Headers"

	// HeaderACMA header key for Access-Control-Max-Age
	HeaderACMA = "Access-Control-Max-Age"

	// AccessControlMaxAgeDefault is the default max age of ACA* headers in seconds
	AccessControlMaxAgeDefault = 86400
)

type cors struct {
	Config Config
	Router *mux.Router
}

// NewCORS returns a middleware to handle CORS requests
func NewCORS(router *mux.Router, config Config) mux.MiddlewareFunc {
	middleware := &cors{Config: config, Router: router}
	return middleware.handler
}

func (c *cors) handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// Safari, MSIE & MS Edge doesn't follow the defined CORS flow yet and expect
		// an ACA* header for all method even if the content type is the standard form.
		// All other known browsers do correctly the OPTIONS request.

		origin := request.Header.Get("Origin")
		if c.Config.AccessControlAllowOrigins.IsIn(origin) || c.Config.AccessControlAllowOrigins.IsIn("*") {
			writer.Header().Set(HeaderACAO, origin)
			writer.Header().Set(HeaderACAM, "GET")                                  // TODO: get them dynamically from Router
			writer.Header().Set(HeaderACAH, "*")                                    // TODO: should be configurable
			writer.Header().Set(HeaderACMA, fmt.Sprint(AccessControlMaxAgeDefault)) // TODO: should be configurable
		}

		if request.Method == http.MethodOptions {
			writer.WriteHeader(http.StatusNoContent)
		} else {
			next.ServeHTTP(writer, request)
		}
	})
}
