package ui

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fabio42/ssl-checker/domains"

	"github.com/AvraamMavridis/randomcolor"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rs/zerolog/log"
)

var (
	appStyle    = lipgloss.NewStyle().Padding(1, 2)
	detailStyle = lipgloss.NewStyle().Align(lipgloss.Center)
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render
)

type config struct {
	Timeout      int
	Silent       bool
	ChefRoot     string
	FilesQuery   map[string]string
	DomainsQuery map[string][]string
	EnvQuery     []string
	envStrWidth  int
	detailView   bool
	exportInput  bool
	exportDone   bool
	report       string
}

func newConfig(timeout int, silent bool, files map[string]string, customList map[string][]string, envs []string) *config {
	cfg := &config{
		Timeout:      timeout,
		Silent:       silent,
		FilesQuery:   files,
		DomainsQuery: customList,
		EnvQuery:     envs,
		report:       domains.DefaultReportFile,
	}
	for f := range files {
		w := utf8.RuneCountInString(f)
		if w > cfg.envStrWidth {
			cfg.envStrWidth = w
		}
	}
	for f := range customList {
		w := utf8.RuneCountInString(f)
		if w > cfg.envStrWidth {
			cfg.envStrWidth = w
		}
	}

	return cfg
}

type processor struct {
	queries   map[string]int
	processed map[string]int
	done      bool
	ch        chan domains.Response
}

func newProc(envs []string) *processor {
	var proc processor

	proc.queries = make(map[string]int)
	proc.processed = make(map[string]int)
	proc.ch = make(chan domains.Response)

	for _, e := range envs {
		proc.queries[e] = 0
	}
	return &proc
}

func waitForResponse(resp chan domains.Response) tea.Cmd {
	return func() tea.Msg {
		return domains.Response(<-resp)
	}
}

type procDone struct{}
type exportDone struct{}

type Model struct {
	cfg  *config
	proc *processor
	keys *listKeyMap

	list         *list.Model
	progressBars []progress.Model
	details      *domainDetails
	exportFile   textinput.Model
}

func NewModel(timeout int, silent bool, files map[string]string, customList map[string][]string) Model {
	var (
		environments []string
		progressBars []progress.Model
	)

	for e := range files {
		environments = append(environments, e)
		progressBars = append(progressBars, progress.New(progress.WithScaledGradient(randomcolor.GetRandomColorInHex(), "#00ff00")))
	}

	for e := range customList {
		environments = append(environments, e)
		progressBars = append(progressBars, progress.New(progress.WithScaledGradient(randomcolor.GetRandomColorInHex(), "#00ff00")))
	}

	cfg := newConfig(timeout, silent, files, customList, environments)
	proc := newProc(environments)
	keys := newListKeyMap()

	lst := list.New([]list.Item{}, itemDelegate{}, 0, 0)
	lst.Title = "SSL queries results"
	lst.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			keys.toggleDetails,
			keys.toggleExport,
		}
	}
	lst.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			keys.toggleDetails,
			keys.toggleExport,
		}
	}

	details := newDetails()

	export := textinput.New()
	export.Placeholder = domains.DefaultReportFile
	export.Focus()
	export.CharLimit = 156
	export.Width = 20

	return Model{
		cfg:          cfg,
		proc:         proc,
		keys:         keys,
		list:         &lst,
		progressBars: progressBars,
		details:      details,
		exportFile:   export,
	}
}

func (m Model) Init() tea.Cmd {
	log.Debug().Msgf("Init: file queries  : %v", m.cfg.FilesQuery)
	log.Debug().Msgf("Init: custom queries: %v", m.cfg.DomainsQuery)
	for env, target := range m.cfg.FilesQuery {
		expandedPath := os.ExpandEnv(target)
		_, err := os.Stat(expandedPath)
		if err != nil {
			log.Fatal().Msgf("Error can't find %s directory: %v", env, err)
		}

		file, err := os.Open(expandedPath)
		if err != nil {
			log.Fatal().Err(err)
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			domain := scanner.Text()
			if len(strings.TrimSpace(domain)) == 0 {
				continue
			}
			m.proc.queries[env] += 1
			go domains.TestDomain(domain, env, m.cfg.Timeout, m.proc.ch)
		}
	}
	for env, target := range m.cfg.DomainsQuery {
		for _, domain := range target {
			m.proc.queries[env] += 1
			go domains.TestDomain(domain, env, m.cfg.Timeout, m.proc.ch)
		}
	}

	var fullscreen tea.Cmd
	if !m.cfg.Silent {
		fullscreen = tea.EnterAltScreen
	}
	return tea.Batch(
		fullscreen,
		waitForResponse(m.proc.ch),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, w := appStyle.GetFrameSize()
		// size list view
		m.list.SetSize(msg.Width-h, msg.Height-w)
		// size progress bars view
		for k := range m.cfg.EnvQuery {
			m.progressBars[k].Width = msg.Width - (m.cfg.envStrWidth + 20)
		}
		// size viewport
		m.details.viewport.Width = msg.Width
		m.details.viewport.Height = msg.Height - 5
		m.details.help.Width = msg.Width
		return m, nil

	case tea.KeyMsg:
		// Handle export prompt
		if m.cfg.exportInput {
			if msg.Type == tea.KeyEnter || msg.Type == tea.KeyEscape {
				input := m.exportFile.Value()
				m.cfg.report = m.exportFile.Value()
				if msg.Type != tea.KeyEscape {
					if input != "" {
						m.cfg.report = input
					} else {
						m.cfg.report = domains.DefaultReportFile
					}
					m.exportResults(false)

					m.ListCursorsEnabled(true)
					m.cfg.exportInput = false
					m.cfg.exportDone = true
					m.list.SetHeight(m.list.Height() + 2)
					// Give feedback that export was successful
					return m, tea.Tick(3*time.Second, func(_ time.Time) tea.Msg {
						return exportDone{}
					})
				}
				m.ListCursorsEnabled(true)
				m.cfg.exportInput = false
				m.list.SetHeight(m.list.Height() + 2)
				return m, nil
			}
			break
		}

		// Don't match any of the keys below if we're actively filtering.
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case msg.String() == "ctrl+c" || msg.String() == "q":
			return m, tea.Quit
		case key.Matches(msg, m.keys.toggleDetails):
			i := m.list.SelectedItem()
			m.details.setData(i.(domains.Response))
			m.cfg.detailView = !m.cfg.detailView
			m.ListCursorsEnabled(!m.cfg.detailView)
			m.details.viewport.YOffset = 1
			return m, nil
		case key.Matches(msg, m.keys.toggleExport):
			m.cfg.exportInput = true
			m.ListCursorsEnabled(false)
			m.list.SetHeight(m.list.Height() - 2)
			return m, nil
		}

	case domains.Response:
		m.proc.processed[msg.Environment] += 1
		m.list.InsertItem(0, msg)

		done := true
		for env := range m.proc.queries {
			if m.proc.queries[env] != m.proc.processed[env] {
				done = false
				break
			}
		}
		if done {
			r := func() tea.Msg {
				return procDone{}
			}
			return m, r
		}
		return m, waitForResponse(m.proc.ch)

	case procDone:
		if m.cfg.Silent {
			m.exportResults(true)
			return m, tea.Quit
		}

		m.proc.done = true
		items := m.list.Items()
		now := time.Now()

		sort.Slice(items, func(i, j int) bool {
			// We want to move errors (null date value) to the end of list
			// adding 99 years there - XXX find a more elegant way
			iDate := items[i].(domains.Response).NotAfter
			jDate := items[j].(domains.Response).NotAfter
			if iDate.IsZero() {
				iDate = now.AddDate(99, 0, 0)
			}
			if jDate.IsZero() {
				jDate = now.AddDate(99, 0, 0)
			}
			return iDate.Before(jDate)
		})
		m.list.SetItems(items)
		time.Sleep(1 * time.Second)
		return m, nil

	case exportDone:
		m.cfg.exportDone = false
		return m, nil
	}
	// This also call our delegate's update function.
	newListModel, cmd := m.list.Update(msg)
	cmds = append(cmds, cmd)

	m.list = &newListModel
	if m.cfg.detailView {
		*m.details.viewport, cmd = m.details.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.cfg.exportInput {
		m.exportFile, cmd = m.exportFile.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.cfg.Silent {
		return ""
	}

	var str strings.Builder
	if m.cfg.detailView {
		str.WriteString(detailStyle.Render(m.details.view(m.list.Height(), m.list.Width())))
		str.WriteString(lipgloss.NewStyle().Padding(0, 4).Render("\n" + m.details.helpView()))

		return str.String()
	} else if m.proc.done {
		if m.cfg.exportInput {
			str.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render("Report file: ") + m.exportFile.View())
		}
		if m.cfg.exportDone {
			str.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("Export successful!"))
		}
		str.WriteString(appStyle.Render(m.list.View()))
		if m.cfg.exportInput {
			str.WriteString(helpStyle("\n Press Enter to confirm or escape to cancel\n"))
		}
	} else {
		str.WriteString("\n Doing some work...\n\n")
		for k, v := range m.cfg.EnvQuery {
			progress := float64(m.proc.processed[v]) / float64(m.proc.queries[v])
			str.WriteString(fmt.Sprintf(" - %*s: %v \n", m.cfg.envStrWidth, v, m.progressBars[k].ViewAs(progress)))
		}
		str.WriteString(helpStyle("\n Press any key to exit\n"))
	}
	return str.String()
}

// exportResults exposer results to user
func (m Model) exportResults(stdOut bool) {
	resp := make([]domains.Response, len(m.list.Items()))
	for k, v := range m.list.Items() {
		resp[k] = v.(domains.Response)
	}
	domains.CreateReport(resp, m.cfg.EnvQuery, m.cfg.report, stdOut)
}

// ListCursorsEnabled manage list default keymap hooks to avoid conflicts
func (m Model) ListCursorsEnabled(state bool) {
	// Also manage list filter for / keymap
	// SetFilteringEnabled seems to reset other keyMap to true so have to be done first
	m.list.SetFilteringEnabled(state)

	m.list.KeyMap.CursorUp.SetEnabled(state)
	m.list.KeyMap.CursorDown.SetEnabled(state)
	m.list.KeyMap.NextPage.SetEnabled(state)
	m.list.KeyMap.PrevPage.SetEnabled(state)
	m.list.KeyMap.GoToStart.SetEnabled(state)
	m.list.KeyMap.GoToEnd.SetEnabled(state)
	m.list.KeyMap.Quit.SetEnabled(state)
	m.list.KeyMap.ShowFullHelp.SetEnabled(state)
	m.list.SetShowHelp(state)
}
