package apidiff

import (
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"testing"
)

const (
	sessionName = "foo"
)

func TestListCommand(t *testing.T) {
	path, err := makeTempStorageDirectory()
	if err != nil {
		t.Fatal(err)
	}

	ad := New(path, Options{})
	sessions, err := ad.List()
	if err != nil {
		t.Fatal(err)
	}

	if len(sessions) != 0 {
		t.Errorf("Expected %q to be empty", path)
	}
}

func TestRecordCommand(t *testing.T) {
	path, err := makeTempStorageDirectory()
	if err != nil {
		t.Fatal(err)
	}
	defer removeTempStorageDirectory(path)

	ad := New(path, Options{Verbose: true})
	manifest := readExampleManifest(t)

	for _, request := range manifest.Requests {
		err = ad.Record(sessionName, request)
		if err != nil {
			t.Error(err)
		}
	}

	sessions, err := ad.List()
	if err != nil {
		t.Error(err)
	}

	if len(sessions) == 0 {
		t.Errorf("Expected to have 1 recorded session but got %d", len(sessions))
	}

	//TODO: test exact session was recorded and has interactions
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
	path := path.Join("example", "simple.yaml")

	reader, err := os.Open(path)
	if err != nil {
		t.Error(err)
	}
	defer reader.Close()

	manifest := NewManifest()
	if err := manifest.Parse(reader); err != nil {
		t.Error(err)
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
