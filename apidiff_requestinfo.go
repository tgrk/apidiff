package apidiff

import (
	"io"
	"net/http"
)

// RequestInfo contains API request details
type RequestInfo struct {
	Session RecordedSession
	URL     string
	Verb    string
	Payload io.Reader
	Headers map[string]string
}

func NewRequest(session RecordedSession, url string) RequestInfo {
	return RequestInfo{
		Session: session,
		URL:     url,
		Verb:    http.MethodGet,
		Headers: make(map[string]string),
	}
}
