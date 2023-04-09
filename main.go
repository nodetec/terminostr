package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	lg "github.com/charmbracelet/lipgloss"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

const relayUrl string = "wss://relay.damus.io"
const useHighPerformanceRenderer = false
const boxViewHeight = 11 // FIXME: bruh, why value is hardcoded

const limit int = 0

type Styles struct {
	AccentColor   lg.Color
	Box           lg.Style
	Title         lg.Style
	Number        lg.Style
	Info          lg.Style
	Separator     lg.Style
	Descrption    lg.Style
	Tag           lg.Style
	ActiveBox     lg.Style
	ViewportTitle lg.Style
	ViewportInfo  lg.Style
	MaxWidth      int
}

type keyMap struct {
	Up    key.Binding
	Down  key.Binding
	Next  key.Binding
	Prev  key.Binding
	Enter key.Binding
	Help  key.Binding
	Quit  key.Binding
}

type model struct {
	cursor         int
	view           bool
	viewportReady  bool
	viewport       viewport.Model
	events         []*nostr.Event
	spinner        spinner.Model
	help           help.Model
	keys           keyMap
	loading        bool
	limit          int
	err            error
	width          int
	height         int
	styles         *Styles
	separator      string
	mainViewHeight int
	currentPage    int
}

type errMsg struct{ err error }
type eventsMsg []*nostr.Event

func DefaultStyles() *Styles {
	s := new(Styles)
	black := lg.Color("0")
	gray := lg.Color("8")
	s.AccentColor = lg.Color("12")
	s.MaxWidth = 90

	boxStyle := lg.NewStyle().
		BorderStyle(lg.RoundedBorder()).
		Margin(2).
		Width(s.MaxWidth).
		MaxWidth(s.MaxWidth + 2)

	s.Box = lg.NewStyle().
		BorderForeground(black).
		Padding(1, 2).
		Inherit(boxStyle)

	s.ActiveBox = lg.NewStyle().
		BorderForeground(s.AccentColor).
		Padding(1, 2).
		Inherit(boxStyle)

	s.Title = lg.NewStyle().Bold(true)

	s.Info = lg.NewStyle().MarginTop(1).Foreground(gray)

	s.Number = lg.NewStyle().
		MarginRight(2).
		Padding(0, 1, 0, 1).
		Background(s.AccentColor).
		Foreground(black).
		Bold(true)

	s.Separator = lg.NewStyle().Foreground(gray).Margin(0, 1, 0, 1)

	s.Descrption = lg.NewStyle().MarginTop(1)

	s.Tag = lg.NewStyle().
		Background(black).
		Bold(true).
		Padding(0, 1).
		Margin(1, 1, 0, 0)

	vptitle := lg.RoundedBorder()
	vptitle.Right = "├"
	s.ViewportTitle = lg.NewStyle().
		BorderStyle(vptitle).
		Padding(0, 1).
		BorderForeground(s.AccentColor)

	vpinfo := lg.RoundedBorder()
	vpinfo.Left = "┤"
	s.ViewportInfo = s.ViewportTitle.Copy().BorderStyle(vpinfo)

	return s
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Help},
		{k.Enter},
		{k.Up},
		{k.Down},
		{k.Next},
		{k.Prev},
		{k.Quit},
	}
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys(tea.KeyUp.String(), "k"),
		key.WithHelp("↑/k", "Up"),
	),
	Down: key.NewBinding(
		key.WithKeys(tea.KeyDown.String(), "j"),
		key.WithHelp("↓/j", "Down"),
	),
	Next: key.NewBinding(
		key.WithKeys(tea.KeyRight.String(), "l"),
		key.WithHelp("→/l", "Next"),
	),
	Prev: key.NewBinding(
		key.WithKeys(tea.KeyLeft.String(), "h"),
		key.WithHelp("←/h", "Prev"),
	),
	Enter: key.NewBinding(
		key.WithKeys(tea.KeyEnter.String()),
		key.WithHelp("󰌑/enter", "Open"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "Help"),
	),
	Quit: key.NewBinding(
		key.WithKeys(tea.KeyCtrlC.String(), tea.KeyEsc.String(), "q"),
		key.WithHelp("esc/q", "Quit"),
	),
}

func initialModel() *model {
	styles := DefaultStyles()

	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lg.NewStyle().Foreground(styles.AccentColor)

	return &model{
		spinner:   s,
		limit:     limit,
		styles:    styles,
		keys:      keys,
		loading:   true,
		help:      help.New(),
		separator: "",
	}
}

func sub() tea.Msg {
	relay, err := nostr.RelayConnect(context.Background(), relayUrl)
	if err != nil {
		panic(err)
	}

	var filters nostr.Filters

	// filters = []nostr.Filter{{
	// 	Kinds: []int{30023},
	// 	Limit: limit,
	// }}

  npub := "npub1ygzj9skr9val9yqxkf67yf9jshtyhvvl0x76jp5er09nsc0p3j6qr260k2"
	if _, v, err := nip19.Decode(npub); err == nil {
		pub := v.(string)
		filters = []nostr.Filter{{
			Kinds:   []int{30023},
			Authors: []string{pub},
			Limit:   limit,
		}}
	} else {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	sub, err := relay.Subscribe(ctx, filters)

	go func() {
		<-sub.EndOfStoredEvents
		// handle end of stored events (EOSE, see NIP-15)
		cancel()
	}()

	if err != nil {
		return errMsg{err}
	}

	var events []*nostr.Event

	for ev := range sub.Events {
		// handle returned event.
		// channel will stay open until the ctx is cancelled (in this case, by calling cancel())
		events = append(events, ev)
	}
	return eventsMsg(events)
}

func (e errMsg) Error() string {
	return e.err.Error()
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, sub)
}

func calculateTotalPages(eventCount int, eventsPerPage int) int {
	totalPages := eventCount / eventsPerPage
	if eventCount%eventsPerPage != 0 {
		totalPages++
	}
	return totalPages - 1
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.mainViewHeight = msg.Height - 2 // FIXME: bruh, why value is hardcoded

		headerHeight := lg.Height(m.headerView()) + 1
		footerHeight := lg.Height(m.footerView())
		verticalMarginHeight := headerHeight + footerHeight
		if !m.viewportReady {
			m.viewport = viewport.New(m.styles.MaxWidth, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.HighPerformanceRendering = useHighPerformanceRenderer
			m.viewportReady = true
		} else {
			m.viewport.Width = m.styles.MaxWidth
			m.viewport.Height = msg.Height - verticalMarginHeight
		}

		if useHighPerformanceRenderer {
			cmds = append(cmds, viewport.Sync(m.viewport))
		}

	case eventsMsg:
		m.events = msg
		m.loading = false

	case errMsg:
		m.err = msg
		m.loading = false

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll

		case key.Matches(msg, m.keys.Quit):
			if m.view {
				m.view = false
			} else {
				return m, tea.Quit
			}

		case key.Matches(msg, m.keys.Up):
			if !m.view {
				if m.cursor > m.currentPage * (m.mainViewHeight / boxViewHeight) {
					m.cursor--
				}
			}

		case key.Matches(msg, m.keys.Down):
			if !m.view {
				if m.cursor < m.currentPage * (m.mainViewHeight / boxViewHeight) + (m.mainViewHeight / boxViewHeight) -1 {
					m.cursor++
				}
			}

		case key.Matches(msg, m.keys.Next):
			if !m.view {
				if m.currentPage < calculateTotalPages(len(m.events), (m.mainViewHeight/boxViewHeight)) {
					m.cursor += m.mainViewHeight / boxViewHeight
					m.currentPage++
				}
			}

		case key.Matches(msg, m.keys.Prev):
			if !m.view {
				if m.currentPage > 0 {
					m.cursor -= m.mainViewHeight / boxViewHeight
					m.currentPage--
				}
			}

		case key.Matches(msg, m.keys.Enter):
			m.view = true
			if m.events != nil {
				out, _ := glamour.Render(m.events[m.cursor].Content, "dark")
				m.viewport.SetContent(out)
			}
		}

	default:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func truncateString(str string, maxLen int) string {
	if lg.Width(str) <= maxLen {
		return str
	}

	left := (maxLen - 3) / 2
	right := maxLen - 3 - left

	return fmt.Sprintf("%s...%s", str[0:left], str[len(str)-right:])
}

func (m model) headerView() string {
	title := m.styles.ViewportTitle.Render("Mr. Pager")
	line := strings.Repeat("─", max(0, m.viewport.Width-lg.Width(title)))
	line = lg.NewStyle().Foreground(m.styles.AccentColor).Render(line)
	return lg.JoinHorizontal(lg.Center, title, line)
}

func (m model) footerView() string {
	info := m.styles.ViewportInfo.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lg.Width(info)))
	line = lg.NewStyle().Foreground(m.styles.AccentColor).Render(line)
	return lg.JoinHorizontal(lg.Center, line, info)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func paginateEvents(events []*nostr.Event, currentPage, eventsPerPage int) []*nostr.Event {
	start := currentPage * eventsPerPage
	end := start + eventsPerPage

	if start > len(events) {
		return []*nostr.Event{}
	}

	if end > len(events) {
		end = len(events)
	}

	return events[start:end]
}

func (m model) View() string {
	if m.loading {
		spinner := m.spinner.View()
		s := fmt.Sprintf("fetching from %s  %s", relayUrl, spinner)
		return lg.Place(m.width, m.height, lg.Center, lg.Center, s)
	}

	s := ""
	if m.view {
		if !m.viewportReady {
			s = "Initializing..."
		}
		s = fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView())
	} else {
		for index, event := range paginateEvents(m.events, m.currentPage, m.mainViewHeight/boxViewHeight) {
			pageOffset := m.currentPage * m.mainViewHeight / boxViewHeight
			index = index + pageOffset
			title := event.Tags.GetFirst([]string{"title"}).Value()
			if lg.Width(title) >= m.styles.MaxWidth-14 {
				title = title[:m.styles.MaxWidth-14] + "..."
			}
			title = m.styles.Title.Render(title)

			number := m.styles.Number.Render(strconv.Itoa(index+1) + ".")
			title = lg.JoinHorizontal(lg.Center, number, title)

			info := ""
			separator := m.styles.Separator.Render(m.separator)

			author := ""
			if v, err := nip19.EncodePublicKey(event.PubKey); err == nil {
				author = truncateString(v, 13)
			}

			author = m.styles.Info.Render(author)
			info = lg.JoinHorizontal(lg.Center, info, author)

			publishedAt := event.Tags.GetFirst([]string{"published_at"})
			if publishedAt != nil {
				p := getRelativeTime(publishedAt.Value())
				p = m.styles.Info.Render(p)
				info = lg.JoinHorizontal(lg.Center, info, separator, p)
			}

			client := event.Tags.GetFirst([]string{"client"})
			if client != nil {
				c := client.Value()
				c = m.styles.Info.Render(c)
				info = lg.JoinHorizontal(lg.Center, info, separator, c)
			}

			descrpition := ""

			summary := event.Tags.GetFirst([]string{"summary"})
			if summary != nil {
				descrpition = summary.Value()
			} else {
				descrpition = event.Content
			}

			firstLine := strings.Split(descrpition, "\n")[0]
			if lg.Width(firstLine) > m.styles.MaxWidth-4 {
				firstLine = firstLine[:m.styles.MaxWidth-8] + "..."
			}
			descrpition = m.styles.Descrption.Render(firstLine)

			etags := event.Tags.GetAll([]string{"t"})
			etags.FilterOut([]string{"title"})

			tags := ""
			for _, tag := range etags {
				if tag.Key() == "title" {
					continue
				}

				t := m.styles.Tag.Render(tag.Value())
				tags = lg.JoinHorizontal(lg.Right, tags, t)
			}

			box := lg.JoinVertical(lg.Left, title, info, descrpition, tags)

			if index == m.cursor {
				box = m.styles.ActiveBox.Render(box)
			} else {
				box = m.styles.Box.Render(box)
			}

			s = lg.JoinVertical(lg.Left, s, box)
		}
	}

	help := m.help.View(m.keys)

	s = lg.JoinVertical(lg.Center, s, help)
	s = lg.Place(m.width, m.height, lg.Center, lg.Top, s)

	return s
}

func main() {
	f, ferr := tea.LogToFile("debug.log", "debug")
	if ferr != nil {
		fmt.Printf("Error: %v", ferr)
	}
	defer f.Close()
	p := tea.NewProgram(initialModel(), tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	if err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
