package smis_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/rebel-l/smis/tests/mocks/http_mock"
	"github.com/rebel-l/smis/tests/mocks/logrus_mock"

	"github.com/sirupsen/logrus"

	"github.com/rebel-l/smis"
)

func TestResponse_WriteJSON(t *testing.T) {
	testCases := []struct {
		name         string
		actual       *smis.Response
		code         int
		payload      interface{}
		expectedCode int
		expectedBody string
	}{
		{
			name:         "response is nil",
			expectedCode: http.StatusOK,
		},
		{
			name:   "success",
			actual: &smis.Response{},
			code:   http.StatusOK,
			payload: struct {
				Name string `json:"name"`
			}{Name: "test"},
			expectedCode: http.StatusOK,
			expectedBody: `{"name":"test"}`,
		},
		{
			name:   "success with log",
			actual: &smis.Response{Log: logrus.New()},
			code:   http.StatusOK,
			payload: struct {
				Name string `json:"name"`
			}{Name: "Herbert"},
			expectedCode: http.StatusOK,
			expectedBody: `{"name":"Herbert"}`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			testCase.actual.WriteJSON(w, testCase.code, testCase.payload)

			if testCase.expectedCode != w.Code {
				t.Errorf("expected code %d but got %d", testCase.expectedCode, w.Code)
			}

			if testCase.expectedBody != w.Body.String() {
				t.Errorf("expected body '%s' but got '%s'", testCase.expectedBody, w.Body.String())
			}
		})
	}
}

func TestResponse_WriteJSON_Error(t *testing.T) {
	testCases := []struct {
		name string
		log  bool
	}{
		{
			name: "with logger",
			log:  true,
		},
		{
			name: "without logger",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			actual := &smis.Response{}

			logMock := logrus_mock.NewMockFieldLogger(ctrl)
			if testCase.log {
				logMock.EXPECT().Error(gomock.Eq("failed to write response: fail")).Times(1)
				actual.Log = logMock
			} else {
				logMock.EXPECT().Error(gomock.Any()).Times(0)
			}

			respMock := http_mock.NewMockResponseWriter(ctrl)
			respMock.EXPECT().WriteHeader(http.StatusOK).Times(1)
			respMock.EXPECT().Write(gomock.Any()).Return(0, errors.New("fail")).Times(1)

			actual.WriteJSON(respMock, 200, "")
		})
	}
}
