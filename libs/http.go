package libs

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/rebel-l/go-utils/slice"
)

func GetMethodsForCurrentURI(request *http.Request, router *mux.Router) slice.StringSlice {
	var methods slice.StringSlice
	possibleMethods := GetAllowedHTTPMethods()
	methods = append(methods, http.MethodOptions) // OPTIONS is the default preflight method and is reserved

	for _, m := range possibleMethods {
		simReq := &http.Request{Method: m, URL: request.URL, RequestURI: request.RequestURI}

		match := &mux.RouteMatch{}
		if !router.Match(simReq, match) || match.MatchErr != nil {
			continue
		}

		methods = append(methods, m)
	}

	return methods
}

func GetAllowedHTTPMethods() slice.StringSlice {
	return slice.StringSlice{
		http.MethodConnect,
		http.MethodDelete,
		http.MethodGet,
		http.MethodHead,
		http.MethodPatch,
		http.MethodPost,
		http.MethodPut,
		http.MethodTrace,
	}
}
