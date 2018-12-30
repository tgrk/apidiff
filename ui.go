package apidiff

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/dnaeon/go-vcr/cassette"
	"github.com/olekukonko/tablewriter"
)

// UI contains helpers for drawing CLI interface
type UI struct {
	out io.Writer
}

// NewUI returns instance
func NewUI(w io.Writer) *UI {
	return &UI{
		out: w,
	}
}

// ListSessions draws table of existing recorded sessions
func (ui *UI) ListSessions(sessions []RecordedSession, showCaption bool) {
	rows := [][]string{}
	for _, session := range sessions {
		rows = append(rows, []string{
			session.Name,
			session.Path,
			strconv.Itoa(len(session.Interactions)),
			session.Created.Format("2006-01-02 15:04:05"),
		})
	}

	fmt.Fprintln(ui.out)

	table := tablewriter.NewWriter(ui.out)
	table.SetCenterSeparator("|")
	table.SetHeader([]string{"Name", "Path", "# Interactions", "Created"})
	table.AppendBulk(rows)
	table.SetCaption(showCaption, fmt.Sprintf(" Total %d session(s)", len(rows)))
	table.Render()

	fmt.Fprintln(ui.out)
}

// ShowSession displays detail of selected session
func (ui *UI) ShowSession(session RecordedSession) {
	if len(session.Interactions) == 0 {
		fmt.Fprintf(ui.out, "No recorded session interactions found")
	} else {
		rows := [][]string{}
		for i, interaction := range session.Interactions {
			rows = append(rows, []string{
				strconv.Itoa(i + 1),
				interaction.Method,
				interaction.URL,
				strconv.Itoa(interaction.StatusCode),
				ui.formatMS(interaction.Stats.ServerProcessing),
				ui.formatMS(interaction.Stats.Duration()),
			})
		}

		fmt.Fprintln(ui.out)

		table := tablewriter.NewWriter(ui.out)
		table.SetCenterSeparator("|")
		table.SetHeader([]string{
			"#",
			"Method",
			"URI",
			"Status",
			"Duration",
			"Processing",
		})
		table.SetCaption(true, fmt.Sprintf(" Total %d interaction(s)", len(session.Interactions)))
		table.AppendBulk(rows)
		table.Render()

		fmt.Fprintln(ui.out)
	}
}

// ShowInteractionDetail displays recorded session interaction by given index
func (ui *UI) ShowInteractionDetail(interaction *cassette.Interaction, stats *RequestStats) {
	fmt.Fprintln(ui.out)

	req := interaction.Request
	resp := interaction.Response

	table := tablewriter.NewWriter(ui.out)
	table.SetCenterSeparator("|")
	table.SetAutoMergeCells(true)
	table.SetRowLine(true)
	table.SetHeader([]string{"Type", "Field", "Value"})

	table.AppendBulk([][]string{
		// request
		[]string{"Request", "URL", req.URL},
		[]string{"Request", "Method", req.Method},
		[]string{"Request", "Headers", fmt.Sprintf("%+v", req.Headers)},
		[]string{"Request", "Params", fmt.Sprintf("%+v", req.Form)},
		[]string{"Request", "Payload", ui.formatJSON(req.Body)},

		// response
		[]string{"Response", "Headers", fmt.Sprintf("%+v", resp.Headers)},
		[]string{"Response", "Status", resp.Status},
		[]string{"Response", "Body", ui.formatJSON(resp.Body)},

		// metrics
		[]string{"Metrics", "DNS Lookup", ui.formatMS(stats.DNSLookup)},
		[]string{"Metrics", "TCP Connection", ui.formatMS(stats.TCPConnection)},
		[]string{"Metrics", "TLS Handshake", ui.formatMS(stats.TLSHandshake)},
		[]string{"Metrics", "Server Processing", ui.formatMS(stats.ServerProcessing)},
		[]string{"Metrics", "Content Transfer", ui.formatMS(stats.ContentTransfer)},
		[]string{"Metrics", "Total duration", ui.formatMS(stats.Duration())},
	})
	table.Render()

	fmt.Fprintln(ui.out)
}

// ShowComparisonResults displays result of comparing source and target
// sessions
func (ui *UI) ShowComparisonResults(source RecordedSession, errors map[int]Differences) {
	if len(source.Interactions) == 0 {
		fmt.Fprintf(ui.out, "No recorded session interactions found")
	} else {
		total := 0
		rows := [][]string{}
		for i := range source.Interactions {
			err := errors[i]
			for headerKey, headerValue := range err.Headers {
				rows = append(rows, []string{
					source.Name,
					strconv.Itoa(i),
					fmt.Sprintf("Header %s", headerKey),
					headerValue.Error(),
				})
				total++
			}
			for _, bodyValue := range err.Body {
				rows = append(rows, []string{
					source.Name,
					strconv.Itoa(i),
					"Body",
					bodyValue.Error(),
				})
				total++
			}
		}

		fmt.Fprintln(ui.out)

		table := tablewriter.NewWriter(ui.out)
		table.SetRowLine(true)
		table.SetAutoWrapText(false)
		table.SetCenterSeparator("|")
		table.SetHeader([]string{"Session", "# Interaction", "Type", "Difference"})
		table.SetCaption(true, fmt.Sprintf(" Total %d errors(s)", total))
		table.AppendBulk(rows)
		table.Render()

		fmt.Fprintln(ui.out)
	}
}

func (ui *UI) formatMS(duration int) string {
	return fmt.Sprintf("%d ms", duration)
}

//TODO: pretty printing does not seem to be working correctly
func (ui *UI) formatJSON(body string) string {
	var prettyJSON bytes.Buffer
	json.Indent(&prettyJSON, []byte(body), "", "   ")
	return string(prettyJSON.Bytes())
}
