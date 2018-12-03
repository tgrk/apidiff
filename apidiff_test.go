package apidiff

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
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

func TestReadingURLsFromFile(t *testing.T) {
	path, err := makeTempStorageDirectory()
	if err != nil {
		t.Fatal(err)
	}
	defer removeTempStorageDirectory(path)

	ad := New(path, Options{})

	urls := getURLs()
	sourceFilePath, err := createFileWithURLs(urls)
	if err != nil {
		t.Error(err)
	}

	reader, err := os.Open(sourceFilePath)
	if err != nil {
		t.Error(err)
	}
	defer reader.Close()

	s := RecordedSession{
		Name: "foo",
	}

	result, err := ad.ReadURLs(s, reader)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(urls, result) {
		t.Error("Expected to read the same urls from file")
	}
}

func TestRecord(t *testing.T) {
	path, err := makeTempStorageDirectory()
	if err != nil {
		t.Fatal(err)
	}
	//defer removeTempStorageDirectory(path)

	ad := New(path, Options{Verbose: true})

	urls := getURLs()
	if len(urls) == 0 {
		t.Error("Missing URLs for recording")
	}

	s := RecordedSession{
		Name: "foo",
	}
	ri := NewRequest(s, urls[0])
	err = ad.Record(ru)
	if err != nil {
		t.Error(err)
	}

	sessions, err := ad.List()
	if err != nil {
		t.Error(err)
	}

	fmt.Printf("DEBUG: sessions=%+v\n", sessions)
}

func TestIsValidURL(t *testing.T) {
	urls := getURLs()
	if len(urls) == 0 {
		t.Error("Missing URLs for recording")
	}
	urls = append(urls, "ftp://invalid")

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

func createFileWithURLs(urls []string) (string, error) {
	file, err := ioutil.TempFile("/tmp", "testapi")
	if err != nil {
		return "", err
	}

	writer := bufio.NewWriter(file)
	defer file.Close()

	for _, url := range urls {
		fmt.Fprintln(writer, url)
	}
	writer.Flush()

	return file.Name(), nil
}

func getURLs() []string {
	return []string{
		"https://api.chucknorris.io/jokes/random",
		"https://jsonplaceholder.typicode.com/posts",
	}
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
