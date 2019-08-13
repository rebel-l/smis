package requestid

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/gorilla/mux"
)

type contextKey string

const (
	// ContextKeyRequestID is the key in the context where to find the RequestID
	ContextKeyRequestID contextKey = "requestID"

	// HeaderRID is the key of the header containing the RequestID
	HeaderRID = "X-Request-ID"
)

// New returns the middleware handler generating the RequestID
func New() mux.MiddlewareFunc {
	return handler
}

// GetID returns the RequestID set to the context. Is empty if the context doesn't contain any RequestID
func GetID(ctx context.Context) string {
	requestID := ctx.Value(ContextKeyRequestID)
	if res, ok := requestID.(string); ok {
		return res
	}

	return ""
}

func handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// add request ID
		ctx := attachRequestID(request.Context())
		request = request.WithContext(ctx)

		// handle next
		next.ServeHTTP(writer, request)

		// add request ID to header
		writer.Header().Set(HeaderRID, GetID(ctx))
	})
}

func attachRequestID(ctx context.Context) context.Context {
	return context.WithValue(ctx, ContextKeyRequestID, uuid.New().String())
}
