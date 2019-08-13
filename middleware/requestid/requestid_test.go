package requestid_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/golang/mock/gomock"

	"github.com/rebel-l/smis/middleware/requestid"
	"github.com/rebel-l/smis/tests/mocks/http_mock"
	"github.com/rebel-l/smis/tests/mocks/logrus_mock"

	"github.com/sirupsen/logrus"
)

func createHandler(ctrl *gomock.Controller) *http_mock.MockHandler {
	handler := http_mock.NewMockHandler(ctrl)
	handler.EXPECT().ServeHTTP(gomock.Any(), gomock.Any()).Times(1)
	return handler
}

func TestNew(t *testing.T) {
	ctrl := gomock.NewController(t)

	// we are not able to return mock from logrus.WithField(), so we simulate an Entry
	entry := &logrus.Entry{
		Logger: logrus.New(),
		Data: logrus.Fields{
			string(requestid.ContextKeyRequestID): uuid.New(),
		},
	}

	logMock := logrus_mock.NewMockFieldLogger(ctrl)
	logMock.EXPECT().
		WithField(gomock.Eq(string(requestid.ContextKeyRequestID)), gomock.Any()).
		Times(1).
		Return(entry)

	mw := requestid.New(logMock)
	handler := mw.Middleware(createHandler(ctrl))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	resp := w.Result()
	header := resp.Header.Get(requestid.HeaderRID)
	if header == "" {
		t.Errorf("header should contain %s but it was not set or empty string", requestid.HeaderRID)
	}

	if err := resp.Body.Close(); err != nil {
		t.Fatalf("failed to close body: %s", err)
	}
}

func TestGetID(t *testing.T) {
	ctx := context.Background()
	res := requestid.GetID(ctx)
	if res != "" {
		t.Errorf("context which didn't pass the middleware should not have a RequestID but got: %s", res)
	}
}
