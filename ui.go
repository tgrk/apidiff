package apidiff

import (
	"fmt"
	"os"
	"strconv"

	"github.com/olekukonko/tablewriter"
)

// UI contains helpers for drawing CLI interface
type UI struct {
}

// NewUI returns instance
func NewUI() *UI {
	return &UI{}
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

	fmt.Println()

	table := tablewriter.NewWriter(os.Stdout)
	table.SetCenterSeparator("|")
	table.SetHeader([]string{"Name", "Path", "# Interactions", "Created"})
	table.AppendBulk(rows)
	table.SetCaption(showCaption, fmt.Sprintf(" Total %d session(s)", len(rows)))
	table.Render()

	fmt.Println()
}

// ShowSessionDetail displays detail of selected session
func (ui *UI) ShowSessionDetail(session RecordedSession) {
	if len(session.Interactions) == 0 {
		fmt.Println("| No recorded session interactions found")
	} else {
		rows := [][]string{}
		for _, interaction := range session.Interactions {
			rows = append(rows, []string{
				interaction.Method,
				strconv.Itoa(interaction.StatusCode),
				fmt.Sprintf("%d ms", interaction.Stats.ServerProcessing),
				interaction.URL,
			})
		}

		fmt.Println()

		table := tablewriter.NewWriter(os.Stdout)
		table.SetCenterSeparator("|")
		table.SetHeader([]string{"Method", "Status", "Duration", "URI"})
		table.SetCaption(true, fmt.Sprintf(" Total %d interaction(s)", len(session.Interactions)))
		table.AppendBulk(rows)
		table.Render()

		fmt.Println()
	}
}

// ShowComparisonResults displays result of comparing source and target
// sessions
func (ui *UI) ShowComparisonResults(source RecordedSession, errors map[int]Differences) {
	if len(source.Interactions) == 0 {
		fmt.Println("| No recorded session interactions found")
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

		fmt.Println()

		table := tablewriter.NewWriter(os.Stdout)
		table.SetRowLine(true)
		table.SetAutoWrapText(false)
		table.SetCenterSeparator("|")
		table.SetHeader([]string{"Session", "# Interaction", "Type", "Difference"})
		table.SetCaption(true, fmt.Sprintf(" Total %d errors(s)", total))
		table.AppendBulk(rows)
		table.Render()

		fmt.Println()
	}
}
