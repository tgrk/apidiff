package apidiff

import (
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/dnaeon/go-vcr/recorder"
	"github.com/tcnksm/go-httpstat"
	"gopkg.in/yaml.v2"
)

// APIDiff instance
type APIDiff struct {
	DirectoryPath string
	Options       Options
}

// Options holds shared CLI arguments from user
type Options struct {
	Verbose bool
	Name    string
}

// RecordedSession represents stored API session
type RecordedSession struct {
	Name    string
	Path    string
	Created time.Time
}

// RequestStats hold HTTP stats metrics
type RequestStats struct {
	DNSLookup        int `yaml:"dns_lookup"`
	TCPConnection    int `yaml:"tcp_connection"`
	TLSHandshake     int `yaml:"tls_andshake"`
	ServerProcessing int `yaml:"server_rocessing"`
	ContentTransfer  int `yaml:"content_transfer"`
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
func (ad *APIDiff) Record(name string, ri RequestInfo) error {
	path := path.Join(ad.DirectoryPath, name, ad.getURLHash(ri.URL))

	if ad.Options.Verbose {
		fmt.Printf("Recording %q to \"%s.yaml\"...\n", ri.URL, path)
	}

	r, err := recorder.New(path)
	if err != nil {
		return err
	}
	defer r.Stop()

	// ri.Payload
	req, err := http.NewRequest(strings.ToUpper(ri.Method), ri.URL, nil)
	if err != nil {
		return err
	}

	for headerKey, headerValue := range ri.Headers {
		for _, childHeaderValue := range headerValue {
			req.Header.Set(headerKey, childHeaderValue)
		}
	}

	// collect metrics
	var stats httpstat.Result
	ctx := httpstat.WithHTTPStat(req.Context(), &stats)
	req = req.WithContext(ctx)

	// Create an HTTP client and inject our recorder
	client := &http.Client{
		Transport: r,
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	err = ad.writeRequestStats(name, ri.URL, stats)
	if err != nil {
		return fmt.Errorf("Unable to write request stats due to %s", err)
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

func (ad *APIDiff) writeRequestStats(name, url string, result httpstat.Result) error {
	filename := fmt.Sprintf("%s_stats.yaml", ad.getURLHash(url))
	path := path.Join(ad.DirectoryPath, name, filename)

	stats := RequestStats{
		DNSLookup:        int(result.DNSLookup / time.Millisecond),
		TCPConnection:    int(result.TCPConnection / time.Millisecond),
		TLSHandshake:     int(result.TLSHandshake / time.Millisecond),
		ServerProcessing: int(result.ServerProcessing / time.Millisecond),
		ContentTransfer:  int(result.ContentTransfer(time.Now()) / time.Millisecond),
	}

	output, err := yaml.Marshal(&stats)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path, output, 0644)
	if err != nil {
		return err
	}

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
