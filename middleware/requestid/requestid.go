package requestid

import (
	"context"
	"net/http"

	"github.com/sirupsen/logrus"

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

type requestID struct {
	Log logrus.FieldLogger
}

// New returns the middleware handler generating the RequestID.
func New(log logrus.FieldLogger) mux.MiddlewareFunc {
	mw := &requestID{Log: log}
	return mw.handler
}

// GetID returns the RequestID set to the context. Is empty if the context doesn't contain any RequestID.
func GetID(ctx context.Context) string {
	requestID := ctx.Value(ContextKeyRequestID)
	if res, ok := requestID.(string); ok {
		return res
	}

	return ""
}

func (r *requestID) handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// add request ID
		ctx := attachRequestID(request.Context())
		request = request.WithContext(ctx)
		log := NewLoggerFromContext(ctx, r.Log)
		log.Infof("Request start: %s - %s", request.Method, request.RequestURI)

		// handle next
		next.ServeHTTP(writer, request)

		// add request ID to header
		writer.Header().Set(HeaderRID, GetID(ctx))
		log.Infof("Request finished: %s - %s", request.Method, request.RequestURI)
	})
}

func attachRequestID(ctx context.Context) context.Context {
	return context.WithValue(ctx, ContextKeyRequestID, uuid.New().String())
}
