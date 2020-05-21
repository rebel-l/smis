package smis_test

import (
	"errors"
	"fmt"
	"io/ioutil"
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
			code:         http.StatusNotFound,
			payload:      struct{}{},
			expectedCode: http.StatusNotFound,
			expectedBody: "{}",
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

			contentType := w.Header().Get(smis.HeaderKeyContentType)
			if contentType != smis.HeaderContentTypeJSON {
				t.Errorf("expected content type '%s' but got '%s'", smis.HeaderContentTypeJSON, contentType)
			}

			if testCase.expectedBody != w.Body.String() {
				t.Errorf("expected body '%s' but got '%s'", testCase.expectedBody, w.Body.String())
			}
		})
	}
}

func TestResponse_WriteJSON_Error(t *testing.T) {
	testCases := []struct {
		// TODO: add a test for Marshal Error (maybe this one should it be)
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

			writerMock := http_mock.NewMockResponseWriter(ctrl)
			writerMock.EXPECT().WriteHeader(http.StatusOK).Times(1)
			header := http.Header{}
			writerMock.EXPECT().Header().Return(header).Times(1)
			writerMock.EXPECT().Write(gomock.Any()).Return(0, errors.New("fail")).Times(1)

			actual.WriteJSON(writerMock, 200, "")

			contentType := header.Get(smis.HeaderKeyContentType)
			if contentType != smis.HeaderContentTypeJSON {
				t.Errorf("expected content type '%s' but got '%s'", smis.HeaderContentTypeJSON, contentType)
			}
		})
	}
}

func TestResponse_WriteJSON_NoWriter(t *testing.T) {
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
				logMock.EXPECT().Error(gomock.Eq("writer is nil")).Times(1)
				actual.Log = logMock
			} else {
				logMock.EXPECT().Error(gomock.Any()).Times(0)
			}

			actual.WriteJSON(nil, 200, "")
		})
	}
}

func TestResponse_WriteJSONError(t *testing.T) { // nolint: funlen
	testCases := []struct {
		name               string
		error              smis.Error
		expectedStatusCode int
		expectedHeader     string
		expectedBody       string
		expectedLogErr     string
		expectedLogWarn    string
	}{
		{
			name: "status code 100 - external only",
			error: smis.Error{
				StatusCode: http.StatusContinue,
				Code:       "MYERR100",
				External:   "external error",
			},
			expectedStatusCode: http.StatusContinue,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR100\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR100 - external error",
			expectedLogWarn: fmt.Sprintf(
				"status code for error response should be of 4xx or 5xx but used %d", http.StatusContinue,
			),
		},
		{
			name: "status code 101 - internal only",
			error: smis.Error{
				StatusCode: http.StatusSwitchingProtocols,
				Code:       "MYERR101",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusSwitchingProtocols,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR101\",\"error\":\"\"}",
			expectedLogErr:     "MYERR101 - internal error",
			expectedLogWarn: fmt.Sprintf(
				"status code for error response should be of 4xx or 5xx but used %d", http.StatusSwitchingProtocols,
			),
		},
		{
			name: "status code 102 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusProcessing,
				Code:       "MYERR102",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusProcessing,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR102\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR102 - internal error",
			expectedLogWarn: fmt.Sprintf(
				"status code for error response should be of 4xx or 5xx but used %d", http.StatusProcessing,
			),
		},
		{
			name: "status code 103 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusEarlyHints,
				Code:       "MYERR103",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusEarlyHints,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR103\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR103 - internal error",
			expectedLogWarn: fmt.Sprintf(
				"status code for error response should be of 4xx or 5xx but used %d", http.StatusEarlyHints,
			),
		},
		{
			name: "status code 200 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusOK,
				Code:       "MYERR200",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusOK,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR200\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR200 - internal error",
			expectedLogWarn: fmt.Sprintf(
				"status code for error response should be of 4xx or 5xx but used %d", http.StatusOK,
			),
		},
		{
			name: "status code 201 - external & internal without code",
			error: smis.Error{
				StatusCode: http.StatusCreated,
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusCreated,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"\",\"error\":\"external error\"}",
			expectedLogErr:     "internal error",
			expectedLogWarn: fmt.Sprintf(
				"status code for error response should be of 4xx or 5xx but used %d", http.StatusCreated,
			),
		},
		{
			name: "status code 202 - external & internal with detail",
			error: smis.Error{
				StatusCode: http.StatusAccepted,
				Code:       "MYERR202",
				External:   "external error",
				Internal:   "internal error",
				Details:    errors.New("details"),
			},
			expectedStatusCode: http.StatusAccepted,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR202\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR202 - internal error: details",
			expectedLogWarn: fmt.Sprintf(
				"status code for error response should be of 4xx or 5xx but used %d", http.StatusAccepted,
			),
		},
		{
			name: "status code 203 - external & internal without code but with details",
			error: smis.Error{
				StatusCode: http.StatusNonAuthoritativeInfo,
				External:   "external error",
				Internal:   "internal error",
				Details:    errors.New("details"),
			},
			expectedStatusCode: http.StatusNonAuthoritativeInfo,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"\",\"error\":\"external error\"}",
			expectedLogErr:     "internal error: details",
			expectedLogWarn: fmt.Sprintf(
				"status code for error response should be of 4xx or 5xx but used %d", http.StatusNonAuthoritativeInfo,
			),
		},
		{
			name: "status code 204 - external only",
			error: smis.Error{
				StatusCode: http.StatusNoContent,
				Code:       "MYERR204",
				External:   "external error",
			},
			expectedStatusCode: http.StatusNoContent,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR204\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR204 - external error",
			expectedLogWarn: fmt.Sprintf(
				"status code for error response should be of 4xx or 5xx but used %d", http.StatusNoContent,
			),
		},
		{
			name: "status code 205 - external only without code",
			error: smis.Error{
				StatusCode: http.StatusResetContent,
				External:   "external error",
			},
			expectedStatusCode: http.StatusResetContent,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"\",\"error\":\"external error\"}",
			expectedLogErr:     "external error",
			expectedLogWarn: fmt.Sprintf(
				"status code for error response should be of 4xx or 5xx but used %d", http.StatusResetContent,
			),
		},
		{
			name: "status code 206 - external only with details",
			error: smis.Error{
				StatusCode: http.StatusPartialContent,
				Code:       "MYERR206",
				External:   "external error",
				Details:    errors.New("details"),
			},
			expectedStatusCode: http.StatusPartialContent,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR206\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR206 - external error: details",
			expectedLogWarn: fmt.Sprintf(
				"status code for error response should be of 4xx or 5xx but used %d", http.StatusPartialContent,
			),
		},
		{
			name: "status code 207 - external only without code but with details",
			error: smis.Error{
				StatusCode: http.StatusMultiStatus,
				External:   "external error",
				Details:    errors.New("details"),
			},
			expectedStatusCode: http.StatusMultiStatus,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"\",\"error\":\"external error\"}",
			expectedLogErr:     "external error: details",
			expectedLogWarn: fmt.Sprintf(
				"status code for error response should be of 4xx or 5xx but used %d", http.StatusMultiStatus,
			),
		},
		{
			name: "status code 208 - without external & internal",
			error: smis.Error{
				StatusCode: http.StatusAlreadyReported,
				Code:       "MYERR208",
			},
			expectedStatusCode: http.StatusAlreadyReported,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR208\",\"error\":\"\"}",
			expectedLogErr:     "MYERR208",
			expectedLogWarn: fmt.Sprintf(
				"status code for error response should be of 4xx or 5xx but used %d", http.StatusAlreadyReported,
			),
		},
		{
			name: "status code 226 - without external & internal but with details",
			error: smis.Error{
				StatusCode: http.StatusIMUsed,
				Code:       "MYERR226",
				Details:    errors.New("details"),
			},
			expectedStatusCode: http.StatusIMUsed,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR226\",\"error\":\"\"}",
			expectedLogErr:     "MYERR226: details",
			expectedLogWarn: fmt.Sprintf(
				"status code for error response should be of 4xx or 5xx but used %d", http.StatusIMUsed,
			),
		},
		{
			name: "status code 300 - external & internal with detail",
			error: smis.Error{
				StatusCode: http.StatusMultipleChoices,
				Code:       "MYERR300",
				External:   "external error",
				Internal:   "internal error",
				Details:    errors.New("details"),
			},
			expectedStatusCode: http.StatusMultipleChoices,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR300\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR300 - internal error: details",
			expectedLogWarn: fmt.Sprintf(
				"status code for error response should be of 4xx or 5xx but used %d", http.StatusMultipleChoices,
			),
		},
		{
			name: "status code 301 - external & internal without code but with details",
			error: smis.Error{
				StatusCode: http.StatusMovedPermanently,
				External:   "external error",
				Internal:   "internal error",
				Details:    errors.New("details"),
			},
			expectedStatusCode: http.StatusMovedPermanently,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"\",\"error\":\"external error\"}",
			expectedLogErr:     "internal error: details",
			expectedLogWarn: fmt.Sprintf(
				"status code for error response should be of 4xx or 5xx but used %d", http.StatusMovedPermanently,
			),
		},
		{
			name: "status code 302 - internal only",
			error: smis.Error{
				StatusCode: http.StatusFound,
				Code:       "MYERR302",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusFound,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR302\",\"error\":\"\"}",
			expectedLogErr:     "MYERR302 - internal error",
			expectedLogWarn: fmt.Sprintf(
				"status code for error response should be of 4xx or 5xx but used %d", http.StatusFound,
			),
		},
		{
			name: "status code 303 - internal only without code",
			error: smis.Error{
				StatusCode: http.StatusSeeOther,
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusSeeOther,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"\",\"error\":\"\"}",
			expectedLogErr:     "internal error",
			expectedLogWarn: fmt.Sprintf(
				"status code for error response should be of 4xx or 5xx but used %d", http.StatusSeeOther,
			),
		},
		{
			name: "status code 304 - internal only with details",
			error: smis.Error{
				StatusCode: http.StatusNotModified,
				Code:       "MYERR304",
				Internal:   "internal error",
				Details:    errors.New("details"),
			},
			expectedStatusCode: http.StatusNotModified,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR304\",\"error\":\"\"}",
			expectedLogErr:     "MYERR304 - internal error: details",
			expectedLogWarn: fmt.Sprintf(
				"status code for error response should be of 4xx or 5xx but used %d", http.StatusNotModified,
			),
		},
		{
			name: "status code 305 - internal only without code but with details",
			error: smis.Error{
				StatusCode: http.StatusUseProxy,
				Internal:   "internal error",
				Details:    errors.New("details"),
			},
			expectedStatusCode: http.StatusUseProxy,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"\",\"error\":\"\"}",
			expectedLogErr:     "internal error: details",
			expectedLogWarn: fmt.Sprintf(
				"status code for error response should be of 4xx or 5xx but used %d", http.StatusUseProxy,
			),
		},
		{
			name: "status code 307 - without external & internal",
			error: smis.Error{
				StatusCode: http.StatusTemporaryRedirect,
				Code:       "MYERR307",
			},
			expectedStatusCode: http.StatusTemporaryRedirect,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR307\",\"error\":\"\"}",
			expectedLogErr:     "MYERR307",
			expectedLogWarn: fmt.Sprintf(
				"status code for error response should be of 4xx or 5xx but used %d", http.StatusTemporaryRedirect,
			),
		},
		{
			name: "status code 308 - without external & internal but with details",
			error: smis.Error{
				StatusCode: http.StatusPermanentRedirect,
				Code:       "MYERR308",
				Details:    errors.New("details"),
			},
			expectedStatusCode: http.StatusPermanentRedirect,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR308\",\"error\":\"\"}",
			expectedLogErr:     "MYERR308: details",
			expectedLogWarn: fmt.Sprintf(
				"status code for error response should be of 4xx or 5xx but used %d", http.StatusPermanentRedirect,
			),
		},
		{
			name: "status code 400 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusBadRequest,
				Code:       "MYERR400",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR400\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR400 - internal error",
		},
		{
			name: "status code 401 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusUnauthorized,
				Code:       "MYERR401",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusUnauthorized,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR401\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR401 - internal error",
		},
		{
			name: "status code 402 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusPaymentRequired,
				Code:       "MYERR402",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusPaymentRequired,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR402\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR402 - internal error",
		},
		{
			name: "status code 403 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusForbidden,
				Code:       "MYERR403",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusForbidden,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR403\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR403 - internal error",
		},
		{
			name: "status code 404 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusNotFound,
				Code:       "MYERR404",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusNotFound,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR404\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR404 - internal error",
		},
		{
			name: "status code 405 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusMethodNotAllowed,
				Code:       "MYERR405",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusMethodNotAllowed,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR405\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR405 - internal error",
		},
		{
			name: "status code 406 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusNotAcceptable,
				Code:       "MYERR406",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusNotAcceptable,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR406\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR406 - internal error",
		},
		{
			name: "status code 407 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusProxyAuthRequired,
				Code:       "MYERR407",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusProxyAuthRequired,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR407\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR407 - internal error",
		},
		{
			name: "status code 408 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusRequestTimeout,
				Code:       "MYERR408",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusRequestTimeout,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR408\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR408 - internal error",
		},
		{
			name: "status code 409 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusConflict,
				Code:       "MYERR409",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusConflict,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR409\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR409 - internal error",
		},
		{
			name: "status code 410 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusGone,
				Code:       "MYERR410",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusGone,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR410\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR410 - internal error",
		},
		{
			name: "status code 411 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusLengthRequired,
				Code:       "MYERR411",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusLengthRequired,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR411\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR411 - internal error",
		},
		{
			name: "status code 412 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusPreconditionFailed,
				Code:       "MYERR412",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusPreconditionFailed,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR412\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR412 - internal error",
		},
		{
			name: "status code 413 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusRequestEntityTooLarge,
				Code:       "MYERR413",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusRequestEntityTooLarge,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR413\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR413 - internal error",
		},
		{
			name: "status code 414 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusRequestURITooLong,
				Code:       "MYERR414",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusRequestURITooLong,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR414\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR414 - internal error",
		},
		{
			name: "status code 415 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusUnsupportedMediaType,
				Code:       "MYERR415",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusUnsupportedMediaType,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR415\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR415 - internal error",
		},
		{
			name: "status code 416 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusRequestedRangeNotSatisfiable,
				Code:       "MYERR416",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusRequestedRangeNotSatisfiable,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR416\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR416 - internal error",
		},
		{
			name: "status code 417 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusExpectationFailed,
				Code:       "MYERR417",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusExpectationFailed,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR417\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR417 - internal error",
		},
		{
			name: "status code 418 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusTeapot,
				Code:       "MYERR418",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusTeapot,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR418\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR418 - internal error",
		},
		{
			name: "status code 421 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusMisdirectedRequest,
				Code:       "MYERR421",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusMisdirectedRequest,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR421\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR421 - internal error",
		},
		{
			name: "status code 422 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusUnprocessableEntity,
				Code:       "MYERR422",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusUnprocessableEntity,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR422\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR422 - internal error",
		},
		{
			name: "status code 423 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusLocked,
				Code:       "MYERR423",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusLocked,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR423\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR423 - internal error",
		},
		{
			name: "status code 424 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusFailedDependency,
				Code:       "MYERR424",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusFailedDependency,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR424\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR424 - internal error",
		},
		{
			name: "status code 425 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusTooEarly,
				Code:       "MYERR425",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusTooEarly,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR425\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR425 - internal error",
		},
		{
			name: "status code 426 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusUpgradeRequired,
				Code:       "MYERR426",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusUpgradeRequired,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR426\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR426 - internal error",
		},
		{
			name: "status code 428 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusPreconditionRequired,
				Code:       "MYERR428",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusPreconditionRequired,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR428\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR428 - internal error",
		},
		{
			name: "status code 429 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusTooManyRequests,
				Code:       "MYERR429",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusTooManyRequests,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR429\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR429 - internal error",
		},
		{
			name: "status code 431 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusRequestHeaderFieldsTooLarge,
				Code:       "MYERR431",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusRequestHeaderFieldsTooLarge,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR431\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR431 - internal error",
		},
		{
			name: "status code 451 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusUnavailableForLegalReasons,
				Code:       "MYERR451",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusUnavailableForLegalReasons,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR451\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR451 - internal error",
		},
		{
			name: "status code 500 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusInternalServerError,
				Code:       "MYERR500",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusInternalServerError,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR500\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR500 - internal error",
		},
		{
			name: "status code 501 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusNotImplemented,
				Code:       "MYERR501",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusNotImplemented,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR501\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR501 - internal error",
		},
		{
			name: "status code 502 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusBadGateway,
				Code:       "MYERR502",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusBadGateway,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR502\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR502 - internal error",
		},
		{
			name: "status code 503 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusServiceUnavailable,
				Code:       "MYERR503",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusServiceUnavailable,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR503\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR503 - internal error",
		},
		{
			name: "status code 504 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusGatewayTimeout,
				Code:       "MYERR504",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusGatewayTimeout,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR504\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR504 - internal error",
		},
		{
			name: "status code 505 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusHTTPVersionNotSupported,
				Code:       "MYERR505",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusHTTPVersionNotSupported,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR505\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR505 - internal error",
		},
		{
			name: "status code 506 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusVariantAlsoNegotiates,
				Code:       "MYERR506",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusVariantAlsoNegotiates,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR506\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR506 - internal error",
		},
		{
			name: "status code 507 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusInsufficientStorage,
				Code:       "MYERR507",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusInsufficientStorage,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR507\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR507 - internal error",
		},
		{
			name: "status code 508 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusLoopDetected,
				Code:       "MYERR508",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusLoopDetected,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR508\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR508 - internal error",
		},
		{
			name: "status code 510 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusNotExtended,
				Code:       "MYERR510",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusNotExtended,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR510\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR510 - internal error",
		},
		{
			name: "status code 511 - external & internal",
			error: smis.Error{
				StatusCode: http.StatusNetworkAuthenticationRequired,
				Code:       "MYERR511",
				External:   "external error",
				Internal:   "internal error",
			},
			expectedStatusCode: http.StatusNetworkAuthenticationRequired,
			expectedHeader:     smis.HeaderContentTypeJSON,
			expectedBody:       "{\"code\":\"MYERR511\",\"error\":\"external error\"}",
			expectedLogErr:     "MYERR511 - internal error",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			logMock := logrus_mock.NewMockFieldLogger(ctrl)
			logMock.EXPECT().Error(gomock.Eq(testCase.expectedLogErr)).Times(1)
			if testCase.expectedLogWarn != "" {
				logMock.EXPECT().Warn(gomock.Eq(testCase.expectedLogWarn)).Times(1)
			}

			resp := &smis.Response{Log: logMock}
			w := httptest.NewRecorder()
			resp.WriteJSONError(w, testCase.error)
			ctrl.Finish()
			res := w.Result()

			// Assert Status Code
			if testCase.expectedStatusCode != res.StatusCode {
				t.Errorf("expected status code %d but got %d", testCase.expectedStatusCode, res.StatusCode)
			}

			// Assert Header
			header := w.Header().Get(smis.HeaderKeyContentType)
			if testCase.expectedHeader != header {
				t.Errorf("expected content type '%s' but got '%s'", testCase.expectedHeader, header)
			}

			// Assert Body
			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				t.Fatalf("failed to read body: %s", err)
			}

			if testCase.expectedBody != string(body) {
				t.Errorf("expected body to be '%s' but got '%s'", testCase.expectedBody, string(body))
			}

			if err := res.Body.Close(); err != nil {
				t.Fatalf("failed to close body: %s", err)
			}
		})
	}
}

func TestResponse_WriteJSONError_WithoutLogger(t *testing.T) {
	resp := &smis.Response{}
	w := httptest.NewRecorder()
	resp.WriteJSONError(w, smis.Error{StatusCode: http.StatusContinue, Code: "ERR-C3PO"})
	res := w.Result()

	// Assert Status Code
	if http.StatusContinue != res.StatusCode {
		t.Errorf("expected status code %d but got %d", http.StatusContinue, res.StatusCode)
	}

	// Assert Header
	header := w.Header().Get(smis.HeaderKeyContentType)
	if smis.HeaderContentTypeJSON != header {
		t.Errorf("expected content type '%s' but got '%s'", smis.HeaderContentTypeJSON, header)
	}

	// Assert Body
	expected := "{\"code\":\"ERR-C3PO\",\"error\":\"\"}"

	actual, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("failed to read body: %s", err)
	}

	if expected != string(actual) {
		t.Errorf("expected body to be '%s' but got '%s'", expected, string(actual))
	}

	if err := res.Body.Close(); err != nil {
		t.Fatalf("failed to close body: %s", err)
	}
}
