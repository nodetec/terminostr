package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	// "github.com/charmbracelet/glamour"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	lg "github.com/charmbracelet/lipgloss"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

const relayUrl string = "wss://relay.damus.io"
const npub string = "npub1qd3hhtge6vhwapp85q8eg03gea7ftuf9um4r8x4lh4xfy2trgvksf6dkva"
const limit int = 5

type Styles struct {
	AccentColor lg.Color
	Box         lg.Style
	Title       lg.Style
	Number      lg.Style
	Info        lg.Style
	Separator   lg.Style
	Descrption  lg.Style
	Tag         lg.Style
	ActiveBox   lg.Style
	BoxWidth    int
}

type keyMap struct {
	Up   key.Binding
	Down key.Binding
	Help key.Binding
	Quit key.Binding
}

type model struct {
	cursor    int
	relay     string
	events    []*nostr.Event
	filter    []*nostr.Filter
	spinner   spinner.Model
	help      help.Model
	keys      keyMap
	loading   bool
	limit     int
	err       error
	width     int
	height    int
	styles    *Styles
	separator string
}

type errMsg struct{ err error }
type eventsMsg []*nostr.Event

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Help},
		{k.Up},
		{k.Down},
		{k.Quit},
	}
}

func DefaultStyles() *Styles {
	s := new(Styles)
	black := lg.Color("0")
	gray := lg.Color("8")
	s.AccentColor = lg.Color("12")
	s.BoxWidth = 90

	boxStyle := lg.NewStyle().
		BorderStyle(lg.RoundedBorder()).
		Margin(2).
		Width(s.BoxWidth).
		MaxWidth(s.BoxWidth + 2)

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

	return s
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
		relay:     relayUrl,
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
	filters = []nostr.Filter{{
		Kinds: []int{30023},
		Limit: limit,
	}}
	// if _, v, err := nip19.Decode(npub); err == nil {
	// 	pub := v.(string)
	// 	filters = []nostr.Filter{{
	// 		Kinds:   []int{1},
	// 		Authors: []string{pub},
	// 		Limit:   1,
	// 	}}
	// } else {
	// 	panic(err)
	// }
	//
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

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

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
			return m, tea.Quit

		case key.Matches(msg, m.keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, m.keys.Down):
			if m.cursor < len(m.events)-1 {
				m.cursor++
			}
		}

	default:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func truncateString(str string, maxLen int) string {
	if len(str) <= maxLen {
		return str
	}

	left := (maxLen - 3) / 2
	right := maxLen - 3 - left

	return fmt.Sprintf("%s...%s", str[0:left], str[len(str)-right:])
}

func (m model) View() string {
	if m.loading {
		spinner := m.spinner.View()
		s := fmt.Sprintf("%s  Loading  %s", spinner, spinner)
		return lg.Place(m.width, m.height, lg.Center, lg.Center, s)
	}

	s := ""
	for index, event := range m.events {
		title := event.Tags.GetFirst([]string{"title"}).Value()
    if len(title) >= m.styles.BoxWidth-14 {
      title = title[:m.styles.BoxWidth-14] + "..."
    }
		title = m.styles.Title.Render(title)

    number := m.styles.Number.Render(strconv.Itoa(index + 1) + ".")
		title = lg.JoinHorizontal(lg.Center, number, title)

		info := ""
		separator := m.styles.Separator.Render(m.separator)

		author := ""
		if v, err := nip19.EncodePublicKey(event.PubKey); err == nil {
			author = truncateString(v, 13)
		}

		author = m.styles.Info.Render(author)
		info = lg.JoinHorizontal(lg.Center, info, author)

		publishedAt := event.Tags.GetFirst([]string{"published_at"}).Value()
		publishedAt = getRelativeTime(publishedAt)
		publishedAt = m.styles.Info.Render(publishedAt)
		info = lg.JoinHorizontal(lg.Center, info, separator, publishedAt)

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
		if len(firstLine) > m.styles.BoxWidth-4 {
			firstLine = firstLine[:m.styles.BoxWidth-8] + "..."
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

	help := m.help.View(m.keys)

	s = lg.JoinVertical(lg.Center, s, help)
	s = lg.Place(m.width, m.height, lg.Center, lg.Top, s)

	return s
}

// func main() {
// 	var events = sub()
// 	// out, _ := glamour.Render(events[1].Content, "dark")
// 	// fmt.Print(out)
// 	for _, event := range events {
// 		out, _ := glamour.Render(event.Content, "dark")
// 		title := event.Tags.GetFirst([]string{"title"})
// 		fmt.Println(title.Value())
// 		// fmt.Print(event.Content)
// 	}
// }

func main() {
	f, ferr := tea.LogToFile("debug.log", "debug")
	if ferr != nil {
		fmt.Printf("Error: %v", ferr)
	}
	defer f.Close()
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	_, err := p.Run()
	if err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
