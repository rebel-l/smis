//go:generate mockgen -destination ./tests/mocks/io_mock/readcloser.go -package io_mock io ReadCloser

package smis_test

import (
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/rebel-l/smis/tests/mocks/io_mock"

	"github.com/rebel-l/smis"
)

func TestParseJSONBody(t *testing.T) {
	testCases := []struct {
		name        string
		testData    string
		expected    string
		expectedErr error
	}{
		{
			name:     "valid JSON",
			testData: "{\"name\":\"R2D2\"}",
			expected: "R2D2",
		},
		{
			name:        "invalid JSON",
			expectedErr: smis.ErrParseBody,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := &struct {
				Name string `json:"name"`
			}{}

			r := ioutil.NopCloser(strings.NewReader(testCase.testData))
			err := smis.ParseJSONBody(r, got)
			if !errors.Is(err, testCase.expectedErr) {
				t.Errorf("expected error '%v' but got '%v'", testCase.expectedErr, err)
			}

			if testCase.expected != got.Name {
				t.Errorf("expected parsed body has name '%s' but got '%s'", testCase.expected, got.Name)
			}
		})
	}
}

func TestParseJSONBody_ReadCloserError(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := io_mock.NewMockReadCloser(ctrl)
	reader.EXPECT().Read(gomock.Any()).Times(1).Return(0, errors.New("failed"))

	err := smis.ParseJSONBody(reader, struct{}{})
	if !errors.Is(err, smis.ErrParseBody) {
		t.Errorf("expected error %v but got %v", smis.ErrParseBody, err)
	}
}

func TestParseJSONRequestBody(t *testing.T) {
	testCases := []struct {
		name        string
		header      string
		testData    string
		expected    string
		expectedErr error
	}{
		{
			name:     "valid JSON",
			header:   smis.HeaderContentTypeJSON,
			testData: "{\"name\":\"R2D2\"}",
			expected: "R2D2",
		},
		{
			name:        "invalid JSON",
			header:      smis.HeaderContentTypeJSON,
			expectedErr: smis.ErrParseBody,
		},
		{
			name:        "wrong header",
			header:      smis.HeaderContentTypePlain,
			expectedErr: smis.ErrParseBody,
		},
		{
			name:        "no header",
			expectedErr: smis.ErrParseBody,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := &struct {
				Name string `json:"name"`
			}{}

			r, err := http.NewRequest(http.MethodGet, "/something", strings.NewReader(testCase.testData))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			if testCase.header != "" {
				r.Header = http.Header{}
				r.Header.Set(smis.HeaderKeyContentType, testCase.header)
			}

			err = smis.ParseJSONRequestBody(r, got)
			if !errors.Is(err, testCase.expectedErr) {
				t.Errorf("expected error '%v' but got '%v'", testCase.expectedErr, err)
			}

			if testCase.expected != got.Name {
				t.Errorf("expected parsed body has name '%s' but got '%s'", testCase.expected, got.Name)
			}
		})
	}
}

func TestParseJSONResponseBody(t *testing.T) {
	testCases := []struct {
		name        string
		header      string
		testData    string
		expected    string
		expectedErr error
	}{
		{
			name:     "valid JSON",
			header:   smis.HeaderContentTypeJSON,
			testData: "{\"name\":\"R2D2\"}",
			expected: "R2D2",
		},
		{
			name:        "invalid JSON",
			header:      smis.HeaderContentTypeJSON,
			expectedErr: smis.ErrParseBody,
		},
		{
			name:        "wrong header",
			header:      smis.HeaderContentTypePlain,
			expectedErr: smis.ErrParseBody,
		},
		{
			name:        "no header",
			expectedErr: smis.ErrParseBody,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := &struct {
				Name string `json:"name"`
			}{}

			r := &http.Response{
				Body: ioutil.NopCloser(strings.NewReader(testCase.testData)),
			}

			if testCase.header != "" {
				r.Header = http.Header{}
				r.Header.Set(smis.HeaderKeyContentType, testCase.header)
			}

			err := smis.ParseJSONResponseBody(r, got)
			if !errors.Is(err, testCase.expectedErr) {
				t.Errorf("expected error '%v' but got '%v'", testCase.expectedErr, err)
			}

			if testCase.expected != got.Name {
				t.Errorf("expected parsed body has name '%s' but got '%s'", testCase.expected, got.Name)
			}
		})
	}
}
