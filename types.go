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

// RequestInteraction represents request info for API interaction
type RequestInteraction struct {
	URL        string      `yaml:"url"`
	Method     string      `yaml:"method"`
	StatusCode int         `yaml:"status_code"`
	Headers    http.Header `yaml:"headers"`
	Payload    string      `yaml:"body"`
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
	Name  string      `yaml:"name"`
	Value interface{} `yaml:"value"`
}

// RequestInfo contains shared API request details
type RequestInfo struct {
	Payload string      `yaml:"body"`
	Headers http.Header `yaml:"headers"`
}

// Differences represents errors between two interactions
type Differences struct {
	URL              string
	InteractionIndex int
	Headers          map[string]error
	Body             map[string]error
	Changed          bool
}
