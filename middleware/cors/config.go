package cors

import "github.com/rebel-l/go-utils/slice"

// Config provides a configuration for the CORS middleware
type Config struct {
	AccessControlAllowOrigins slice.StringSlice `json:"access_control_allow_origins,omitempty"`
	AccessContolAllowHeaders  slice.StringSlice `json:"access_contol_allow_headers,omitempty"`
	AccessControlMaxAge       int               `json:"access_control_max_age,omitempty"`
}
