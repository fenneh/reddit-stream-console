package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/fenneh/reddit-stream-console/internal/config"
	"github.com/fenneh/reddit-stream-console/internal/reddit"
)

type mode int

const (
	modeMenu mode = iota
	modeThreadList
	modeComments
	modeURLInput
)

const refreshInterval = 5 * time.Second

var (
	headerStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220"))
	statusStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("102"))
	errorStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	menuTitle        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220"))
	menuItem         = lipgloss.NewStyle().Foreground(lipgloss.Color("151"))
	menuSelected     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	commentHeader    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	commentBodyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	commentTreeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	commentAuthor    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229"))
	commentScore     = lipgloss.NewStyle().Foreground(lipgloss.Color("151"))
	commentTime      = lipgloss.NewStyle().Foreground(lipgloss.Color("110"))
)

type Model struct {
	mode            mode
	menu            list.Model
	threads         list.Model
	menuItems       []config.MenuItem
	currentMenu     *config.MenuItem
	threadsData     []reddit.Thread
	comments        []reddit.Comment
	commentFilter   string
	filterActive    bool
	filterInput     textinput.Model
	urlInput        textinput.Model
	viewport        viewport.Model
	width           int
	height          int
	panelWidth      int
	panelHeight     int
	innerWidth      int
	innerHeight     int
	status          string
	err             string
	userScrolled    bool
	client          *reddit.Client
	currentThread   *reddit.Thread
	refreshEnabled  bool
	loadingComments bool
}

func NewModel(menuItems []config.MenuItem, client *reddit.Client) Model {
	menuDelegate := list.NewDefaultDelegate()
	menuDelegate.Styles.SelectedTitle = menuSelected
	menuDelegate.Styles.SelectedDesc = menuSelected
	menuDelegate.Styles.NormalTitle = menuItem
	menuDelegate.Styles.NormalDesc = menuItem
	menuList := list.New(menuItemsToItems(menuItems), menuDelegate, 0, 0)
	menuList.Title = ""
	menuList.Styles.Title = menuTitle
	menuList.SetShowPagination(false)
	menuList.SetShowHelp(false)
	menuList.SetShowStatusBar(false)
	menuList.SetFilteringEnabled(false)

	threadDelegate := list.NewDefaultDelegate()
	threadDelegate.Styles.SelectedTitle = menuSelected
	threadDelegate.Styles.SelectedDesc = menuSelected
	threadDelegate.Styles.NormalTitle = menuItem
	threadDelegate.Styles.NormalDesc = menuItem
	threadList := list.New([]list.Item{}, threadDelegate, 0, 0)
	threadList.Title = ""
	threadList.Styles.Title = menuTitle
	threadList.SetShowPagination(false)
	threadList.SetShowHelp(false)
	threadList.SetShowStatusBar(false)
	threadList.SetFilteringEnabled(false)

	filterInput := textinput.New()
	filterInput.Placeholder = "filter comments"
	filterInput.Prompt = "/ "

	urlInput := textinput.New()
	urlInput.Placeholder = "https://reddit.com/r/..."
	urlInput.Prompt = "> "

	vp := viewport.New(0, 0)
	vp.HighPerformanceRendering = false

	return Model{
		mode:        modeMenu,
		menu:        menuList,
		threads:     threadList,
		menuItems:   menuItems,
		filterInput: filterInput,
		urlInput:    urlInput,
		viewport:    vp,
		client:      client,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

type threadsLoadedMsg struct {
	threads  []reddit.Thread
	err      error
	menuItem config.MenuItem
}

type commentsLoadedMsg struct {
	comments []reddit.Comment
	title    string
	err      error
}

type refreshTickMsg struct{}

type urlThreadMsg struct {
	thread reddit.Thread
	err    error
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		var handled bool
		var cmd tea.Cmd
		m, cmd, handled = m.handleKey(keyMsg)
		if handled {
			return m, cmd
		}
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		m.updateViewport()
		return m, nil
	case tea.KeyMsg:
		// allow list and input widgets to handle navigation
	case threadsLoadedMsg:
		m.status = ""
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		if len(msg.threads) == 0 {
			m.err = fmt.Sprintf("no threads found for %s", msg.menuItem.Title)
			m.mode = modeMenu
			return m, nil
		}
		m.err = ""
		m.currentMenu = &msg.menuItem
		m.threadsData = msg.threads
		m.threads.SetItems(threadsToItems(msg.threads))
		m.mode = modeThreadList
		m.threads.Title = msg.menuItem.Title
		return m, nil
	case commentsLoadedMsg:
		m.loadingComments = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.err = ""
		m.comments = msg.comments
		m.updateViewport()
		if !m.userScrolled {
			m.viewport.GotoBottom()
		}
		if msg.title != "" && m.currentThread != nil {
			m.currentThread.Title = msg.title
		}
		return m, nil
	case refreshTickMsg:
		if m.mode == modeComments && m.refreshEnabled {
			return m, tea.Batch(refreshTickCmd(), fetchCommentsCmd(m.client, m.currentThread))
		}
		return m, nil
	case urlThreadMsg:
		m.status = ""
		if msg.err != nil {
			m.err = msg.err.Error()
			m.mode = modeMenu
			return m, nil
		}
		m.err = ""
		m.currentThread = &msg.thread
		m.mode = modeComments
		m.refreshEnabled = true
		m.userScrolled = false
		m.loadingComments = true
		return m, tea.Batch(fetchCommentsCmd(m.client, m.currentThread), refreshTickCmd())
	}

	var cmd tea.Cmd
	switch m.mode {
	case modeMenu:
		m.menu, cmd = m.menu.Update(msg)
	case modeThreadList:
		m.threads, cmd = m.threads.Update(msg)
	case modeComments:
		if m.filterActive {
			m.filterInput, cmd = m.filterInput.Update(msg)
			m.commentFilter = m.filterInput.Value()
			m.updateViewport()
		}
	case modeURLInput:
		m.urlInput, cmd = m.urlInput.Update(msg)
	}

	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	if len(cmds) == 0 {
		return m, nil
	}
	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	header := lipgloss.NewStyle().Width(m.width).Padding(0, 1).Render(headerStyle.Render(m.headerTitle()))
	body := ""

	switch m.mode {
	case modeMenu:
		body = m.menuView()
	case modeThreadList:
		body = m.threadListView()
	case modeURLInput:
		content := fmt.Sprintf("Enter Reddit URL\n\n%s\n\n[enter] submit  [esc] cancel", m.urlInput.View())
		body = content
	case modeComments:
		content := m.viewport.View()
		if m.filterActive {
			content = content + "\n" + m.filterInput.View()
		}
		body = content
	}

	footer := lipgloss.NewStyle().Width(m.width).Padding(0, 1).Render(m.footerView())
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m *Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	switch msg.String() {
	case "ctrl+c", "q":
		return *m, tea.Quit, true
	}

	switch m.mode {
	case modeMenu:
		if msg.String() == "enter" {
			item := m.menu.SelectedItem()
			menuItem, ok := item.(menuItemItem)
			if !ok {
				return *m, nil, true
			}
			if menuItem.item.Type == "separator" {
				return *m, nil, true
			}
			if menuItem.item.Type == "url_input" {
				m.mode = modeURLInput
				m.urlInput.SetValue("")
				m.urlInput.Focus()
				return *m, nil, true
			}
			m.status = fmt.Sprintf("Loading %s...", menuItem.item.Title)
			m.err = ""
			return *m, fetchThreadsCmd(m.client, menuItem.item), true
		}
	case modeThreadList:
		switch msg.String() {
		case "enter":
			item := m.threads.SelectedItem()
			threadItem, ok := item.(threadItem)
			if !ok {
				return *m, nil, true
			}
			m.currentThread = &threadItem.thread
			m.mode = modeComments
			m.refreshEnabled = true
			m.userScrolled = false
			m.loadingComments = true
			m.commentFilter = ""
			m.filterActive = false
			m.filterInput.SetValue("")
			m.updateViewport()
			return *m, tea.Batch(fetchCommentsCmd(m.client, m.currentThread), refreshTickCmd()), true
		case "backspace":
			m.mode = modeMenu
			m.currentMenu = nil
			return *m, nil, true
		case "esc":
			m.mode = modeMenu
			m.currentMenu = nil
			return *m, nil, true
		}
	case modeComments:
		return m.handleCommentsKeys(msg)
	case modeURLInput:
		switch msg.String() {
		case "enter":
			url := strings.TrimSpace(m.urlInput.Value())
			if url == "" {
				m.mode = modeMenu
				return *m, nil, true
			}
			m.status = "Loading thread..."
			m.err = ""
			return *m, fetchThreadFromURLCmd(m.client, url), true
		case "esc":
			m.mode = modeMenu
			return *m, nil, true
		}
	}

	return *m, nil, false
}

func (m *Model) handleCommentsKeys(msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	if m.filterActive {
		switch msg.String() {
		case "esc":
			m.filterActive = false
			m.filterInput.SetValue("")
			m.commentFilter = ""
			m.updateViewport()
			return *m, nil, true
		case "enter":
			if strings.TrimSpace(m.filterInput.Value()) == "" {
				m.filterActive = false
				m.filterInput.SetValue("")
				m.commentFilter = ""
				m.updateViewport()
				return *m, nil, true
			}
		}
	}

	switch msg.String() {
	case "r":
		if m.currentThread != nil {
			m.loadingComments = true
			return *m, fetchCommentsCmd(m.client, m.currentThread), true
		}
	case "esc":
		m.mode = modeMenu
		m.currentThread = nil
		m.refreshEnabled = false
		return *m, nil, true
	case "backspace":
		if m.filterActive {
			return *m, nil, true
		}
		m.mode = modeThreadList
		m.currentThread = nil
		m.refreshEnabled = false
		return *m, nil, true
	case "end":
		m.userScrolled = false
		m.viewport.GotoBottom()
		return *m, nil, true
	case "/":
		m.filterActive = !m.filterActive
		if m.filterActive {
			m.filterInput.Focus()
		} else {
			m.filterInput.Blur()
			m.filterInput.SetValue("")
			m.commentFilter = ""
			m.updateViewport()
		}
		m.resize()
		return *m, nil, true
	case "up", "k":
		m.viewport.LineUp(1)
		m.userScrolled = true
		return *m, nil, true
	case "down", "j":
		m.viewport.LineDown(1)
		if m.viewport.AtBottom() {
			m.userScrolled = false
		}
		return *m, nil, true
	case "pgup":
		m.viewport.ViewUp()
		m.userScrolled = true
		return *m, nil, true
	case "pgdown":
		m.viewport.ViewDown()
		if m.viewport.AtBottom() {
			m.userScrolled = false
		}
		return *m, nil, true
	}

	return *m, nil, false
}

func (m *Model) resize() {
	headerHeight := 1
	footerHeight := 1

	m.panelHeight = m.height - headerHeight - footerHeight
	if m.panelHeight < 0 {
		m.panelHeight = 0
	}
	m.panelWidth = m.width
	if m.panelWidth < 0 {
		m.panelWidth = 0
	}

	m.innerWidth = m.panelWidth
	m.innerHeight = m.panelHeight
	if m.innerWidth < 0 {
		m.innerWidth = 0
	}
	if m.innerHeight < 0 {
		m.innerHeight = 0
	}

	viewportHeight := m.innerHeight
	if m.mode == modeComments && m.filterActive {
		viewportHeight--
	}
	if viewportHeight < 0 {
		viewportHeight = 0
	}

	m.menu.SetSize(m.innerWidth, m.innerHeight)
	m.threads.SetSize(m.innerWidth, m.innerHeight)
	m.viewport.Width = m.innerWidth
	m.viewport.Height = viewportHeight
	m.filterInput.Width = m.innerWidth
	m.urlInput.Width = m.innerWidth
	if m.mode == modeComments {
		m.updateViewport()
	}
}

func (m *Model) updateViewport() {
	if m.viewport.Width == 0 {
		return
	}
	content := renderComments(m.comments, m.viewport.Width, m.commentFilter)
	m.viewport.SetContent(content)
}

func (m *Model) bodyHeight() int {
	return m.panelHeight
}

func (m *Model) menuView() string {
	title := menuTitle.Render("Reddit Stream Console")
	subtitle := statusStyle.Render("Live comment stream, zero fluff.")
	divider := strings.Repeat("-", max(0, m.width))
	return lipgloss.JoinVertical(lipgloss.Left, title, subtitle, divider, m.menu.View())
}

func (m *Model) threadListView() string {
	title := menuTitle.Render("Threads")
	if m.currentMenu != nil && m.currentMenu.Title != "" {
		title = menuTitle.Render(m.currentMenu.Title)
	}
	divider := strings.Repeat("-", max(0, m.width))
	return lipgloss.JoinVertical(lipgloss.Left, title, divider, m.threads.View())
}

func (m *Model) footerView() string {
	if m.err != "" {
		return errorStyle.Render(m.err)
	}
	if m.status != "" {
		return statusStyle.Render(m.status)
	}

	switch m.mode {
	case modeMenu:
		return statusStyle.Render("[enter] select  [q] quit")
	case modeThreadList:
		return statusStyle.Render("[enter] open  [backspace] menu  [q] quit")
	case modeURLInput:
		return statusStyle.Render("[enter] submit  [esc] back  [q] quit")
	case modeComments:
		if m.loadingComments {
			return statusStyle.Render("loading comments...")
		}
		return statusStyle.Render("[/] filter  [r] refresh  [end] bottom  [backspace] back  [esc] menu  [q] quit")
	}

	return ""
}

func (m *Model) headerTitle() string {
	switch m.mode {
	case modeThreadList:
		if m.currentMenu != nil {
			return m.currentMenu.Title
		}
	case modeComments:
		if m.currentThread != nil {
			return m.currentThread.Title
		}
	}
	return "Reddit Stream Console"
}

func menuItemsToItems(items []config.MenuItem) []list.Item {
	out := make([]list.Item, 0, len(items))
	for _, item := range items {
		out = append(out, menuItemItem{item: item})
	}
	return out
}

func threadsToItems(threads []reddit.Thread) []list.Item {
	out := make([]list.Item, 0, len(threads))
	for _, t := range threads {
		out = append(out, threadItem{thread: t})
	}
	return out
}

type menuItemItem struct {
	item config.MenuItem
}

func (m menuItemItem) Title() string {
	return m.item.Title
}

func (m menuItemItem) Description() string {
	return m.item.Description
}

func (m menuItemItem) FilterValue() string {
	return m.item.Title
}

type threadItem struct {
	thread reddit.Thread
}

func (t threadItem) Title() string {
	return t.thread.Title
}

func (t threadItem) Description() string {
	return ""
}

func (t threadItem) FilterValue() string {
	return t.thread.Title
}

func fetchThreadsCmd(client *reddit.Client, item config.MenuItem) tea.Cmd {
	maxAge := item.MaxAgeHours
	if maxAge == 0 {
		maxAge = 24
	}
	limit := item.Limit
	if limit == 0 {
		limit = 50
	}
	query := reddit.ThreadQuery{
		Type:                item.Type,
		Subreddit:           item.Subreddit,
		Flairs:              item.Flair,
		MaxAgeHours:         maxAge,
		Limit:               limit,
		TitleMustContain:    item.TitleMustContain,
		TitleMustNotContain: item.TitleMustNotContain,
	}
	return func() tea.Msg {
		threads, err := client.FindThreads(query)
		return threadsLoadedMsg{threads: threads, err: err, menuItem: item}
	}
}

func fetchCommentsCmd(client *reddit.Client, thread *reddit.Thread) tea.Cmd {
	if thread == nil {
		return nil
	}
	return func() tea.Msg {
		comments, title, err := client.FetchComments(thread.Permalink)
		return commentsLoadedMsg{comments: comments, title: title, err: err}
	}
}

func fetchThreadFromURLCmd(client *reddit.Client, url string) tea.Cmd {
	return func() tea.Msg {
		thread, err := client.ThreadFromURL(url)
		return urlThreadMsg{thread: thread, err: err}
	}
}

func refreshTickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(time.Time) tea.Msg {
		return refreshTickMsg{}
	})
}

func renderComments(comments []reddit.Comment, width int, filter string) string {
	if width <= 0 {
		return ""
	}
	var b strings.Builder
	filterLower := strings.ToLower(strings.TrimSpace(filter))

	roots := buildCommentTree(comments, filterLower)
	var walk func(nodes []*commentNode, hasNext []bool)
	walk = func(nodes []*commentNode, hasNext []bool) {
		for i, node := range nodes {
			isLast := i == len(nodes)-1
			headerPrefix := headerPrefix(hasNext, isLast)
			bodyPrefix := bodyPrefix(hasNext, isLast)

			header := formatHeader(node.comment)
			for _, line := range wrapWithPrefix(header, width, headerPrefix) {
				b.WriteString(commentTreeStyle.Render(headerPrefix))
				b.WriteString(strings.TrimPrefix(line, headerPrefix))
				b.WriteString("\n")
			}
			for _, line := range wrapWithPrefix(node.comment.Body, width, bodyPrefix) {
				b.WriteString(commentTreeStyle.Render(bodyPrefix))
				b.WriteString(commentBodyStyle.Render(strings.TrimPrefix(line, bodyPrefix)))
				b.WriteString("\n")
			}
			b.WriteString("\n")

			if len(node.children) > 0 {
				walk(node.children, append(hasNext, !isLast))
			}
		}
	}
	walk(roots, nil)
	return b.String()
}

func formatHeader(comment reddit.Comment) string {
	author := commentAuthor.Render(comment.Author)
	score := commentScore.Render(fmt.Sprintf("%d points", comment.Score))
	timeText := commentTime.Render(comment.FormattedTime)
	separator := commentTreeStyle.Render(" • ")
	return author + separator + score + separator + timeText
}

func wrapWithPrefix(text string, width int, prefix string) []string {
	if width <= 0 {
		return []string{prefix + text}
	}
	available := width - len(prefix)
	if available <= 0 {
		return []string{prefix + text}
	}

	lines := make([]string, 0, 8)
	paragraphs := strings.Split(text, "\n")
	for _, paragraph := range paragraphs {
		words := ansiFields(paragraph)
		if len(words) == 0 {
			lines = append(lines, prefix)
			continue
		}
		line := words[0]
		for _, word := range words[1:] {
			if lipgloss.Width(line)+1+lipgloss.Width(word) > available {
				lines = append(lines, prefix+line)
				line = word
				continue
			}
			line = line + " " + word
		}
		lines = append(lines, prefix+line)
	}
	return lines
}

type commentNode struct {
	comment  reddit.Comment
	children []*commentNode
}

func buildCommentTree(comments []reddit.Comment, filterLower string) []*commentNode {
	nodes := make(map[string]*commentNode, len(comments))
	order := make([]*commentNode, 0, len(comments))

	for _, c := range comments {
		if filterLower != "" {
			author := strings.ToLower(c.Author)
			body := strings.ToLower(c.Body)
			if !strings.Contains(author, filterLower) && !strings.Contains(body, filterLower) {
				continue
			}
		}
		node := &commentNode{comment: c}
		nodes[c.ID] = node
		order = append(order, node)
	}

	roots := make([]*commentNode, 0, len(order))
	for _, node := range order {
		parentID := strings.TrimSpace(node.comment.ParentID)
		if parentID == "" {
			roots = append(roots, node)
			continue
		}
		parent, ok := nodes[parentID]
		if !ok {
			roots = append(roots, node)
			continue
		}
		parent.children = append(parent.children, node)
	}
	return roots
}

func headerPrefix(hasNext []bool, isLast bool) string {
	var b strings.Builder
	for _, sibling := range hasNext {
		if sibling {
			b.WriteString("|  ")
		} else {
			b.WriteString("   ")
		}
	}
	if isLast {
		b.WriteString("`- ")
	} else {
		b.WriteString("+- ")
	}
	return b.String()
}

func bodyPrefix(hasNext []bool, isLast bool) string {
	var b strings.Builder
	for _, sibling := range hasNext {
		if sibling {
			b.WriteString("|  ")
		} else {
			b.WriteString("   ")
		}
	}
	if isLast {
		b.WriteString("   ")
	} else {
		b.WriteString("|  ")
	}
	return b.String()
}

func ansiFields(text string) []string {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	return strings.Fields(text)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
