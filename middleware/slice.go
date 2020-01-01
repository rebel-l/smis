package middleware

import "github.com/gorilla/mux"

// Slice represents a slice of mux.MiddlewareFunc
type Slice []mux.MiddlewareFunc

// WalkCallback represents the callback function executed by the Walk() method
type WalkCallback func(middleware mux.MiddlewareFunc) error

// Walk iterates over all elements of the slice and executes the given callback. If a callback throws an error,
// the loop stops and returns this error.
func (s Slice) Walk(f WalkCallback) error {
	for _, m := range s {
		if err := f(m); err != nil {
			return err
		}
	}

	return nil
}
