package apidiff

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"net/http"
	"sort"
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

// Fingerprint returns unique signature of request that
// is used for later comparison
func (ri RequestInteraction) Fingerprint() string {
	h := fnv.New32a()

	var sortedHeaderKeys []string
	for k := range ri.Headers {
		sortedHeaderKeys = append(sortedHeaderKeys, k)
	}
	sort.Strings(sortedHeaderKeys)

	var headers bytes.Buffer
	for _, name := range sortedHeaderKeys {
		for _, value := range ri.Headers[name] {
			headers.WriteString(name)
			headers.WriteString(value)
		}
	}

	fingerprint := fmt.Sprintf(
		"%s%s%d%s%s",
		ri.URL,
		ri.Method,
		ri.StatusCode,
		headers.String(),
		ri.Payload,
	)

	_, err := h.Write([]byte(fingerprint))
	if err != nil {
		panic(err)
	}
	return fmt.Sprint(h.Sum32())
}

// RequestStats hold HTTP stats metrics
type RequestStats struct {
	DNSLookup        int `yaml:"dns_lookup"`
	TCPConnection    int `yaml:"tcp_connection"`
	TLSHandshake     int `yaml:"tls_andshake"`
	ServerProcessing int `yaml:"server_processing"`
	ContentTransfer  int `yaml:"content_transfer"`
}

// Duration returns total time spend on request
func (rs RequestStats) Duration() int {
	return rs.DNSLookup + rs.TCPConnection + rs.TLSHandshake + rs.ServerProcessing + rs.ContentTransfer
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
