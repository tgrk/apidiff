package apidiff

import (
	"net/http"
	"time"
)

// Options holds shared CLI arguments from user
type Options struct {
	Verbose bool
	Name    string
}

// RecordedSession represents stored API session
type RecordedSession struct {
	Name         string
	Path         string
	Interactions []RecordedInteraction
	Created      time.Time
}

// RecordedInteraction represents recorded API interaction
type RecordedInteraction struct {
	URL        string
	Method     string
	StatusCode int
	Stats      RequestStats
}

// RequestStats hold HTTP stats metrics
type RequestStats struct {
	DNSLookup        int `yaml:"dns_lookup"`
	TCPConnection    int `yaml:"tcp_connection"`
	TLSHandshake     int `yaml:"tls_andshake"`
	ServerProcessing int `yaml:"server_rocessing"`
	ContentTransfer  int `yaml:"content_transfer"`
}

// MatchingRules contains filter to be applied to stored interactions
type MatchingRules struct {
	MatchURL      bool
	IgnoreHeaders []http.Header
}

// RequestInfo contains API request details
type RequestInfo struct {
	URL     string      `yaml:"url"`
	Method  string      `yaml:"method"`
	Payload string      `yaml:"body"`
	Headers http.Header `yaml:"headers"`
}
