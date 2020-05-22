package smis_test

import (
	"errors"
	"testing"

	"github.com/rebel-l/smis"
)

func TestError_GetInternal(t *testing.T) { // nolint: funlen
	testCases := []struct {
		name     string
		error    smis.Error
		expected string
	}{
		{
			name:     "internal only",
			error:    smis.Error{Internal: "internal error"},
			expected: "internal error",
		},
		{
			name:     "internal only with code",
			error:    smis.Error{Code: "ERR01", Internal: "internal error"},
			expected: "ERR01 - internal error",
		},
		{
			name:     "from external error",
			error:    smis.Error{External: "external error"},
			expected: "external error",
		},
		{
			name:     "from external error with code",
			error:    smis.Error{Code: "ERR02", External: "external error"},
			expected: "ERR02 - external error",
		},
		{
			name:     "internal & external",
			error:    smis.Error{External: "external error", Internal: "internal error"},
			expected: "internal error",
		},
		{
			name:     "internal & external with code",
			error:    smis.Error{Code: "ERR03", External: "external error", Internal: "internal error"},
			expected: "ERR03 - internal error",
		},
		{
			name:     "internal only with detail",
			error:    smis.Error{Internal: "internal error"}.WithDetails(errors.New("details")),
			expected: "internal error: details",
		},
		{
			name:     "internal only with detail & code",
			error:    smis.Error{Code: "ERR04", Internal: "internal error"}.WithDetails(errors.New("details")),
			expected: "ERR04 - internal error: details",
		},
		{
			name:     "from external error with detail",
			error:    smis.Error{External: "external error"}.WithDetails(errors.New("details")),
			expected: "external error: details",
		},
		{
			name:     "from external error with detail & code",
			error:    smis.Error{Code: "ERR05", External: "external error"}.WithDetails(errors.New("details")),
			expected: "ERR05 - external error: details",
		},
		{
			name: "internal & external with detail",
			error: smis.Error{
				External: "external error",
				Internal: "internal error",
			}.WithDetails(errors.New("details")),
			expected: "internal error: details",
		},
		{
			name: "internal & external with detail & code",
			error: smis.Error{
				Code:     "ERR06",
				External: "external error",
				Internal: "internal error",
			}.WithDetails(errors.New("details")),
			expected: "ERR06 - internal error: details",
		},
		{
			name:     "code only",
			error:    smis.Error{Code: "ERR06"},
			expected: "ERR06",
		},
		{
			name:     "code only with detail",
			error:    smis.Error{Code: "ERR07"}.WithDetails(errors.New("details")),
			expected: "ERR07: details",
		},
		{
			name:     "empty with detail",
			error:    smis.Error{}.WithDetails(errors.New("details")),
			expected: "details",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := testCase.error.GetInternal()
			if testCase.expected != actual {
				t.Errorf("expected error '%s' but got '%s'", testCase.expected, actual)
			}
		})
	}
}

func TestError_String(t *testing.T) {
	testCases := []struct {
		name     string
		error    smis.Error
		expected string
	}{
		{
			name:     "external only",
			error:    smis.Error{External: "external error"},
			expected: "external error",
		},
		{
			name:  "internal only",
			error: smis.Error{Internal: "internal error"},
		},
		{
			name:     "external only with code",
			error:    smis.Error{Code: "MYERR001", External: "external error"},
			expected: "MYERR001 - external error",
		},
		{
			name:     "internal only with code",
			error:    smis.Error{Code: "MYERR002", Internal: "internal error"},
			expected: "MYERR002",
		},
		{
			name:     "external & internal",
			error:    smis.Error{External: "external error", Internal: "internal error"},
			expected: "external error",
		},
		{
			name:     "external & internal with code",
			error:    smis.Error{Code: "MYERR003", External: "external error", Internal: "internal error"},
			expected: "MYERR003 - external error",
		},
		{
			name:     "code only",
			error:    smis.Error{Code: "MYERR004"},
			expected: "MYERR004",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := testCase.error.String()
			if testCase.expected != actual {
				t.Errorf("expected error '%s' but got '%s'", testCase.expected, actual)
			}
		})
	}
}

func TestError_WithDetails(t *testing.T) { // nolint: funlen
	testCases := []struct {
		name             string
		original         smis.Error
		details          error
		expectedNew      smis.Error
		expectedOriginal smis.Error
	}{
		{
			name: "internal only",
			original: smis.Error{
				Code:     "ERR001",
				Internal: "internal error",
			},
			details: errors.New("details"),
			expectedNew: smis.Error{
				Code:     "ERR001",
				Internal: "internal error",
				Details:  errors.New("details"),
			},
			expectedOriginal: smis.Error{
				Code:     "ERR001",
				Internal: "internal error",
			},
		},
		{
			name: "external only",
			original: smis.Error{
				Code:     "ERR002",
				External: "external error",
			},
			details: errors.New("details"),
			expectedNew: smis.Error{
				Code:     "ERR002",
				External: "external error",
				Internal: "external error",
				Details:  errors.New("details"),
			},
			expectedOriginal: smis.Error{
				Code:     "ERR002",
				External: "external error",
			},
		},
		{
			name: "internal & external",
			original: smis.Error{
				Code:     "ERR003",
				External: "external error",
				Internal: "internal error",
			},
			details: errors.New("details"),
			expectedNew: smis.Error{
				Code:     "ERR003",
				External: "external error",
				Internal: "internal error",
				Details:  errors.New("details"),
			},
			expectedOriginal: smis.Error{
				Code:     "ERR003",
				External: "external error",
				Internal: "internal error",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := testCase.original.WithDetails(testCase.details)
			assertError(t, "new", testCase.expectedNew, actual)
			assertError(t, "original", testCase.original, testCase.expectedOriginal)
		})
	}
}

func assertError(t *testing.T, msg string, expected, actual smis.Error) {
	t.Helper()

	if expected.StatusCode != actual.StatusCode {
		t.Errorf("%s: expected status code %d but got %d", msg, expected.StatusCode, actual.StatusCode)
	}

	if expected.Code != actual.Code {
		t.Errorf("%s: expected code '%s' but got '%s'", msg, expected.Code, actual.Code)
	}

	if expected.External != actual.External {
		t.Errorf("%s: expected external message '%s' but got '%s'", msg, expected.External, actual.External)
	}

	if expected.Internal != actual.Internal {
		t.Errorf("%s: expected internal message '%s' but got '%s'", msg, expected.Internal, actual.Internal)
	}

	if expected.Details == nil && actual.Details != nil ||
		expected.Details != nil && actual.Details == nil ||
		expected.Details != nil && actual.Details != nil && expected.Details.Error() != actual.Details.Error() {
		t.Errorf("%s: expected detail '%v' but got '%v'", msg, expected.Details, actual.Details)
	}
}
