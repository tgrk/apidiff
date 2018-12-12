package apidiff

import (
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/dnaeon/go-vcr/cassette"
	"github.com/dnaeon/go-vcr/recorder"
	"github.com/tcnksm/go-httpstat"
	"github.com/yudai/gojsondiff"
	"gopkg.in/yaml.v2"
)

// APIDiff instance
type APIDiff struct {
	DirectoryPath string
	Options       Options
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

			// iterates over saved interactions
			paths, err := ad.listInteractions(session.Path)
			if err != nil {
				return sessions, err
			}

			for _, p := range paths {
				interaction, err := ad.loadInteraction(p)
				if err != nil {
					continue
				}
				session.Interactions = append(session.Interactions, interaction)
			}
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

// Show returns an existing recorded session otherwise an error
func (ad *APIDiff) Show(name string) (RecordedSession, error) {
	session := RecordedSession{}

	files, err := ioutil.ReadDir(ad.DirectoryPath)
	if err != nil {
		return session, err
	}

	var found = false
	for _, file := range files {
		if file.IsDir() && name == file.Name() {
			sessionPath := path.Join(ad.DirectoryPath, name)
			session = RecordedSession{
				Name:    file.Name(),
				Path:    sessionPath,
				Created: file.ModTime(),
			}

			paths, err := ad.listInteractions(sessionPath)
			if err != nil {
				return session, err
			}

			// iterates over saved interactions
			for _, p := range paths {
				interaction, err := ad.loadInteraction(p)
				if err != nil {
					continue
				}
				session.Interactions = append(session.Interactions, interaction)
			}
			found = true
			break
		}
	}

	if !found {
		return session, fmt.Errorf("Unable to find session %q", name)
	}

	return session, nil
}

// Record stores requested URL using casettes into a defined directory
func (ad *APIDiff) Record(dir, name string, ri RequestInfo, rules []MatchingRules) error {
	path := path.Join(ad.getPath(dir, name), ad.getURLHash(ri.URL))

	if ad.Options.Verbose {
		fmt.Printf("Recording %q to \"%s.yaml\"...\n", ri.URL, path)
	}

	r, err := recorder.New(path)
	if err != nil {
		return err
	}
	defer func() {
		if err = r.Stop(); err != nil {
			panic(err)
		}
	}()

	// custom request matcher based on specified rules
	r.SetMatcher(func(r *http.Request, cr cassette.Request) bool {
		if len(rules) > 0 {
			for _, rule := range rules {
				if rule.Name == "match_url" {
					return rule.Value.(bool)
				}
			}
		}

		return cassette.DefaultMatcher(r, cr)
	})

	// custom filter for stored request data
	r.AddFilter(func(ci *cassette.Interaction) error {
		if len(rules) > 0 {
			for _, rule := range rules {
				if rule.Name == "ignore_headers" {
					for _, headerKey := range rule.Value.([]interface{}) {
						delete(ci.Request.Headers, headerKey.(string))
					}
				}
			}
		}
		return nil
	})

	//TODO: ri.Payload
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
	defer func() {
		if err = resp.Body.Close(); err != nil {
			panic(err)
		}
	}()

	err = ad.writeRequestStats(path, ri.URL, stats)
	if err != nil {
		return fmt.Errorf("Unable to write request stats - %s", err)
	}

	return nil
}

// Compare compare stored session against a manifest
func (ad *APIDiff) Compare(source RecordedSession, target Manifest) (map[int][]error, error) {
	var results = make(map[int][]error)
	rules := target.MatchingRules

	// create temp location for target cassettes
	tcDir, err := ioutil.TempDir("/tmp", "apidifftest")
	if err != nil {
		return results, err
	}

	scPath := ad.getPath(ad.DirectoryPath, source.Name)

	for i, tr := range target.Requests {
		// record target into temporary location
		err := ad.Record(tcDir, source.Name, tr, rules)
		if err != nil {
			return results, err
		}

		// wait for cassette untill it is store in FS
		targetCassettePath := path.Join(tcDir, source.Name, ad.getURLHash(tr.URL))
		if err = ad.waitForFile(fmt.Sprintf("%s.yaml", targetCassettePath), 0); err != nil {
			return results, err
		}

		// load target cassette
		tc, err := cassette.Load(targetCassettePath)
		if err != nil {
			return results, err
		}

		// load source cassette
		si := source.Interactions[i]
		sc, err := cassette.Load(path.Join(scPath, ad.getURLHash(si.URL)))
		if err != nil {
			return results, err
		}

		// do comparison and collect errors
		results[i] = ad.compareInteractions(rules, *sc.Interactions[0], *tc.Interactions[0])
	}

	// finally cleanup all temporary resources
	if err = os.RemoveAll(tcDir); err != nil {
		return results, nil
	}

	return results, nil
}

// Delete an existing recorded session; otherwise returns error
func (ad *APIDiff) Delete(name string) error {
	path := ad.getPath(ad.DirectoryPath, name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return err
	}

	if ad.Options.Verbose {
		fmt.Printf("Recorded session %q was removed...\n", name)
	}

	return os.RemoveAll(path)
}

func (ad *APIDiff) writeRequestStats(path, url string, result httpstat.Result) error {
	dirpath := filepath.Dir(path)
	if _, err := os.Stat(dirpath); os.IsNotExist(err) {
		if err := os.MkdirAll(dirpath, os.ModePerm); err != nil {
			return err
		}
	}

	filepath := fmt.Sprintf("%s_stats.yaml", path)
	file, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer func() {
		if err = file.Close(); err != nil {
			panic(err)
		}
	}()

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

	_, err = file.Write(output)
	if err != nil {
		return err
	}

	if ad.Options.Verbose {
		fmt.Printf("Writing request metrics for %q into %q\"...\n", url, filepath)
	}
	return nil
}

func (ad *APIDiff) compareInteractions(rules []MatchingRules, source cassette.Interaction, target cassette.Interaction) []error {
	errors := make([]error, 0)

	// basic request comparison
	sr := source.Request
	tr := target.Request

	// compare headers
	for sk, sv := range sr.Headers {
		tv, found := tr.Headers[sk]
		if !found {
			errors = append(errors, fmt.Errorf("header %q is missing", sk))
		}
		if !reflect.DeepEqual(sv, tv) {
			errors = append(errors, fmt.Errorf("header %q value should be %v but got %v", sk, sv, tv))
		}
	}

	// compare body using JSON diff
	jd := gojsondiff.New()
	diff, err := jd.Compare([]byte(sr.Body), []byte(tr.Body))
	if err != nil {
		return errors
	}

	if diff.Modified() {
		//TODO: serialize deltas into errors?
		fmt.Printf("DEBUG: deltas=%+v\n", diff.Deltas())
	}

	return errors
}

func (ad *APIDiff) isValidURL(strURL string) bool {
	uri, err := url.Parse(strURL)
	if err != nil || (uri.Scheme != "http" && uri.Scheme != "https") {
		return false
	}
	return true
}

func (ad *APIDiff) listInteractions(basePath string) ([]string, error) {
	paths := []string{}

	files, err := ioutil.ReadDir(basePath)
	if err != nil {
		return paths, err
	}

	for _, file := range files {
		if !file.IsDir() && !strings.HasSuffix(file.Name(), "_stats.yaml") {
			paths = append(paths, path.Join(basePath, file.Name()))
		}
	}
	return paths, err
}

func (ad *APIDiff) loadInteraction(path string) (RecordedInteraction, error) {
	interaction := RecordedInteraction{}

	// parse interactions
	c, err := ad.loadCassette(path)
	if err != nil {
		return interaction, err
	}

	// parse interaction stats
	stats, err := ad.loadRequestStats(path)
	if err != nil {
		return interaction, nil
	}

	interaction = RecordedInteraction{
		URL:        c.Request.URL,
		Method:     c.Request.Method,
		StatusCode: c.Response.Code,
		Stats:      *stats,
	}

	return interaction, nil
}

func (ad *APIDiff) loadCassette(path string) (*cassette.Interaction, error) {
	c, err := cassette.Load(strings.Replace(path, ".yaml", "", 1))
	if err != nil {
		return nil, err
	}
	return c.Interactions[0], nil
}

func (ad *APIDiff) loadRequestStats(path string) (*RequestStats, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	stats := &RequestStats{}
	err = yaml.Unmarshal(data, stats)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

func (ad *APIDiff) waitForFile(path string, retry int) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if retry <= 50 {
			time.Sleep(100 * time.Millisecond)
			return ad.waitForFile(path, retry+1)
		}
		return fmt.Errorf("unable to read file %q within time", path)
	}
	return nil
}

func (ad *APIDiff) getPath(dir, name string) string {
	return path.Join(dir, name)
}

func (ad *APIDiff) getURLHash(url string) string {
	h := fnv.New32a()
	_, err := h.Write([]byte(url))
	if err != nil {
		panic(err)
	}
	return fmt.Sprint(h.Sum32())
}
