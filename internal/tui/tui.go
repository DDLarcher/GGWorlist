package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"

	"wordlist-generator/internal/mixer"
	"wordlist-generator/internal/numeric"
	"wordlist-generator/internal/ui"
)

var (
	primary = lipgloss.Color("#22CCFF")
	bg      = lipgloss.Color("#000000")
	white   = lipgloss.Color("#FFFFFF")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primary).
			Background(bg).
			MarginBottom(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primary).
			Background(bg).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(primary).
			Background(bg).
			Bold(true)

	promptStyle = lipgloss.NewStyle().
			Foreground(primary).
			Background(bg)

	errorStyle = lipgloss.NewStyle().
			Foreground(white).
			Background(lipgloss.Color("#7F0000")).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
		Foreground(primary).
		Background(bg)

	boxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primary).
		Background(bg).
		Foreground(primary).
		Padding(0, 2).
		MarginTop(1)

	greenStyle = lipgloss.NewStyle().
			Foreground(primary).
			Background(bg)

	greenBoldStyle = lipgloss.NewStyle().
			Foreground(primary).
			Background(bg).
			Bold(true)
)

const logo = `   ___________      __            _____     __
  / ___/ ___/ | /| / /__  _______/ / (_)__ / /_
 / (_ / (_ /| |/ |/ / _ \/ __/ _  / / (_-</ __/
 \___/\___/ |__/|__/\___/_/  \_,_/_/_/___/\__/
                                              `

type menuItem struct {
	title string
	desc  string
}

func (m menuItem) Title() string       { return m.title }
func (m menuItem) Description() string { return m.desc }
func (m menuItem) FilterValue() string { return m.title }

type generateMsg struct {
	written   uint64
	files     int
	elapsed   time.Duration
	outDir    string
	err       error
	capped    bool
}

type progressMsg struct {
	written uint64
	files   int
}

type screen int

const (
	screenMode screen = iota
	screenOutputDir
	screenMaxGB
	screenMixerSource
	screenMixerWords
	screenMixerFile
	screenMixerMaxCap
	screenMixerCasing
	screenMixerNumbers        // ask y/n
	screenMixerNumberRanges    // collect ranges
	screenMixerSpec            // ask y/n
	screenMixerSpecLen
	screenMixerDepth
	screenMixerWarning
	screenNumericLength
	screenNumericWarning
	screenGenerating
	screenDone
)

type Model struct {
	screen         screen
	prevScreen     screen
	list           list.Model
	textInput      textinput.Model
	mode           int
	outDir         string
	maxBytes       uint64
	words          []string
	maxCap         int
	casingIdxs     []int
	casingSelected map[int]bool
	appendNums     bool
	numbers        []int
	appendSpec     bool
	specCombos     []string
	maxSpecLen     int
	sizes          []int
	numericLen     int
	err            string
	progress       string
	result         generateMsg
	width          int
	height         int
}

func New() Model {
	ti := textinput.New()
	ti.CharLimit = 1000
	m := Model{
		screen:         screenMode,
		textInput:      ti,
		casingIdxs:     []int{0, 1, 2, 3},
		casingSelected: map[int]bool{0: true, 1: true, 2: true, 3: true},
		maxSpecLen:     4,
		sizes:          []int{2, 3},
	}
	m.initModeList()
	return m
}

func (m *Model) styleList(l list.Model) list.Model {
	l.Styles.Title = l.Styles.Title.
		Foreground(primary).
		Background(bg).
		Bold(true)
	l.Styles.TitleBar = l.Styles.TitleBar.
		Background(bg)
	l.Styles.HelpStyle = l.Styles.HelpStyle.
		Foreground(primary).
		Background(bg)
	l.Styles.StatusBar = l.Styles.StatusBar.
		Foreground(primary).
		Background(bg)
	l.Styles.PaginationStyle = l.Styles.PaginationStyle.
		Foreground(primary).
		Background(bg)

	d := list.NewDefaultDelegate()
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.
		Foreground(white).
		Background(primary).
		Bold(true)
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.
		Foreground(white).
		Background(primary)
	d.Styles.NormalTitle = d.Styles.NormalTitle.
		Foreground(primary).
		Background(bg)
	d.Styles.NormalDesc = d.Styles.NormalDesc.
		Foreground(primary).
		Background(bg)
	d.Styles.FilterMatch = d.Styles.FilterMatch.
		Foreground(primary).
		Background(bg)
	l.SetDelegate(d)
	return l
}

func (m *Model) initModeList() {
	items := []list.Item{
		menuItem{"Word Mixer", "Combine words with casings, numbers, special chars"},
		menuItem{"Numeric Generator", "All digit combos of length n, no 3 consecutive"},
	}
	l := list.New(items, list.NewDefaultDelegate(), 60, 10)
	l.Title = "Choose Mode"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l = m.styleList(l)
	m.list = l
}

func (m *Model) initList(title string, items []list.Item, width int) {
	l := list.New(items, list.NewDefaultDelegate(), width, 12)
	l.Title = title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l = m.styleList(l)
	m.list = l
}

func (m *Model) initInput(placeholder string) {
	m.textInput = textinput.New()
	m.textInput.CharLimit = 1000
	m.textInput.Placeholder = placeholder
	m.textInput.PlaceholderStyle = lipgloss.NewStyle().Foreground(primary).Background(bg)
	m.textInput.PromptStyle = lipgloss.NewStyle().Foreground(primary).Background(bg)
	m.textInput.TextStyle = lipgloss.NewStyle().Foreground(white).Background(bg)
	m.textInput.Cursor.Style = lipgloss.NewStyle().Foreground(white).Background(bg)
	m.textInput.Focus()
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width-4, 12)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.screen == screenGenerating {
				return m, nil
			}
			return m, tea.Quit
		case "esc":
			if m.screen == screenDone {
				return m, tea.Quit
			}
			if m.prevScreen >= 0 && m.screen != screenMode {
				m.screen = m.prevScreen
				m.prevScreen = -1
				return m, nil
			}
			return m, tea.Quit
		}

	case generateMsg:
		m.result = msg
		m.screen = screenDone
		return m, nil

	case progressMsg:
		m.progress = fmt.Sprintf("Written: %d  File #%d", msg.written, msg.files)
		return m, nil
	}

	switch m.screen {
	case screenMode:
		return m.updateMode(msg)
	case screenOutputDir:
		return m.updateOutputDir(msg)
	case screenMaxGB:
		return m.updateMaxGB(msg)
	case screenMixerSource:
		return m.updateMixerSource(msg)
	case screenMixerWords:
		return m.updateMixerWords(msg)
	case screenMixerFile:
		return m.updateMixerFile(msg)
	case screenMixerMaxCap:
		return m.updateMixerMaxCap(msg)
	case screenMixerCasing:
		return m.updateMixerCasing(msg)
	case screenMixerNumbers:
		return m.updateMixerNumbers(msg)
	case screenMixerNumberRanges:
		return m.updateMixerNumberRanges(msg)
	case screenMixerSpec:
		return m.updateMixerSpec(msg)
	case screenMixerSpecLen:
		return m.updateMixerSpecLen(msg)
	case screenMixerDepth:
		return m.updateMixerDepth(msg)
	case screenMixerWarning:
		return m.updateMixerWarning(msg)
	case screenNumericLength:
		return m.updateNumericLength(msg)
	case screenNumericWarning:
		return m.updateNumericWarning(msg)
	case screenGenerating:
		return m, nil
	case screenDone:
		return m, nil
	}
	return m, nil
}

func (m Model) View() string {
	var content string
	switch m.screen {
	case screenMode:
		content = m.viewLogo() + "\n" + m.list.View()
	case screenOutputDir:
		content = m.viewOutputDir()
	case screenMaxGB:
		content = m.viewMaxGB()
	case screenMixerSource:
		content = m.viewHeader("Word Mixer") + "\n" + m.list.View()
	case screenMixerWords:
		content = m.viewMixerWords()
	case screenMixerFile:
		content = m.viewMixerFile()
	case screenMixerMaxCap:
		content = m.viewMixerMaxCap()
	case screenMixerCasing:
		content = m.viewHeader("Choose Casing Modes") + "\n" + m.viewCasingList()
	case screenMixerNumbers:
		content = m.viewMixerNumbers()
	case screenMixerNumberRanges:
		content = m.viewMixerNumberRanges()
	case screenMixerSpec:
		content = m.viewMixerSpec()
	case screenMixerSpecLen:
		content = m.viewMixerSpecLen()
	case screenMixerDepth:
		content = m.viewHeader("Choose Combination Depth") + "\n" + m.list.View()
	case screenMixerWarning:
		content = m.viewMixerWarning()
	case screenNumericLength:
		content = m.viewNumericLength()
	case screenNumericWarning:
		content = m.viewNumericWarning()
	case screenGenerating:
		content = m.viewGenerating()
	case screenDone:
		content = m.viewDone()
	}

	fullBg := lipgloss.NewStyle().Background(bg).Foreground(primary)
	return fullBg.Render(content)
}

func (m Model) viewLogo() string {
	return titleStyle.Render(logo)
}

func (m Model) viewHeader(title string) string {
	return headerStyle.Render(title)
}

func (m Model) viewOutputDir() string {
	return m.viewHeader("Output Directory") + "\n\n" +
		dimStyle.Render("Where to save the wordlist files") + "\n\n" +
		m.textInput.View() + "\n\n" +
		dimStyle.Render("Enter = continue (default: output)  |  esc = back")
}

func (m Model) viewMaxGB() string {
	return m.viewHeader("Max Total Output") + "\n\n" +
		dimStyle.Render("Maximum output size in GB (empty = no cap)") + "\n\n" +
		m.textInput.View() + "\n\n" +
		dimStyle.Render("Enter = continue  |  esc = back")
}

func (m Model) viewMixerWords() string {
	return m.viewHeader("Word Mixer — Input Words") + "\n\n" +
		dimStyle.Render("Type words separated by spaces or commas") + "\n" +
		dimStyle.Render("Enter a blank line to finish") + "\n\n" +
		m.textInput.View() + "\n\n" +
		m.viewLoadedWords() + "\n" +
		dimStyle.Render("Enter = add word(s)  |  two Enters = finish  |  esc = back")
}

func (m Model) viewLoadedWords() string {
	if len(m.words) == 0 {
		return dimStyle.Render("(no words yet)")
	}
	return selectedStyle.Render(fmt.Sprintf("Loaded %d words: %s", len(m.words), strings.Join(m.words, ", ")))
}

func (m Model) viewCasingList() string {
	casings := []string{"PascalCase (CinoCat)", "camelCase (cinoCat)", "ALL CAPS (CINOCAT)", "all lowercase (cinocat)"}
	lines := []string{}
	for i, c := range casings {
		marker := "○"
		if m.casingSelected[i] {
			marker = "●"
		}
		cursor := " "
		if i == m.list.Index() {
			cursor = ">"
		}
		lines = append(lines, greenStyle.Render(fmt.Sprintf("  %s %s %s", cursor, marker, c)))
	}
	lines = append(lines, "")
	lines = append(lines, greenStyle.Render("↑↓ navigate  |  space toggle  |  enter confirm"))
	return strings.Join(lines, "\n")
}

func (m Model) viewMixerFile() string {
	return m.viewHeader("Word Mixer — Load From File") + "\n\n" +
		dimStyle.Render("Path to a text file containing words") + "\n\n" +
		m.textInput.View() + "\n\n" +
		dimStyle.Render("Enter = load  |  esc = back")
}

func (m Model) viewMixerMaxCap() string {
	return m.viewHeader("Word Mixer — Max Length Cap") + "\n\n" +
		dimStyle.Render("Skip combinations longer than this (0 = no cap)") + "\n\n" +
		m.textInput.View() + "\n\n" +
		dimStyle.Render("Enter = continue  |  esc = back")
}

func (m Model) viewMixerNumbers() string {
	return m.viewHeader("Word Mixer — Append Numbers?") + "\n\n" +
		dimStyle.Render("Append number ranges to words?") + "\n\n" +
		dimStyle.Render("[y] yes  |  [n] no (default)") + "\n\n" +
		m.textInput.View() + "\n\n" +
		dimStyle.Render("Enter = continue  |  esc = back")
}

func (m Model) viewMixerNumberRanges() string {
	return m.viewHeader("Word Mixer — Number Ranges") + "\n\n" +
		dimStyle.Render("Examples: 1-100, 1900-2030, 69, 1-10,1900-2030,69") + "\n\n" +
		m.textInput.View() + "\n\n" +
		dimStyle.Render("Enter = continue  |  esc = back")
}

func (m Model) viewMixerSpec() string {
	return m.viewHeader("Word Mixer — Special Characters") + "\n\n" +
		dimStyle.Render("Append ! ? $ . @ # * & + = combinations?") + "\n\n" +
		dimStyle.Render("[y] yes  |  [n] no (default)") + "\n\n" +
		m.textInput.View() + "\n\n" +
		dimStyle.Render("Enter = continue  |  esc = back")
}

func (m Model) viewMixerSpecLen() string {
	return m.viewHeader("Word Mixer — Max Special Chars") + "\n\n" +
		dimStyle.Render("How many special chars to combine (1-8, default: 4)") + "\n\n" +
		m.textInput.View() + "\n\n" +
		dimStyle.Render("Enter = continue  |  esc = back")
}

func (m Model) viewMixerWarning() string {
	return m.viewHeader("WARNING") + "\n\n" +
		errorStyle.Render("Large output estimated. This will use significant memory.") + "\n\n" +
		dimStyle.Render("Continue? [y] yes  |  [n] no (default)") + "\n\n" +
		m.textInput.View()
}

func (m Model) viewNumericLength() string {
	return m.viewHeader("Numeric Generator — String Length") + "\n\n" +
		dimStyle.Render("Length of digit strings (1-16, recommended: 1-14)") + "\n\n" +
		m.textInput.View() + "\n\n" +
		dimStyle.Render("Enter = continue  |  esc = back")
}

func (m Model) viewNumericWarning() string {
	return m.viewHeader("Numeric Generator — Size Warning") + "\n\n" +
		errorStyle.Render("Estimated output exceeds your size cap!") + "\n" +
		dimStyle.Render("The generator will STOP when it hits the cap.") + "\n\n" +
		dimStyle.Render("Continue anyway? [y] yes  |  [n] no (default)") + "\n\n" +
		m.textInput.View()
}

func (m Model) viewGenerating() string {
	return m.viewHeader("Generating...") + "\n\n" +
		m.progress + "\n\n" +
		dimStyle.Render("Press Ctrl+C to stop early")
}

func (m Model) viewDone() string {
	r := m.result
	lines := []string{}

	lines = append(lines, greenBoldStyle.Render("Generation Complete"))
	lines = append(lines, greenStyle.Render(strings.Repeat("─", 24)))
	lines = append(lines, "")

	stats := [][2]string{
		{"Words written", fmt.Sprintf("%d", r.written)},
		{"Files created", fmt.Sprintf("%d", r.files)},
		{"Elapsed time", r.elapsed.String()},
	}
	if r.elapsed.Seconds() > 0 {
		stats = append(stats, [2]string{"Rate", fmt.Sprintf("%.0f words/sec", float64(r.written)/r.elapsed.Seconds())})
	}
	stats = append(stats, [2]string{"Output directory", r.outDir})

	maxLabel := 0
	for _, s := range stats {
		if len(s[0]) > maxLabel {
			maxLabel = len(s[0])
		}
	}

	for _, s := range stats {
		lines = append(lines, fmt.Sprintf("  %-*s  %s",
			maxLabel,
			greenStyle.Render(s[0]),
			greenBoldStyle.Render(s[1])))
	}

	if r.err != nil {
		lines = append(lines, "")
		lines = append(lines, errorStyle.Render(r.err.Error()))
	}
	if r.capped {
		lines = append(lines, "")
		lines = append(lines, errorStyle.Render("Stopped at size cap."))
	}

	lines = append(lines, "")
	lines = append(lines, dimStyle.Render("Press Enter or esc to exit"))

	return boxStyle.Render(strings.Join(lines, "\n"))
}

func (m Model) updateMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			m.mode = int(m.list.Index())
			m.prevScreen = screenMode
			if m.mode == 0 {
				m.initInput("output")
				m.screen = screenOutputDir
			} else {
				m.initInput("output")
				m.screen = screenOutputDir
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) updateOutputDir(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			m.outDir = m.textInput.Value()
			if m.outDir == "" {
				m.outDir = "output"
			}
			m.initInput("")
			m.screen = screenMaxGB
			m.prevScreen = screenOutputDir
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m Model) updateMaxGB(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			val := m.textInput.Value()
			if val == "" {
				m.maxBytes = 0
			} else {
				var gb float64
				fmt.Sscanf(val, "%f", &gb)
				if gb > 0 {
					m.maxBytes = uint64(gb * float64(1000*1000*1000))
				}
			}
			if m.mode == 0 {
				m.initMixerSourceList()
				m.screen = screenMixerSource
			} else {
				m.initInput("8")
				m.screen = screenNumericLength
			}
			m.prevScreen = screenMaxGB
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m *Model) initMixerSourceList() {
	items := []list.Item{
		menuItem{"Type Words Directly", "Enter words in the terminal"},
		menuItem{"Load From File", "Read words from a text file"},
	}
	m.initList("Word Mixer — Input Source", items, 60)
}

func (m Model) updateMixerSource(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			idx := m.list.Index()
			m.initInput("")
			if idx == 0 {
				m.words = []string{}
				m.screen = screenMixerWords
			} else {
				m.screen = screenMixerFile
			}
			m.prevScreen = screenMixerSource
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) updateMixerWords(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			val := m.textInput.Value()
			if val == "" {
				if len(m.words) == 0 {
					m.err = "No words provided."
					return m, nil
				}
				m.initInput("0")
				m.screen = screenMixerMaxCap
				m.prevScreen = screenMixerWords
				return m, nil
			}
			parts := strings.FieldsFunc(val, func(r rune) bool {
				return r == ',' || r == ' ' || r == '\t'
			})
			seen := make(map[string]bool)
			for _, p := range parts {
				p = strings.TrimSpace(p)
				p = strings.TrimPrefix(p, "\ufeff")
				if p == "" || seen[p] {
					continue
				}
				seen[p] = true
				m.words = append(m.words, p)
			}
			m.textInput.SetValue("")
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m Model) updateMixerFile(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			path := m.textInput.Value()
			if path != "" {
				data, err := readFile(path)
				if err != nil {
					m.err = err.Error()
					return m, nil
				}
				parts := strings.FieldsFunc(string(data), func(r rune) bool {
					return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
				})
				seen := make(map[string]bool)
				for _, p := range parts {
					p = strings.TrimSpace(p)
					p = strings.TrimPrefix(p, "\ufeff")
					if p == "" || seen[p] {
						continue
					}
					seen[p] = true
					m.words = append(m.words, p)
				}
			}
			m.initInput("0")
			m.screen = screenMixerMaxCap
			m.prevScreen = screenMixerFile
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m Model) updateMixerMaxCap(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			val := m.textInput.Value()
			cap, _ := parseIntSafe(val, 0)
			if cap < 0 {
				cap = 0
			}
			m.maxCap = cap
			m.initCasingList()
			m.screen = screenMixerCasing
			m.prevScreen = screenMixerMaxCap
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m *Model) initCasingList() {
	items := []list.Item{
		menuItem{"PascalCase", "CinoCat"},
		menuItem{"camelCase", "cinoCat"},
		menuItem{"ALL CAPS", "CINOCAT"},
		menuItem{"all lowercase", "cinocat"},
	}
	m.initList("Choose Casing Modes (space to toggle, enter to confirm)", items, 50)
}

func (m Model) updateMixerCasing(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case " ":
			idx := m.list.Index()
			m.casingSelected[idx] = !m.casingSelected[idx]
			return m, nil
		case "enter":
			m.casingIdxs = []int{}
			for i := 0; i < 4; i++ {
				if m.casingSelected[i] {
					m.casingIdxs = append(m.casingIdxs, i)
				}
			}
			if len(m.casingIdxs) == 0 {
				m.casingIdxs = []int{0}
			}
			m.initInput("n")
			m.screen = screenMixerNumbers
			m.prevScreen = screenMixerCasing
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) updateMixerNumbers(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			val := strings.ToLower(m.textInput.Value())
			if val == "y" || val == "yes" {
				m.appendNums = true
				m.initInput("1-100, 1900-2030")
				m.screen = screenMixerNumberRanges
				m.prevScreen = screenMixerNumbers
				return m, nil
			}
			m.appendNums = false
			m.initInput("n")
			m.screen = screenMixerSpec
			m.prevScreen = screenMixerNumbers
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m Model) updateMixerNumberRanges(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			rangesStr := m.textInput.Value()
			if rangesStr != "" {
				nums, err := mixer.ParseNumberRanges(rangesStr)
				if err != nil {
					m.err = err.Error()
					return m, nil
				}
				m.numbers = nums
			}
			m.initInput("n")
			m.screen = screenMixerSpec
			m.prevScreen = screenMixerNumberRanges
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m Model) updateMixerSpec(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			val := strings.ToLower(m.textInput.Value())
			if val == "y" || val == "yes" {
				m.appendSpec = true
				m.initInput("4")
				m.screen = screenMixerSpecLen
				m.prevScreen = screenMixerSpec
				return m, nil
			}
			m.appendSpec = false
			m.initDepthList(len(m.words))
			m.screen = screenMixerDepth
			m.prevScreen = screenMixerSpec
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m Model) updateMixerSpecLen(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			val := m.textInput.Value()
			n, _ := parseIntSafe(val, 4)
			if n < 1 || n > 8 {
				n = 4
			}
			m.maxSpecLen = n
			m.initDepthList(len(m.words))
			m.screen = screenMixerDepth
			m.prevScreen = screenMixerSpecLen
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m *Model) initDepthList(n int) {
	items := []list.Item{
		menuItem{"1-word only", "Single words: Apple, Banana"},
		menuItem{"2-word combos", "AppleBanana, BananaApple"},
		menuItem{"2-word and 3-word combos", "Recommended balance"},
		menuItem{fmt.Sprintf("All combos 1..%d words", n), "Maximum output"},
	}
	m.initList("Choose Combination Depth", items, 60)
}

func (m Model) updateMixerDepth(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			idx := m.list.Index()
			n := len(m.words)
			switch idx {
			case 0:
				m.sizes = []int{1}
			case 1:
				m.sizes = []int{2}
			case 2:
				m.sizes = []int{2, 3}
			case 3:
				m.sizes = make([]int, n)
				for i := 0; i < n; i++ {
					m.sizes[i] = i + 1
				}
			}
			m.screen = screenGenerating
			return m, m.startGenerate()
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) updateMixerWarning(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			val := strings.ToLower(m.textInput.Value())
			if val != "y" && val != "yes" {
				m.screen = screenMode
				m.initModeList()
				return m, nil
			}
			m.screen = screenGenerating
			return m, m.startGenerate()
		}
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m Model) updateNumericLength(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			val := m.textInput.Value()
			n, _ := parseIntSafe(val, 0)
			if n < 1 || n > 16 {
				m.err = "Length must be 1-16"
				return m, nil
			}
			m.numericLen = n
			m.screen = screenGenerating
			return m, m.startNumeric()
		}
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m Model) updateNumericWarning(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			val := strings.ToLower(m.textInput.Value())
			if val != "y" && val != "yes" {
				m.screen = screenMode
				m.initModeList()
				return m, nil
			}
			m.screen = screenGenerating
			return m, m.startNumeric()
		}
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m Model) startGenerate() tea.Cmd {
	words := m.words
	maxCap := m.maxCap
	casingIdxs := m.casingIdxs
	appendNums := m.appendNums
	numbers := m.numbers
	appendSpec := m.appendSpec
	specCombos := m.specCombos
	maxSpecLen := m.maxSpecLen
	sizes := m.sizes
	maxBytes := m.maxBytes
	outDir := m.outDir

	return func() tea.Msg {
		combos := mixer.AllCombos(words, sizes)
		cfg := &mixer.Config{
			Words:      words,
			MaxCap:     maxCap,
			CasingIdxs: casingIdxs,
			AppendNums: appendNums,
			Numbers:    numbers,
			AppendSpec: appendSpec,
			SpecCombos: specCombos,
			MaxSpecLen: maxSpecLen,
			Sizes:      sizes,
			Combos:     combos,
			MaxBytes:   maxBytes,
		}

		_ = os.MkdirAll(outDir, 0755)
		written, files, elapsed, err := cfg.GenerateWithProgress(outDir, func(w uint64, f int) {
		})
		capped := err != nil
		return generateMsg{written: written, files: files, elapsed: elapsed, outDir: outDir, err: err, capped: capped}
	}
}

func (m Model) startNumeric() tea.Cmd {
	n := m.numericLen
	maxBytes := m.maxBytes
	outDir := m.outDir

	return func() tea.Msg {
		_ = os.MkdirAll(outDir, 0755)
		_ = numeric.CountValid(n)
		fw := ui.NewFileWriter(outDir)
		start := time.Now()
		var written uint64
		var stopped bool
		total := numeric.CountValid(n)
		progressEvery := uint64(1000000)
		if total < 1000000 {
			progressEvery = total / 100
			if progressEvery == 0 {
				progressEvery = 1
			}
		}
		digits := make([]byte, n)
		var generate func(pos int, last1, last2 byte)
		generate = func(pos int, last1, last2 byte) {
			if stopped {
				return
			}
			if pos == n {
				if err := fw.WriteLine(string(digits)); err != nil {
					stopped = true
					return
				}
				written++
				if maxBytes > 0 && fw.CurSize > maxBytes {
					stopped = true
				}
				return
			}
			for d := byte(0); d < 10; d++ {
				if pos >= 2 && d == last1 && d == last2 {
					continue
				}
				digits[pos] = '0' + d
				generate(pos+1, d, last1)
				if stopped {
					return
				}
			}
		}
		generate(0, 0xFF, 0xFF)
		fw.Close()
		elapsed := time.Since(start)
		var err error
		if stopped {
			err = fmt.Errorf("stopped at size cap. Wrote %d of %d", written, total)
		}
		return generateMsg{written: written, files: fw.Files, elapsed: elapsed, outDir: outDir, err: err, capped: stopped}
	}
}

func parseIntSafe(s string, def int) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return def, nil
	}
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	if err != nil {
		return def, fmt.Errorf("invalid number: %s", s)
	}
	return n, nil
}

func readFile(path string) ([]byte, error) {
	return readFileImpl(path)
}