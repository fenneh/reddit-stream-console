package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/fenneh/reddit-stream-console/internal/config"
	"github.com/fenneh/reddit-stream-console/internal/reddit"
)

// Colors matching the original Python app
var (
	tealTview       = tcell.NewRGBColor(101, 146, 135)
	warmCreamTview  = tcell.NewRGBColor(255, 230, 169)
	warmOrangeTview = tcell.NewRGBColor(222, 170, 121)
	sageGreenTview  = tcell.NewRGBColor(177, 194, 158)
)

func init() {
	// Use single-line borders globally (both normal and focused)
	tview.Borders.Horizontal = '─'
	tview.Borders.Vertical = '│'
	tview.Borders.TopLeft = '┌'
	tview.Borders.TopRight = '┐'
	tview.Borders.BottomLeft = '└'
	tview.Borders.BottomRight = '┘'
	tview.Borders.HorizontalFocus = '─'
	tview.Borders.VerticalFocus = '│'
	tview.Borders.TopLeftFocus = '┌'
	tview.Borders.TopRightFocus = '┐'
	tview.Borders.BottomLeftFocus = '└'
	tview.Borders.BottomRightFocus = '┘'
}

type TviewApp struct {
	app          *tview.Application
	pages        *tview.Pages
	header       *tview.TextView
	menuView     *tview.TextView // Custom menu using TextView
	menuIndex    int             // Current menu selection
	threadView   *tview.TextView // Custom thread list using TextView
	threadIndex  int             // Current thread selection
	commentsView *tview.TextView
	urlInput     *tview.InputField
	filterInput  *tview.InputField
	statusBar    *tview.TextView
	mainFlex     *tview.Flex

	client        *reddit.Client
	menuItems     []config.MenuItem
	threadsData   []reddit.Thread
	comments      []reddit.Comment
	currentThread *reddit.Thread
	currentMenu   *config.MenuItem

	filterActive   bool
	commentFilter  string
	refreshEnabled bool
	stopRefresh    chan struct{}
}

func NewTviewApp(menuItems []config.MenuItem, client *reddit.Client) *TviewApp {
	ta := &TviewApp{
		app:         tview.NewApplication(),
		pages:       tview.NewPages(),
		menuItems:   menuItems,
		client:      client,
		stopRefresh: make(chan struct{}),
	}

	ta.setupUI()
	return ta
}

func (ta *TviewApp) setupUI() {
	// Header
	ta.header = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	ta.header.SetBackgroundColor(tealTview)
	ta.header.SetTextColor(warmCreamTview)

	// Custom menu using TextView for full control
	ta.menuView = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	ta.menuView.SetBackgroundColor(tcell.ColorDefault)
	ta.menuIndex = 0
	// Skip to first non-separator
	for ta.menuIndex < len(ta.menuItems) && ta.menuItems[ta.menuIndex].Type == "separator" {
		ta.menuIndex++
	}

	// Thread list - custom TextView like menu
	ta.threadView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetTextAlign(tview.AlignCenter)
	ta.threadView.SetBackgroundColor(tcell.ColorDefault)
	ta.threadIndex = 0

	// Comments view - this is the key component with built-in scrolling
	ta.commentsView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true).
		SetWordWrap(true)
	ta.commentsView.SetBackgroundColor(tcell.ColorDefault)
	ta.commentsView.SetBorder(true)
	ta.commentsView.SetBorderColor(tealTview)
	ta.commentsView.SetBorderPadding(0, 0, 1, 1)

	// URL input
	ta.urlInput = tview.NewInputField().
		SetLabel("URL: ").
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetFieldTextColor(warmCreamTview).
		SetLabelColor(warmCreamTview)

	// Filter input
	ta.filterInput = tview.NewInputField().
		SetLabel("/ ").
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetFieldTextColor(warmCreamTview).
		SetLabelColor(warmOrangeTview)

	// Status bar
	ta.statusBar = tview.NewTextView().
		SetDynamicColors(true)
	ta.statusBar.SetBackgroundColor(tealTview)
	ta.statusBar.SetTextColor(warmCreamTview)

	// Build pages
	ta.buildMenuPage()
	ta.buildThreadListPage()
	ta.buildCommentsPage()
	ta.buildURLInputPage()

	// Set up main layout
	ta.mainFlex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(ta.header, 1, 0, false).
		AddItem(ta.pages, 0, 1, true).
		AddItem(ta.statusBar, 1, 0, false)

	ta.app.SetRoot(ta.mainFlex, true)
	ta.showMenu()

	// Global key handler
	ta.app.SetInputCapture(ta.globalKeyHandler)
}

func (ta *TviewApp) buildMenuPage() {
	// Wrap menu in a flex for vertical centering
	menuFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(ta.menuView, 0, 2, true).
		AddItem(nil, 0, 1, false)
	menuFlex.SetBackgroundColor(tcell.ColorDefault)
	menuFlex.SetBorder(true)
	menuFlex.SetBorderColor(tealTview)
	ta.pages.AddPage("menu", menuFlex, true, false)
}

func (ta *TviewApp) renderMenu() {
	ta.menuView.Clear()

	var lines []string
	lines = append(lines, "") // Top padding

	for i, item := range ta.menuItems {
		if item.Type == "separator" {
			lines = append(lines, "")
			continue
		}

		if i == ta.menuIndex {
			// Selected: orange with arrow
			lines = append(lines, fmt.Sprintf("[#DEAA79::b]→ %s[-:-:-]", item.Title))
			if item.Description != "" {
				lines = append(lines, fmt.Sprintf("[#888888]  %s[-]", item.Description))
			}
		} else {
			// Normal: sage green
			lines = append(lines, fmt.Sprintf("[#B1C29E]  %s[-]", item.Title))
			if item.Description != "" {
				lines = append(lines, fmt.Sprintf("[#666666]  %s[-]", item.Description))
			}
		}
	}

	fmt.Fprint(ta.menuView, strings.Join(lines, "\n"))
}

func (ta *TviewApp) menuUp() {
	orig := ta.menuIndex
	for {
		ta.menuIndex--
		if ta.menuIndex < 0 {
			ta.menuIndex = len(ta.menuItems) - 1
		}
		if ta.menuIndex == orig {
			break // Wrapped around
		}
		if ta.menuItems[ta.menuIndex].Type != "separator" {
			break
		}
	}
	ta.renderMenu()
}

func (ta *TviewApp) menuDown() {
	orig := ta.menuIndex
	for {
		ta.menuIndex++
		if ta.menuIndex >= len(ta.menuItems) {
			ta.menuIndex = 0
		}
		if ta.menuIndex == orig {
			break // Wrapped around
		}
		if ta.menuItems[ta.menuIndex].Type != "separator" {
			break
		}
	}
	ta.renderMenu()
}

func (ta *TviewApp) buildThreadListPage() {
	// Center the thread list like the menu
	threadFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(ta.threadView, 0, 3, true).
		AddItem(nil, 0, 1, false)
	threadFlex.SetBackgroundColor(tcell.ColorDefault)
	threadFlex.SetBorder(true)
	threadFlex.SetBorderColor(tealTview)
	ta.pages.AddPage("threads", threadFlex, true, false)
}

func (ta *TviewApp) renderThreadList() {
	ta.threadView.Clear()

	if len(ta.threadsData) == 0 {
		fmt.Fprint(ta.threadView, "[#888888]No threads found[-]")
		return
	}

	var lines []string
	for i, thread := range ta.threadsData {
		if i == ta.threadIndex {
			// Selected: orange with arrow
			lines = append(lines, fmt.Sprintf("[#DEAA79::b]→ %s[-:-:-]", thread.Title))
		} else {
			// Normal: sage green
			lines = append(lines, fmt.Sprintf("[#B1C29E]  %s[-]", thread.Title))
		}
	}

	fmt.Fprint(ta.threadView, strings.Join(lines, "\n"))

	// Scroll to keep selection visible
	ta.threadView.ScrollTo(ta.threadIndex, 0)
}

func (ta *TviewApp) threadUp() {
	if len(ta.threadsData) == 0 {
		return
	}
	ta.threadIndex--
	if ta.threadIndex < 0 {
		ta.threadIndex = len(ta.threadsData) - 1
	}
	ta.renderThreadList()
}

func (ta *TviewApp) threadDown() {
	if len(ta.threadsData) == 0 {
		return
	}
	ta.threadIndex++
	if ta.threadIndex >= len(ta.threadsData) {
		ta.threadIndex = 0
	}
	ta.renderThreadList()
}

func (ta *TviewApp) buildCommentsPage() {
	commentsFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(ta.commentsView, 0, 1, true)
	ta.pages.AddPage("comments", commentsFlex, true, false)
}

func (ta *TviewApp) buildURLInputPage() {
	// Styled label
	label := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	label.SetBackgroundColor(tcell.ColorDefault)
	fmt.Fprint(label, "[#FFE6A9::b]Enter Reddit Thread URL[-:-:-]")

	// Style the input field
	ta.urlInput.SetBackgroundColor(tcell.ColorDefault)
	ta.urlInput.SetFieldBackgroundColor(tcell.NewRGBColor(40, 40, 40))
	ta.urlInput.SetFieldTextColor(warmCreamTview)
	ta.urlInput.SetLabelColor(warmOrangeTview)
	ta.urlInput.SetLabel("→ ")
	ta.urlInput.SetPlaceholder("https://reddit.com/r/...")
	ta.urlInput.SetPlaceholderTextColor(tcell.NewRGBColor(100, 100, 100))

	// Hint text
	hint := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	hint.SetBackgroundColor(tcell.ColorDefault)
	fmt.Fprint(hint, "[#888888]Press [#DEAA79]Enter[-] to submit  •  [#DEAA79]Esc[-] to go back[-]")

	// Center everything
	inputBox := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(nil, 0, 1, false).
		AddItem(ta.urlInput, 60, 0, true).
		AddItem(nil, 0, 1, false)

	// Inner content
	innerFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(label, 1, 0, false).
		AddItem(nil, 1, 0, false).
		AddItem(inputBox, 1, 0, true).
		AddItem(nil, 2, 0, false).
		AddItem(hint, 1, 0, false).
		AddItem(nil, 0, 1, false)
	innerFlex.SetBackgroundColor(tcell.ColorDefault)
	innerFlex.SetBorder(true)
	innerFlex.SetBorderColor(tealTview)

	// Wrap in flex for centering with some margin
	urlFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 1, 0, false).
		AddItem(innerFlex, 0, 1, true).
		AddItem(nil, 1, 0, false)
	urlFlex.SetBackgroundColor(tcell.ColorDefault)

	ta.pages.AddPage("url", urlFlex, true, false)
}

func (ta *TviewApp) globalKeyHandler(event *tcell.EventKey) *tcell.EventKey {
	// Get current page
	pageName, _ := ta.pages.GetFrontPage()

	// Don't intercept keys when in input fields
	if pageName == "url" || ta.filterActive {
		if event.Key() == tcell.KeyEscape {
			if ta.filterActive {
				ta.hideFilter()
				return nil
			}
			ta.showMenu()
			return nil
		}
		return event
	}

	// Menu page navigation
	if pageName == "menu" {
		switch event.Key() {
		case tcell.KeyUp:
			ta.menuUp()
			return nil
		case tcell.KeyDown:
			ta.menuDown()
			return nil
		case tcell.KeyEnter:
			ta.selectMenuItem(ta.menuIndex)
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'k', 'K':
				ta.menuUp()
				return nil
			case 'j', 'J':
				ta.menuDown()
				return nil
			}
		}
	}

	// Thread list navigation
	if pageName == "threads" {
		switch event.Key() {
		case tcell.KeyUp:
			ta.threadUp()
			return nil
		case tcell.KeyDown:
			ta.threadDown()
			return nil
		case tcell.KeyEnter:
			ta.selectThread(ta.threadIndex)
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'k', 'K':
				ta.threadUp()
				return nil
			case 'j', 'J':
				ta.threadDown()
				return nil
			}
		}
	}

	switch event.Key() {
	case tcell.KeyEscape:
		switch pageName {
		case "threads":
			ta.showMenu()
			return nil
		case "comments":
			ta.stopAutoRefresh()
			ta.showThreads()
			return nil
		}
	case tcell.KeyRune:
		switch event.Rune() {
		case 'q', 'Q':
			ta.app.Stop()
			return nil
		case 'r', 'R':
			if pageName == "comments" {
				ta.refreshComments()
				return nil
			}
		case '/':
			if pageName == "comments" {
				ta.showFilter()
				return nil
			}
		}
	}

	return event
}

func (ta *TviewApp) showMenu() {
	ta.updateHeader("Reddit Stream Console", "Q:Quit  Enter:Select")
	ta.renderMenu()
	ta.pages.SwitchToPage("menu")
	ta.app.SetFocus(ta.menuView)
}

func (ta *TviewApp) showThreads() {
	title := "Threads"
	if ta.currentMenu != nil {
		title = ta.currentMenu.Title
	}
	ta.updateHeader(title, "Q:Quit  Enter:Open  Esc:Back")
	ta.renderThreadList()
	ta.pages.SwitchToPage("threads")
	ta.app.SetFocus(ta.threadView)
}

func (ta *TviewApp) showComments() {
	title := "Comments"
	if ta.currentThread != nil {
		title = ta.currentThread.Title
	}
	ta.updateHeader(title, "Q:Quit  R:Refresh  /:Filter  Esc:Back")
	ta.pages.SwitchToPage("comments")
	ta.app.SetFocus(ta.commentsView)
}

func (ta *TviewApp) showURLInput() {
	ta.updateHeader("Enter URL", "Enter:Submit  Esc:Back")
	ta.urlInput.SetText("")
	ta.urlInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			url := ta.urlInput.GetText()
			if url != "" {
				ta.loadThreadFromURL(url)
			}
		} else if key == tcell.KeyEscape {
			ta.showMenu()
		}
	})
	ta.pages.SwitchToPage("url")
	ta.app.SetFocus(ta.urlInput)
}

func (ta *TviewApp) showFilter() {
	ta.filterActive = true
	ta.filterInput.SetText(ta.commentFilter)
	ta.filterInput.SetDoneFunc(func(key tcell.Key) {
		ta.commentFilter = ta.filterInput.GetText()
		ta.hideFilter()
		ta.renderComments()
	})
	ta.filterInput.SetChangedFunc(func(text string) {
		ta.commentFilter = text
		ta.renderComments()
	})

	// Add filter to comments page
	commentsFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(ta.commentsView, 0, 1, false).
		AddItem(ta.filterInput, 1, 0, true)
	ta.pages.AddPage("comments", commentsFlex, true, true)
	ta.app.SetFocus(ta.filterInput)
}

func (ta *TviewApp) hideFilter() {
	ta.filterActive = false
	commentsFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(ta.commentsView, 0, 1, true)
	ta.pages.AddPage("comments", commentsFlex, true, true)
	ta.app.SetFocus(ta.commentsView)
}

func (ta *TviewApp) updateHeader(title, keys string) {
	ta.header.Clear()
	fmt.Fprintf(ta.header, " [::b]%s", title)

	ta.statusBar.Clear()
	fmt.Fprintf(ta.statusBar, " %s", formatKeys(keys))
}

func (ta *TviewApp) setStatus(msg string) {
	ta.statusBar.Clear()
	fmt.Fprintf(ta.statusBar, " %s", msg)
}

// formatKeys formats "Q:Quit  R:Refresh" into styled "[Q] Quit  [R] Refresh"
func formatKeys(keys string) string {
	parts := strings.Fields(keys)
	var formatted []string
	for _, part := range parts {
		if idx := strings.Index(part, ":"); idx != -1 {
			key := part[:idx]
			desc := part[idx+1:]
			// Orange key in brackets, cream description
			formatted = append(formatted, fmt.Sprintf("[#DEAA79][[#FFE6A9]%s[#DEAA79]][-] %s", key, desc))
		} else {
			formatted = append(formatted, part)
		}
	}
	return strings.Join(formatted, "  ")
}

func (ta *TviewApp) selectMenuItem(idx int) {
	if idx < 0 || idx >= len(ta.menuItems) {
		return
	}

	item := ta.menuItems[idx]
	if item.Type == "separator" {
		return
	}

	if item.Type == "url_input" {
		ta.showURLInput()
		return
	}

	ta.currentMenu = &item
	ta.setStatus("Loading threads...")
	ta.app.ForceDraw()

	go func() {
		threads, err := ta.fetchThreads(item)
		ta.app.QueueUpdateDraw(func() {
			if err != nil {
				ta.setStatus(fmt.Sprintf("Error: %v", err))
				return
			}
			if len(threads) == 0 {
				ta.setStatus("No threads found")
				return
			}
			ta.threadsData = threads
			ta.populateThreadList()
			ta.showThreads()
		})
	}()
}

func (ta *TviewApp) fetchThreads(item config.MenuItem) ([]reddit.Thread, error) {
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

	return ta.client.FindThreads(query)
}

func (ta *TviewApp) populateThreadList() {
	ta.threadIndex = 0
	ta.renderThreadList()
}

func (ta *TviewApp) selectThread(idx int) {
	if idx < 0 || idx >= len(ta.threadsData) {
		return
	}

	ta.currentThread = &ta.threadsData[idx]
	ta.comments = nil
	ta.commentFilter = ""
	ta.commentsView.Clear()
	ta.setStatus("Loading comments...")
	ta.app.ForceDraw()

	ta.loadComments()
	ta.showComments()
	ta.startAutoRefresh()
}

func (ta *TviewApp) loadThreadFromURL(url string) {
	ta.setStatus("Loading thread...")
	ta.app.ForceDraw()

	go func() {
		thread, err := ta.client.ThreadFromURL(url)
		ta.app.QueueUpdateDraw(func() {
			if err != nil {
				ta.setStatus(fmt.Sprintf("Error: %v", err))
				ta.showMenu()
				return
			}
			ta.currentThread = &thread
			ta.comments = nil
			ta.commentFilter = ""
			ta.commentsView.Clear()
			ta.loadComments()
			ta.showComments()
			ta.startAutoRefresh()
		})
	}()
}

func (ta *TviewApp) loadComments() {
	if ta.currentThread == nil {
		return
	}

	go func() {
		comments, title, err := ta.client.FetchComments(ta.currentThread.Permalink)
		ta.app.QueueUpdateDraw(func() {
			if err != nil {
				ta.setStatus(fmt.Sprintf("Error: %v", err))
				return
			}
			if title != "" {
				ta.currentThread.Title = title
				ta.updateHeader(title, "Q:Quit  R:Refresh  /:Filter  Esc:Back")
			}
			ta.comments = comments
			ta.renderComments()
			// Scroll to bottom
			ta.commentsView.ScrollToEnd()
		})
	}()
}

func (ta *TviewApp) refreshComments() {
	ta.setStatus("Refreshing...")
	ta.loadComments()
}

func (ta *TviewApp) startAutoRefresh() {
	ta.stopAutoRefresh()
	ta.refreshEnabled = true
	ta.stopRefresh = make(chan struct{})

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if ta.refreshEnabled {
					ta.app.QueueUpdateDraw(func() {
						ta.loadComments()
					})
				}
			case <-ta.stopRefresh:
				return
			}
		}
	}()
}

func (ta *TviewApp) stopAutoRefresh() {
	ta.refreshEnabled = false
	select {
	case ta.stopRefresh <- struct{}{}:
	default:
	}
}

func (ta *TviewApp) renderComments() {
	ta.commentsView.Clear()

	// Get terminal width for wrapping
	_, _, width, _ := ta.commentsView.GetInnerRect()
	if width <= 0 {
		width = 80 // fallback
	}

	filterLower := strings.ToLower(strings.TrimSpace(ta.commentFilter))
	roots := buildCommentTree(ta.comments, filterLower)

	var walk func(nodes []*commentNode, depth int)
	walk = func(nodes []*commentNode, depth int) {
		for _, node := range nodes {
			indent := strings.Repeat("  ", depth)
			arrow := ""
			if depth > 0 {
				arrow = "[#DEAA79]→[-] "
			}

			// Header: author • score • time
			header := fmt.Sprintf("%s%s[#FFE6A9::b]%s[-:-:-] [#666666]•[-] [#B1C29E]%d points[-] [#666666]•[-] [#659287]%s[-]",
				indent, arrow, node.comment.Author, node.comment.Score, node.comment.FormattedTime)
			fmt.Fprintln(ta.commentsView, header)

			// Body with proper wrapping
			bodyIndent := indent
			if depth > 0 {
				bodyIndent = indent + "  "
			}

			// Wrap body text to maintain indentation
			bodyWidth := width - len(bodyIndent) - 2
			if bodyWidth < 20 {
				bodyWidth = 20
			}

			for _, paragraph := range strings.Split(node.comment.Body, "\n") {
				if strings.TrimSpace(paragraph) == "" {
					fmt.Fprintln(ta.commentsView)
					continue
				}
				wrappedLines := wrapText(paragraph, bodyWidth)
				for _, line := range wrappedLines {
					fmt.Fprintf(ta.commentsView, "%s%s\n", bodyIndent, line)
				}
			}
			fmt.Fprintln(ta.commentsView)

			if len(node.children) > 0 {
				walk(node.children, depth+1)
			}
		}
	}

	walk(roots, 0)
}

// wrapText wraps text to the specified width
func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	var lines []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{}
	}

	currentLine := words[0]
	for _, word := range words[1:] {
		if len(currentLine)+1+len(word) <= width {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}
	lines = append(lines, currentLine)
	return lines
}

func (ta *TviewApp) Run() error {
	return ta.app.Run()
}

// commentNode represents a comment with its children for tree building
type commentNode struct {
	comment  reddit.Comment
	children []*commentNode
}

// buildCommentTree builds a tree structure from flat comments
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
