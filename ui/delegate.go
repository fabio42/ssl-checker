package ui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/fabio42/ssl-checker/domains"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type listKeyMap struct {
	toggleDetails key.Binding
	toggleExport  key.Binding
}

func newListKeyMap() *listKeyMap {
	return &listKeyMap{
		toggleDetails: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "toggle domain details"),
		),
		toggleExport: key.NewBinding(
			key.WithKeys("E"),
			key.WithHelp("E", "export results to file"),
		),
	}
}

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	var (
		title, details string
		leftPadding    = 2
		entry          = lipgloss.NewStyle().Width(m.Width()).PaddingLeft(leftPadding)
	)

	if i, ok := listItem.(domains.Response); ok {
		title = i.Title()
		details = Details(i)
	} else {
		return
	}

	spacer := m.Width() - lipgloss.Width(title) - leftPadding
	detailsTrunc := truncDetails(details, &spacer, 1)

	if index == m.Index() {
		entry = entry.
			Border(lipgloss.NormalBorder(), false, false, false, true).
			PaddingLeft(1).Foreground(lipgloss.Color("170"))
	}

	fmt.Fprint(w, entry.Render(title+strings.Repeat(" ", spacer)+detailsTrunc))
}

// truncDetails truncate details when screen is too small
func truncDetails(str string, width *int, padding int) string {
	runes := []rune(str)
	initialLenStr := len(runes)
	for *width-lipgloss.Width(string(runes)) < padding {
		if index := len(runes) - 1; index > 0 {
			runes = runes[:index]
		} else {
			break
		}
	}
	if initialLenStr > len(runes) {
		runes[0] = '…'
	}

	final := string(runes)

	*width -= lipgloss.Width(final)
	if *width < 0 {
		*width = 0
	}
	return final
}

func Details(i domains.Response) string {
	var (
		red    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
		orange = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
		green  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	)

	if i.Error == nil {
		now := time.Now()
		inOneMonth := now.AddDate(0, 1, 0)
		inFourMonth := now.AddDate(0, 4, 0)
		var dateOutput string
		if i.NotAfter.Before(inOneMonth) {
			dateOutput = red.Render(i.NotAfter.Format("2006-01-02"))
		} else if i.NotAfter.Before(inFourMonth) {
			dateOutput = orange.Render(i.NotAfter.Format("2006-01-02"))
		} else {
			dateOutput = green.Render(i.NotAfter.Format("2006-01-02"))
		}
		return fmt.Sprintf("%v | %v", i.Issuer.CommonName, dateOutput)
	} else {
		return fmt.Sprintf("%s %v", orange.Render("Error:"), red.Render(i.KnownError()))
	}
}
