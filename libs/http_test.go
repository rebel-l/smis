package libs_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	"github.com/rebel-l/smis/libs"
)

func TestGetMethodsForCurrentURI(t *testing.T) {
	router := mux.NewRouter()
	handler := func(_ http.ResponseWriter, _ *http.Request) {}
	router.HandleFunc("/myEndpoint", handler).Methods(http.MethodGet, http.MethodPut)

	expected := "GET,PUT"

	request := httptest.NewRequest(http.MethodPut, "/myEndpoint", nil)
	got := libs.GetMethodsForCurrentURI(request, router).String()

	if expected != got {
		t.Errorf("expected methods '%s' but got '%s'", expected, got)
	}
}
