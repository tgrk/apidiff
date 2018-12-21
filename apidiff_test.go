package apidiff

import (
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

func TestSessionManagement(t *testing.T) {
	path, err := makeTempStorageDirectory()
	if err != nil {
		panic(err)
	}
	defer removeTempStorageDirectory(path)

	ad := New(path, Options{Verbose: true})
	manifest := readExampleManifest("constant.yaml", t)

	// record session based on example
	for _, interaction := range manifest.Interactions {
		err = ad.Record(
			path,
			sessionName,
			interaction,
			manifest.Request,
			manifest.MatchingRules,
		)
		if err != nil {
			panic(err)
		}
	}

	// list created session
	sessions, err := ad.List()
	if err != nil {
		panic(err)
	}

	if len(sessions) == 0 {
		t.Fatalf("Expected to have 1 recorded session but got %d", len(sessions))
	}

	// show an existing session
	session, err := ad.Show(sessionName)
	if err != nil {
		panic(err)
	}

	if session.Created != sessions[0].Created {
		t.Error("Expected to have same sessions but got different")
	}
	if session.Name != sessionName {
		t.Errorf("Expect to have session name %q but got %q", sessionName, session.Name)
	}

	if len(session.Interactions) == 0 {
		t.Fatalf("Expect to have a recorded interaction but got none")
	}

	interaction := session.Interactions[0]
	expectedRequest := manifest.Interactions[0]
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

	// delete existing session
	err = ad.Delete(sessionName)
	if err != nil {
		panic(err)
	}

	// list created session
	noSessions, err := ad.List()
	if err != nil {
		panic(err)
	}

	if len(noSessions) != 0 {
		t.Errorf("Expected %q to be empty after deletion", path)
	}
}

func TestCompareSameSession(t *testing.T) {
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
		t.Errorf("Expect to have no session but got %d", len(sessions))
	}

	manifest := readExampleManifest("constant.yaml", t)

	for _, interaction := range manifest.Interactions {
		err = ad.Record(
			path,
			sessionName,
			interaction,
			manifest.Request,
			manifest.MatchingRules,
		)
		if err != nil {
			panic(err)
		}
	}
	session, err := ad.Show(sessionName)
	if err != nil {
		panic(err)
	}

	differences, err := ad.Compare(session, *manifest)
	if err != nil {
		panic(err)
	}

	if len(differences[0].Headers) != 0 && len(differences[0].Body) != 0 {
		t.Error("Expect to have no differences for same manifest")
	}
}

func TestCompareDifferentSession(t *testing.T) {
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
		t.Errorf("Expect to have no session but got %d", len(sessions))
	}

	manifest := readExampleManifest("different.yaml", t)

	for _, interaction := range manifest.Interactions {
		err = ad.Record(
			path,
			sessionName,
			interaction,
			manifest.Request,
			manifest.MatchingRules,
		)
		if err != nil {
			panic(err)
		}
	}
	session, err := ad.Show(sessionName)
	if err != nil {
		panic(err)
	}

	differences, err := ad.Compare(session, *manifest)
	if err != nil {
		panic(err)
	}

	if len(differences[0].Headers) == 0 {
		t.Error("Expect to have different HTTP ETag header but got same")
	}
	if len(differences[0].Body) == 0 {
		t.Error("Expect to have different JSON payload but got same")
	}
}

func TestManifestParsing(t *testing.T) {
	manifest := readExampleManifest("constant.yaml", t)

	expected := Manifest{
		Version: 1,
		Interactions: []RequestInteraction{
			RequestInteraction{
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

func readExampleManifest(filename string, t *testing.T) *Manifest {
	path := path.Join("examples", filename)

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
