package apidiff

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
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
			paths, _ := ad.listInteractions(session.Path)

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

// Record stores requested URL using casettes into a defined
// directory
func (ad *APIDiff) Record(name string, ri RequestInfo, rules []MatchingRules) error {
	path := path.Join(ad.getPath(name), ad.getURLHash(ri.URL))

	if ad.Options.Verbose {
		fmt.Printf("Recording %q to \"%s.yaml\"...\n", ri.URL, path)
	}

	r, err := recorder.New(path)
	if err != nil {
		return err
	}
	defer r.Stop()

	fmt.Printf("DEBUG: recorder=%+v\n", r)

	// ri.Payload
	req, err := http.NewRequest(strings.ToUpper(ri.Method), ri.URL, nil)
	if err != nil {
		return err
	}

	if len(rules) > 0 {
		r.SetMatcher(func(r *http.Request, c cassette.Request) bool {
			var b bytes.Buffer
			if _, err := b.ReadFrom(r.Body); err != nil {
				return false
			}
			r.Body = ioutil.NopCloser(&b)

			var matching = true

			//TODO: apply excludes
			// for _, header := range r.Header {

			// for _, header := range rules.IgnoreHeaders {
			// 	for _, header := range filter.Headers {
			// 		//TODO: compare headers
			// 	}
			// }
			fmt.Printf("DEBUG: req=%+v\n", r)
			fmt.Printf("DEBUG: casette=%+v\n", c)

			sourceBody := b.Bytes()
			targetBody := []byte(c.Body)

			// compare body using JSON diff
			jd := gojsondiff.New()
			diff, err := jd.Compare(sourceBody, targetBody)
			if err != nil {
				fmt.Printf("DEBUG: diff=%+v\n", err)
				//TODO: what about using channel to pass errors back?
				//errors[ad.getURLHash(r.URL)] = err
				matching = false
			}
			if diff.Modified() {
				fmt.Printf("DEBUG: deltas=%+v\n", diff.Deltas())
				matching = false
			}

			// if rules[0].MatchURL {
			// 	matching = cassette.DefaultMatcher(r, i)
			// }

			return matching
		})
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
	defer resp.Body.Close()

	err = ad.writeRequestStats(path, ri.URL, stats)
	if err != nil {
		return fmt.Errorf("Unable to write request stats - %s", err)
	}

	return nil
}

// Compare compare two stored sessions
func (ad *APIDiff) Compare(source RecordedSession, target Manifest) map[int]error {

	//TODO: load interactions
	//TODO: match on source and target urls or index?
	// source.Path
	fmt.Printf("DEBUG: source=%+v\n", source)
	fmt.Printf("DEBUG: target=%+v\n", target)

	var errors = make(map[int]error, 0)

	for _, request := range target.Requests {
		// ri := source.Interactions[i]

		rules := target.MatchingRules
		fmt.Printf("DEBUG: rules=%+v\n", rules)

		//TODO: verbose
		err := ad.Record(source.Name, request, rules)
		if err != nil {
			//TODO: logging
		}

	}

	return errors
}

func (ad *APIDiff) compareInteraction(source, target RecordedInteraction) {

}

// Delete an existing recorded session; otherwise returns error
func (ad *APIDiff) Delete(name string) error {
	path := ad.getPath(name)
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
	file, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

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

func (ad *APIDiff) getPath(name string) string {
	return path.Join(ad.DirectoryPath, name)
}

func (ad *APIDiff) getURLHash(url string) string {
	h := fnv.New32a()
	h.Write([]byte(url))
	return fmt.Sprint(h.Sum32())
}
