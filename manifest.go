package apidiff

import (
	"bytes"
	"io"

	"gopkg.in/yaml.v2"
)

// Manifest holds all information needed for running
// requests against API
type Manifest struct {
	Version       int                  `yaml:"version"`
	MatchingRules []MatchingRules      `yaml:"matching_rules"`
	Request       RequestInfo          `yaml:"request"`
	Interactions  []RequestInteraction `yaml:"interactions"`
}

// NewManifest creates an empty manifest
func NewManifest() *Manifest {
	return &Manifest{}
}

// Parse YAML document
func (m *Manifest) Parse(r io.Reader) error {
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(r)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(buf.Bytes(), m)
}
