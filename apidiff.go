package apidiff

import (
	"bufio"
	"fmt"
	"hash/fnv"
	"io"
	"io/ioutil"
	"net/http"
	// "net/http/httptrace"
	"net/url"
	"path"
	"time"

	// "github.com/dnaeon/go-vcr/cassette"
	"github.com/dnaeon/go-vcr/recorder"
)

// APIDiff instance
type APIDiff struct {
	DirectoryPath string
	Options       Options
}

// Options holds shared CLI arguments from user
type Options struct {
	Verbose bool
}

// RecordedSession represents stored API session
type RecordedSession struct {
	Name    string
	Path    string
	Created time.Time
}

func (rs RecordedSession) String() string {
	return fmt.Sprintf("Name: %s, Created: %s\n", rs.Name, rs.Created.Format("2006-01-02 15:04:05"))
}

// New creates a new instance
func New(path string, options Options) *APIDiff {
	return &APIDiff{
		DirectoryPath: path,
		Options:       options,
	}
}

// ReadURLs reads URL per line from supplied reader and return
// slice of validated URL or an error
func (ad *APIDiff) ReadURLs(s RecordedSession, r io.Reader) ([]RequestInfo, error) {
	ri := []RequestInfo{}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		ri = append(ri, NewRequest(s, scanner.Text()))
	}

	if err := scanner.Err(); err != nil {
		return ri, err
	}

	return ri, nil
}

// List existing stored API recording sessions
func (ad *APIDiff) List() ([]RecordedSession, error) {
	sessions := []RecordedSession{}
	files, err := ioutil.ReadDir(ad.DirectoryPath)
	if err != nil {
		return sessions, err
	}

	for _, file := range files {
		if file.IsDir() {
			session := RecordedSession{
				Name:    file.Name(),
				Path:    path.Join(ad.DirectoryPath, file.Name()),
				Created: file.ModTime(),
			}
			sessions = append(sessions, session)
		}
	}
	return sessions, nil
}

// Record stores requested URL using casettes into a defined
// directory
func (ad *APIDiff) Record(input RequestInfo) error {
	path := path.Join(ad.DirectoryPath, input.Session.Name, ad.getURLHash(input.URL))

	if ad.Options.Verbose {
		fmt.Printf("Recording %q to \"%s.yaml\"...\n", input.URL, path)
	}

	r, err := recorder.New(path)
	if err != nil {
		return err
	}
	defer r.Stop()

	//TODO: write metrics about request

	// trace := &httptrace.ClientTrace
	// 	GotConn: func(connInfo httptrace.GotConnInfo) {
	// 		fmt.Printf("Got Conn: %+v\n", connInfo)
	// 	},
	// 	DNSDone: func(dnsInfo httptrace.DNSDoneInfo) {
	// 		fmt.Printf("DNS Info: %+v\n", dnsInfo)
	// 	},
	// }

	req, err := http.NewRequest(input.Verb, input.URL, input.Payload)
	if err != nil {
		return err
	}

	for headerKey, headerValue := range input.Headers {
		req.Header.Set(headerKey, headerValue)
	}

	// req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	// Create an HTTP client and inject our recorder
	client := &http.Client{
		Transport: r,
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// Compare compare two stored sessions
func (ad *APIDiff) Compare() error {
	// r, err := recorder.New("fixtures/matchers")
	// if err != nil {
	// 	return err
	// }
	// defer r.Stop()

	// r.SetMatcher(func(r *http.Request, i cassette.Request) bool {
	// 	var b bytes.Buffer
	// 	if _, err := b.ReadFrom(r.Body); err != nil {
	// 		return false
	// 	}
	// 	r.Body = ioutil.NopCloser(&b)
	// 	return cassette.DefaultMatcher(r, i) && (b.String() == "" || b.String() == i.Body)
	// })

	return nil
}

func (ad *APIDiff) isValidURL(strURL string) bool {
	uri, err := url.Parse(strURL)
	if err != nil || (uri.Scheme != "http" && uri.Scheme != "https") {
		return false
	}
	return true
}

func (ad *APIDiff) getURLHash(url string) string {
	h := fnv.New32a()
	h.Write([]byte(url))
	return fmt.Sprint(h.Sum32())
}
