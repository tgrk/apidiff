package apidiff

import (
	"bytes"
	"io"

	"gopkg.in/yaml.v2"
)

// Manifest holds all information needed for running
// requests against API
type Manifest struct {
	Version       int             `yaml:"version"`
	MatchingRules []MatchingRules `yaml:"matching_rules"`
	Requests      []RequestInfo   `yaml:"requests"`
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

	err = yaml.Unmarshal(buf.Bytes(), m)
	if err != nil {
		return err
	}
	return nil
}
