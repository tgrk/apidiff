package apidiff

import (
	"net/http"
)

// RequestInfo contains API request details
type RequestInfo struct {
	URL     string      `yaml:"url"`
	Method  string      `yaml:"method"`
	Payload string      `yaml:"body"`
	Headers http.Header `yaml:"headers"`
}
