package apidiff

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	"github.com/yudai/gojsondiff/formatter"
	"gopkg.in/yaml.v2"
)

var formatterConfig = formatter.AsciiFormatterConfig{
	ShowArrayIndex: true,
	Coloring:       true,
}

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

// Detail returns interaction from recorded session given
// its name and index
func (ad *APIDiff) Detail(name string, interactionIndex int) (*cassette.Interaction, *RequestStats, error) {
	sessionPath := path.Join(ad.DirectoryPath, name)
	paths, err := ad.listInteractions(sessionPath)
	if err != nil {
		return nil, nil, err
	}

	var result *cassette.Interaction
	var stats RequestStats
	found := false
	i := 0
	for _, path := range paths {
		if i == interactionIndex-1 {
			// parse interactions
			result, err = ad.loadCassette(path)
			if err != nil {
				return nil, nil, err
			}

			// parse interaction stats
			stats, err = ad.loadRequestStats(path)
			if err != nil {
				return nil, nil, err
			}
			found = true
			break
		}
		i++
	}

	if !found {
		return nil, nil, fmt.Errorf("unable to find interaction with index %d", interactionIndex)
	}

	return result, &stats, nil
}

// Record stores requested URL using casettes into a defined directory
func (ad *APIDiff) Record(dir, name string, interaction RequestInteraction, ri RequestInfo, rules []MatchingRules) error {
	url := interaction.URL
	method := strings.ToUpper(interaction.Method)
	path := path.Join(ad.getPath(dir, name), interaction.Fingerprint())

	if ad.Options.Verbose {
		fmt.Printf("Recording %s %q into \"%s.yaml\"...\n", method, url, path)
	}

	r, err := ad.createRecorder(path, rules)
	if err != nil {
		return err
	}
	defer func() {
		if err = r.Stop(); err != nil {
			panic(err)
		}
	}()

	// create request from manifest defintion
	var payload io.Reader
	if ri.Payload != "" {
		payload = strings.NewReader(ri.Payload)
	}

	// send interaction specific payload
	if interaction.Payload != "" {
		payload = strings.NewReader(interaction.Payload)
	}

	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		return err
	}

	//TODO: apply common and interaction specific HTTP headers
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

	err = ad.writeRequestStats(path, stats)
	if err != nil {
		return fmt.Errorf("Unable to write request stats - %s", err)
	}

	if ad.Options.Verbose {
		fmt.Printf("Request finished with status: %s\n", resp.Status)
		fmt.Println("---")
	}

	return nil
}

// Compare compare stored session against a manifest
func (ad *APIDiff) Compare(source RecordedSession, target Manifest) (map[int]Differences, error) {
	var results = make(map[int]Differences)
	rules := target.MatchingRules

	// create temp location for target cassettes
	tcDir, err := ioutil.TempDir("/tmp", "apidifftest")
	if err != nil {
		return results, err
	}

	scPath := ad.getPath(ad.DirectoryPath, source.Name)

	for i, interaction := range target.Interactions {
		// record target into temporary location
		err = ad.Record(
			tcDir,
			source.Name,
			interaction,
			target.Request,
			rules,
		)
		if err != nil {
			return results, err
		}

		// wait for cassette until it is store in FS
		targetCassettePath := path.Join(
			tcDir,
			source.Name,
			interaction.Fingerprint(),
		)
		if err = ad.waitForFile(fmt.Sprintf("%s.yaml", targetCassettePath), 0); err != nil {
			return results, err
		}

		// load target cassette
		tc, err := cassette.Load(targetCassettePath)
		if err != nil {
			return results, err
		}

		// load source cassette
		if len(source.Interactions) > i {
			sc, err := cassette.Load(
				path.Join(scPath, interaction.Fingerprint()),
			)
			if err != nil {
				return results, err
			}

			// do comparison and collect errors
			result, err := ad.compareInteractions(
				i,
				rules,
				*sc.Interactions[0],
				*tc.Interactions[0],
			)
			if err != nil {
				return results, err
			}
			results[i] = result
		}
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

func (ad *APIDiff) writeRequestStats(path string, result httpstat.Result) error {
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
		fmt.Printf("Writing request metrics into %q...\n", filepath)
	}
	return nil
}

func (ad *APIDiff) compareInteractions(idx int, rules []MatchingRules, source cassette.Interaction, target cassette.Interaction) (Differences, error) {
	result := Differences{
		InteractionIndex: idx,
		Headers:          make(map[string]error),
		Body:             make(map[string]error),
	}

	// basic response comparison
	sr := source.Response
	tr := target.Response

	// header ignore rules
	ignoreHeaders := make(map[string]bool)
	for _, rule := range rules {
		if rule.Name == "ignore_headers" {
			for _, headerKey := range rule.Value.([]interface{}) {
				ignoreHeaders[headerKey.(string)] = true
			}
			break
		}
	}

	// compare headers
	for sk, sv := range sr.Headers {
		// skip excluded headers
		if _, found := ignoreHeaders[sk]; found {
			continue
		}

		tv, found := tr.Headers[sk]
		if !found {
			result.Headers[sk] = errors.New("header is missing")
			result.Changed = true
		}
		if !reflect.DeepEqual(sv, tv) {
			result.Headers[sk] = fmt.Errorf("expect %v but got %v", sv, tv)
			result.Changed = true
		}
	}

	// compare body using JSON diff
	jd := gojsondiff.New()
	diff, err := jd.Compare([]byte(sr.Body), []byte(tr.Body))
	if err != nil {
		return result, err
	}

	if diff.Modified() {
		// get source JSON for showing difference
		var diffJSON map[string]interface{}
		err := json.Unmarshal([]byte(sr.Body), &diffJSON)
		if err != nil {
			return result, err
		}

		formatter := formatter.NewAsciiFormatter(diffJSON, formatterConfig)
		diffString, err := formatter.Format(diff)
		if err != nil {
			return result, err
		}
		result.Body["payload"] = fmt.Errorf("%s", diffString)
		result.Changed = true
	}

	return result, nil
}

func (ad *APIDiff) createRecorder(path string, rules []MatchingRules) (*recorder.Recorder, error) {
	r, err := recorder.New(path)
	if err != nil {
		return r, err
	}

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

	return r, err
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
		Stats:      stats,
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

func (ad *APIDiff) loadRequestStats(path string) (RequestStats, error) {
	stats := RequestStats{}

	path = strings.Replace(path, ".yaml", "_stats.yaml", -1)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return stats, err
	}

	err = yaml.Unmarshal(data, &stats)
	if err != nil {
		return stats, err
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
