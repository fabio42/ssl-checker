package ui

import (
	"fmt"
	"strings"

	"github.com/fabio42/ssl-checker/domains"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/rs/zerolog/log"
)

type domainDetails struct {
	keys     detailsKeyMap
	help     help.Model
	viewport *viewport.Model
	renderer *glamour.TermRenderer
}

func newDetails() *domainDetails {
	vp := viewport.New(10, 10)
	vp.Style = lipgloss.NewStyle().
		Align(lipgloss.Center, lipgloss.Center).
		BorderStyle(lipgloss.RoundedBorder()).
		PaddingRight(2).
		BorderForeground(lipgloss.Color("62"))

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		log.Fatal().Err(err)
	}

	str, err := renderer.Render("Something went wrong")
	if err != nil {
		log.Fatal().Err(err)
	}

	vp.SetContent(str)

	return &domainDetails{
		keys:     setDetailsKeyMap(),
		help:     help.New(),
		viewport: &vp,
		renderer: renderer,
	}
}

func (m domainDetails) helpView() string {
	h := m.help.View(m.keys)
	return helpStyle(h)
}

func (m domainDetails) setData(i domains.Response) {
	var details strings.Builder
	details.WriteString(fmt.Sprintf("# %v\n", i.Domain))
	details.WriteString("\n")
	if i.Error != nil {
		details.WriteString(fmt.Sprintf("- Error       : %v\n", i.Error))
	} else {
		details.WriteString("## Issuer")
		details.WriteString("\n")
		details.WriteString(fmt.Sprintf("- Organization: %s\n", i.Issuer.Organization[0]))
		details.WriteString(fmt.Sprintf("- Common Name : %s\n", i.Issuer.CommonName))
		details.WriteString(fmt.Sprintf("- Country     : %s\n", i.Issuer.Country[0]))
		details.WriteString("## Validity")
		details.WriteString("\n")
		details.WriteString(fmt.Sprintf("- Not before: %v\n", i.NotBefore))
		details.WriteString(fmt.Sprintf("- Not after : %v\n", i.NotAfter))
		details.WriteString("## Certificate Details:")
		details.WriteString("\n")
		details.WriteString(fmt.Sprintf("- Common Name : %v\n", i.Subject.CommonName))
		if len(i.Subject.Organization) > 0 {
			details.WriteString(fmt.Sprintf("- Organization: %s\n", i.Subject.Organization[0]))
		}
		if len(i.Subject.Locality) > 0 {
			details.WriteString(fmt.Sprintf("- Locatity    : %s\n", i.Subject.Locality[0]))
		}
		if len(i.Subject.Province) > 0 {
			details.WriteString(fmt.Sprintf("- State       : %s\n", i.Subject.Province[0]))
		}
		if len(i.Subject.Country) > 0 {
			details.WriteString(fmt.Sprintf("- Country     : %s\n", i.Subject.Country[0]))
		}
		details.WriteString("## Alternate Names:")
		details.WriteString("\n")
		for _, v := range i.SAN {
			details.WriteString(fmt.Sprintf("  - %v\n", v))
		}
	}

	str, err := m.renderer.Render(details.String())
	if err != nil {
		log.Fatal().Err(err)
	}
	m.viewport.SetContent(str)
}

func (m domainDetails) view(h, w int) string {
	str := m.viewport.View()
	content := lipgloss.Place(
		w, h,
		lipgloss.Center, lipgloss.Center,
		str,
	)
	return content
}

type detailsKeyMap struct {
	CursorUp    key.Binding
	CursorDown  key.Binding
	ExitDetails key.Binding
	Quit        key.Binding
}

func (k detailsKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.CursorUp, k.CursorDown, k.ExitDetails, k.Quit}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k detailsKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.CursorUp, k.CursorDown}, // first column
		{k.ExitDetails, k.Quit},    // second column
	}
}

func setDetailsKeyMap() detailsKeyMap {
	return detailsKeyMap{
		CursorUp:    key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		CursorDown:  key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		ExitDetails: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "return to list")),
		Quit:        key.NewBinding(key.WithKeys("q", "esc"), key.WithHelp("q", "quit")),
	}
}
