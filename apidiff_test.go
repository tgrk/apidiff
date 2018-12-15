package apidiff

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"
)

const (
	sessionName = "foo"
)

func TestListCommand(t *testing.T) {
	path, err := makeTempStorageDirectory()
	if err != nil {
		panic(err)
	}

	ad := New(path, Options{})
	sessions, err := ad.List()
	if err != nil {
		panic(err)
	}

	if len(sessions) != 0 {
		t.Errorf("Expected %q to be empty", path)
	}
}

func TestRecordCommand(t *testing.T) {
	path, err := makeTempStorageDirectory()
	if err != nil {
		panic(err)
	}
	defer removeTempStorageDirectory(path)

	ad := New(path, Options{Verbose: true})
	manifest := readExampleManifest(t)

	for _, request := range manifest.Requests {
		err = ad.Record(path, sessionName, request, []MatchingRules{})
		if err != nil {
			panic(err)
		}
	}

	sessions, err := ad.List()
	if err != nil {
		panic(err)
	}

	if len(sessions) == 0 {
		t.Fatalf("Expected to have 1 recorded session but got %d", len(sessions))
	}

	session := sessions[0]
	if session.Name != sessionName {
		t.Errorf("Expect to have session name %q but got %q", sessionName, session.Name)
	}

	if len(session.Interactions) == 0 {
		t.Fatalf("Expect to have a recorded interaction but got none")
	}

	fmt.Printf("DEBUG: session=%+v\n", session)

	interaction := session.Interactions[0]
	expectedRequest := manifest.Requests[0]
	if expectedRequest.URL != interaction.URL {
		t.Errorf("Expected to record URL %q but got %q",
			expectedRequest.URL,
			interaction.URL,
		)
	}
	if strings.ToLower(expectedRequest.Method) != strings.ToLower(interaction.Method) {
		t.Errorf("Expected to record %q request but got %q",
			expectedRequest.Method,
			interaction.Method,
		)
	}
}

func TestManifestParsing(t *testing.T) {
	manifest := readExampleManifest(t)

	expected := Manifest{
		Version: 1,
		Requests: []RequestInfo{
			RequestInfo{
				URL:    "https://api.chucknorris.io/jokes/random",
				Method: "get",
				Headers: map[string][]string{
					"Content-Type": []string{"application/json; charset=utf-8"},
				},
			},
		},
	}

	if reflect.DeepEqual(expected, manifest) {
		t.Errorf("Expect manifest to be equal to %+v got %+v", expected, manifest)
	}
}

func TestIsValidURL(t *testing.T) {
	urls := []string{
		"http://www.example.com",
		"https://api.example.com/foo",
		"ftp://somewhere",
	}

	expected := []bool{
		true, true, false,
	}

	ad := New("", Options{})
	for i, url := range urls {
		if expected[i] != ad.isValidURL(url) {
			t.Errorf("Expected %q to be invalid", url)
		}
	}
}

func readExampleManifest(t *testing.T) *Manifest {
	path := path.Join("examples", "simple.yaml")

	reader, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer reader.Close()

	manifest := NewManifest()
	if err := manifest.Parse(reader); err != nil {
		panic(err)
	}

	return manifest
}

func makeTempStorageDirectory() (string, error) {
	dir, err := ioutil.TempDir("/tmp", "apidifftest")
	if err != nil {
		return "", err
	}
	return dir, nil
}

func removeTempStorageDirectory(path string) error {
	return os.RemoveAll(path)
}
