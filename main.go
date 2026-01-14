package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Key bindings
type keyMap struct {
	PlayPause key.Binding
	Prev      key.Binding
	Next      key.Binding
	Faster    key.Binding
	Slower    key.Binding
	JumpBack  key.Binding
	JumpFwd   key.Binding
	Restart   key.Binding
	OpenFile  key.Binding
	Quit      key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.PlayPause, k.Prev, k.Next, k.Faster, k.Slower}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.PlayPause, k.Prev, k.Next},
		{k.Faster, k.Slower, k.Restart},
		{k.JumpBack, k.JumpFwd, k.OpenFile},
	}
}

// Filepicker key bindings
type fpKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Open   key.Binding
	Back   key.Binding
	Select key.Binding
	Cancel key.Binding
}

func (k fpKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Open, k.Back, k.Select, k.Cancel}
}

func (k fpKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Open},
		{k.Back, k.Select, k.Cancel},
	}
}

var fpKeys = fpKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Open: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "open dir"),
	),
	Back: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "back"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc", "q"),
		key.WithHelp("esc", "cancel"),
	),
}

var keys = keyMap{
	PlayPause: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "play/pause"),
	),
	Prev: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "prev word"),
	),
	Next: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "next word"),
	),
	Faster: key.NewBinding(
		key.WithKeys("up", "k", "+", "="),
		key.WithHelp("↑/k", "faster"),
	),
	Slower: key.NewBinding(
		key.WithKeys("down", "j", "-", "_"),
		key.WithHelp("↓/j", "slower"),
	),
	JumpBack: key.NewBinding(
		key.WithKeys("["),
		key.WithHelp("[", "-10 words"),
	),
	JumpFwd: key.NewBinding(
		key.WithKeys("]"),
		key.WithHelp("]", "+10 words"),
	),
	Restart: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "restart"),
	),
	OpenFile: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open file"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

// ORP (Optimal Recognition Point) calculation
func calculateORP(word string) int {
	length := utf8.RuneCountInString(word)
	switch {
	case length <= 1:
		return 0
	case length <= 5:
		return 1
	case length <= 9:
		return 2
	case length <= 13:
		return 3
	default:
		return 4
	}
}

// Tokenize splits text into words
func tokenize(text string) []string {
	fields := strings.Fields(text)
	var words []string
	for _, f := range fields {
		if f != "" {
			words = append(words, f)
		}
	}
	return words
}

// sanitizeHTML extracts text content from HTML using html-to-markdown
func sanitizeHTML(htmlContent []byte) string {
	md, err := htmltomarkdown.ConvertString(string(htmlContent))
	if err != nil {
		return string(htmlContent)
	}

	return md
}

// fetchURL fetches content from a URL with a timeout
func fetchURL(urlStr string) ([]byte, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(context.Background(), "GET", urlStr, nil)
	if err != nil {
		return nil, err
	}

	// Set user agent to avoid being blocked by some servers
	req.Header.Set("User-Agent", "skim/1.0 (+https://github.com/varunrandery/skim)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	return io.ReadAll(resp.Body)
}

// isURL checks if a string is a valid URL
func isURL(str string) bool {
	_, err := url.ParseRequestURI(str)
	if err != nil {
		return false
	}

	u, err := url.Parse(str)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}

	return true
}

// isBinaryFile checks if content appears to be binary by looking for null bytes
func isBinaryFile(content []byte) bool {
	checkSize := min(8192, len(content))
	for i := 0; i < checkSize; i++ {
		if content[i] == 0 {
			return true
		}
	}
	return false
}

func truncateWord(word string) string {
	if utf8.RuneCountInString(word) <= 32 {
		return word
	}
	runes := []rune(word)
	return string(runes[:31]) + "..."
}

var textFileExtensions = []string{
	".txt", ".md", ".markdown",
	".go", ".js", ".ts", ".jsx", ".tsx",
	".py", ".rb", ".rs", ".c", ".h", ".cpp", ".hpp",
	".java", ".kt", ".swift", ".cs",
	".html", ".css", ".scss", ".sass", ".less",
	".json", ".yaml", ".yml", ".toml", ".xml",
	".sh", ".bash", ".zsh", ".fish",
	".sql", ".graphql",
	".vim", ".lua", ".el", ".lisp", ".clj",
	".r", ".R", ".jl",
	".tex", ".org", ".rst", ".adoc",
	".conf", ".cfg", ".ini", ".env",
	".gitignore", ".dockerignore", ".editorconfig",
}

type tickMsg time.Time

type model struct {
	words        []string
	currentIdx   int
	wpm          int
	paused       bool
	width        int
	height       int
	quit         bool
	focusCol     int
	help         help.Model
	keys         keyMap
	progress     progress.Model
	filepicker   filepicker.Model
	showPicker   bool
	selectedFile string
	fileError    string
}

func initialModel(words []string, wpm int) model {
	h := help.New()
	h.ShowAll = true

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)

	fp := filepicker.New()
	fp.CurrentDirectory, _ = os.Getwd()
	fp.ShowHidden = false
	fp.AllowedTypes = textFileExtensions

	return model{
		words:      words,
		currentIdx: 0,
		wpm:        wpm,
		paused:     true,
		focusCol:   40,
		help:       h,
		keys:       keys,
		progress:   p,
		filepicker: fp,
		showPicker: len(words) == 0,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tickCmd(m.wpm), tea.EnterAltScreen, m.filepicker.Init())
}

func tickCmd(wpm int) tea.Cmd {
	interval := time.Minute / time.Duration(wpm)
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
		m.height = msg.Height
		m.focusCol = msg.Width / 2
		m.help.Width = msg.Width
		m.filepicker.SetHeight(min(20, msg.Height-15))
	}

	if m.showPicker {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc", "q":
				m.showPicker = false
				return m, nil
			}
		}

		var cmd tea.Cmd
		m.filepicker, cmd = m.filepicker.Update(msg)

		if didSelect, path := m.filepicker.DidSelectFile(msg); didSelect {
			content, err := os.ReadFile(path)
			if err != nil {
				m.fileError = "Error reading file"
			} else if isBinaryFile(content) {
				m.fileError = "Cannot open binary file"
			} else {
				words := tokenize(string(content))
				if len(words) > 0 {
					m.words = words
					m.currentIdx = 0
					m.paused = true
					m.selectedFile = path
					m.fileError = ""
				} else {
					m.fileError = "No words found in file"
				}
			}
			m.showPicker = false
			return m, nil
		}

		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.quit = true
			return m, tea.Quit

		case key.Matches(msg, m.keys.OpenFile):
			m.showPicker = true
			m.paused = true
			m.filepicker = filepicker.New()
			m.filepicker.CurrentDirectory, _ = os.Getwd()
			m.filepicker.ShowHidden = false
			m.filepicker.AllowedTypes = textFileExtensions
			if m.height > 0 {
				m.filepicker.SetHeight(m.height - 15)
			}
			return m, m.filepicker.Init()

		case key.Matches(msg, m.keys.PlayPause):
			m.paused = !m.paused
			if !m.paused {
				return m, tickCmd(m.wpm)
			}
			return m, nil

		case key.Matches(msg, m.keys.Prev):
			if m.currentIdx > 0 {
				m.currentIdx--
			}
			return m, nil

		case key.Matches(msg, m.keys.Next):
			if m.currentIdx < len(m.words)-1 {
				m.currentIdx++
			}
			return m, nil

		case key.Matches(msg, m.keys.Faster):
			m.wpm += 25
			if m.wpm > 1000 {
				m.wpm = 1000
			}
			return m, nil

		case key.Matches(msg, m.keys.Slower):
			m.wpm -= 25
			if m.wpm < 50 {
				m.wpm = 50
			}
			return m, nil

		case key.Matches(msg, m.keys.JumpBack):
			m.currentIdx -= 10
			if m.currentIdx < 0 {
				m.currentIdx = 0
			}
			return m, nil

		case key.Matches(msg, m.keys.JumpFwd):
			m.currentIdx += 10
			if m.currentIdx >= len(m.words) {
				m.currentIdx = len(m.words) - 1
			}
			return m, nil

		case key.Matches(msg, m.keys.Restart):
			m.currentIdx = 0
			m.paused = true
			return m, nil
		}

	case tickMsg:
		if !m.paused && m.currentIdx < len(m.words)-1 {
			m.currentIdx++
			return m, tickCmd(m.wpm)
		} else if m.currentIdx >= len(m.words)-1 {
			m.paused = true
		}
		if !m.paused {
			return m, tickCmd(m.wpm)
		}

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	if m.quit {
		return ""
	}

	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	if m.showPicker {
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))

		title := titleStyle.Render("Select a file to open")
		titleLine := strings.Repeat(" ", max(0, (m.width-lipgloss.Width(title))/2)) + title

		pickerHeight := m.height - 10
		pickerStyle := lipgloss.NewStyle().Height(pickerHeight).MaxHeight(pickerHeight)
		picker := pickerStyle.Render(m.filepicker.View())

		fpHelp := m.help.View(fpKeys)
		var helpLines strings.Builder
		for line := range strings.SplitSeq(fpHelp, "\n") {
			lineWidth := lipgloss.Width(line)
			helpLines.WriteString(strings.Repeat(" ", max(0, (m.width-lineWidth)/2)) + line + "\n")
		}

		return titleLine + "\n\n" + picker + "\n\n\n\n" + helpLines.String()
	}

	if len(m.words) == 0 {
		if m.fileError != "" {
			return m.fileError + ". Press 'o' to open a text file or provide a URL as an argument."
		}
		return "No words to display. Press 'o' to open a text file or provide a URL as an argument."
	}

	word := m.words[m.currentIdx]
	// Truncate long words to prevent UI overflow
	truncatedWord := truncateWord(word)

	orpIdx := calculateORP(truncatedWord)
	runes := []rune(truncatedWord)

	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	highlightStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	contextStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	halfWidth := 30 // chars on each side of ORP
	wordLen := utf8.RuneCountInString(word)
	charsBeforeORP := orpIdx
	charsAfterORP := wordLen - orpIdx

	beforeSectionWidth := max(0, halfWidth-charsBeforeORP)
	var beforeBuilder strings.Builder
	for i := 0; i < m.currentIdx; i++ {
		beforeBuilder.WriteString(m.words[i] + " ")
	}
	beforeStr := beforeBuilder.String()
	beforeRunes := []rune(beforeStr)
	var contextBefore string
	if len(beforeRunes) > beforeSectionWidth {
		contextBefore = string(beforeRunes[len(beforeRunes)-beforeSectionWidth:])
	} else if beforeSectionWidth > 0 {
		contextBefore = strings.Repeat(" ", beforeSectionWidth-len(beforeRunes)) + beforeStr
	}
	contextBeforeRendered := contextStyle.Render(contextBefore)

	var wordParts []string
	for i, r := range runes {
		if i == orpIdx {
			wordParts = append(wordParts, highlightStyle.Render(string(r)))
		} else {
			wordParts = append(wordParts, normalStyle.Render(string(r)))
		}
	}
	renderedWord := strings.Join(wordParts, "")

	afterSectionWidth := max(0, halfWidth-charsAfterORP)
	var afterBuilder strings.Builder
	for i := m.currentIdx + 1; i < len(m.words) && afterBuilder.Len() < afterSectionWidth+20; i++ {
		afterBuilder.WriteString(" " + m.words[i])
	}
	afterStr := afterBuilder.String()
	afterRunes := []rune(afterStr)
	var contextAfter string
	if len(afterRunes) > afterSectionWidth {
		contextAfter = string(afterRunes[:afterSectionWidth])
	} else if afterSectionWidth > 0 {
		contextAfter = afterStr + strings.Repeat(" ", afterSectionWidth-len(afterRunes))
	}
	contextAfterRendered := contextStyle.Render(contextAfter)

	leftPadding := max(0, m.focusCol-halfWidth)

	focusLine := strings.Repeat(" ", m.focusCol) + dimStyle.Render("│")

	wordLine := strings.Repeat(" ", leftPadding) + contextBeforeRendered + renderedWord + contextAfterRendered

	progressPercent := float64(m.currentIdx+1) / float64(len(m.words))
	wordsRemaining := len(m.words) - m.currentIdx - 1
	timeRemaining := time.Duration(wordsRemaining) * time.Minute / time.Duration(m.wpm)

	statusLine := statusStyle.Render(fmt.Sprintf(
		"%d WPM │ ~%s remaining",
		m.wpm,
		formatDuration(timeRemaining),
	))

	progressBar := m.progress.ViewAs(progressPercent)

	helpView := m.help.View(m.keys)

	bottomSectionHeight := 8
	wordRowY := m.height/2 - 1

	var output strings.Builder

	output.WriteString(strings.Repeat("\n", max(0, wordRowY-1)))
	output.WriteString(focusLine + "\n")
	output.WriteString(wordLine + "\n")

	gapHeight := m.height - wordRowY - 2 - bottomSectionHeight
	output.WriteString(strings.Repeat("\n", max(0, gapHeight)))

	progressWidth := lipgloss.Width(progressBar)
	output.WriteString(strings.Repeat(" ", max(0, (m.width-progressWidth)/2)) + progressBar + "\n")
	output.WriteString("\n")

	output.WriteString(strings.Repeat(" ", max(0, (m.width-lipgloss.Width(statusLine))/2)) + statusLine + "\n")
	output.WriteString("\n")

	for line := range strings.SplitSeq(helpView, "\n") {
		lineWidth := lipgloss.Width(line)
		output.WriteString(strings.Repeat(" ", max(0, (m.width-lineWidth)/2)) + line + "\n")
	}

	return output.String()
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}

func main() {
	wpm := flag.Int("wpm", 500, "Words per minute (50-1000)")
	flag.Parse()

	if *wpm < 50 {
		*wpm = 50
	} else if *wpm > 1000 {
		*wpm = 1000
	}

	var words []string
	args := flag.Args()

	// Check if stdin has piped data
	stdinInfo, _ := os.Stdin.Stat()
	hasStdin := (stdinInfo.Mode() & os.ModeCharDevice) == 0

	if hasStdin {
		// Read from stdin
		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
			os.Exit(1)
		}
		if isBinaryFile(content) {
			fmt.Fprintln(os.Stderr, "Cannot read binary content from stdin")
			os.Exit(1)
		}
		words = tokenize(string(content))
		if len(words) == 0 {
			fmt.Fprintln(os.Stderr, "No words found in stdin")
			os.Exit(1)
		}
	} else if len(args) >= 1 {
		source := args[0]

		// Check if the source is a URL
		if isURL(source) {
			fmt.Printf("Fetching content from URL: %s\n", source)
			content, err := fetchURL(source)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error fetching URL: %v\n", err)
				os.Exit(1)
			}

			sanitizedContent := sanitizeHTML(content)
			words = tokenize(sanitizedContent)

			if len(words) == 0 {
				fmt.Fprintln(os.Stderr, "No words found in URL content")
				os.Exit(1)
			}
		} else {
			// Treat as a file path
			filePath := source
			content, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
				os.Exit(1)
			}
			if isBinaryFile(content) {
				fmt.Fprintln(os.Stderr, "Cannot open binary file")
				os.Exit(1)
			}
			words = tokenize(string(content))
			if len(words) == 0 {
				fmt.Fprintln(os.Stderr, "No words found in file")
				os.Exit(1)
			}
		}
	}

	// Set up program options
	opts := []tea.ProgramOption{tea.WithAltScreen()}

	// If stdin was used for content, we need to reopen /dev/tty for keyboard input
	if hasStdin {
		tty, err := os.Open("/dev/tty")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening /dev/tty for input: %v\n", err)
			os.Exit(1)
		}
		defer tty.Close()
		opts = append(opts, tea.WithInput(tty))
	}

	p := tea.NewProgram(initialModel(words, *wpm), opts...)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
