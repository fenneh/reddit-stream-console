package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/fenneh/reddit-stream-console/internal/config"
	"github.com/fenneh/reddit-stream-console/internal/reddit"
	"github.com/fenneh/reddit-stream-console/internal/theme"
)

// Version is set at build time via ldflags
var Version = "dev"

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

	// Inherit the terminal's real background everywhere by default,
	// instead of tview's hardcoded ColorBlack/ColorBlue/ColorGreen.
	tview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault
	tview.Styles.ContrastBackgroundColor = tcell.ColorDefault
	tview.Styles.MoreContrastBackgroundColor = tcell.ColorDefault
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

	// Wrapping flexes whose borders need re-theming on theme change
	menuFlex     *tview.Flex
	threadFlex   *tview.Flex
	urlInnerFlex *tview.Flex

	client        *reddit.Client
	menuItems     []config.MenuItem
	threadsData   []reddit.Thread
	comments      []reddit.Comment
	currentThread *reddit.Thread
	currentMenu   *config.MenuItem

	theme         theme.Theme
	startupNotice string // shown briefly in the status bar at launch

	filterActive   bool
	commentFilter  string
	refreshEnabled bool
	stopRefresh    chan struct{}

	latestVersion string // Latest version from GitHub, empty if current or unknown

	// Split pane support
	primaryPane    *CommentPane
	secondaryPane  *CommentPane
	activePaneID   string // "primary" or "secondary"
	splitMode      bool
	splitDirection int // tview.FlexRow (horizontal) or FlexColumn (vertical)
}

func NewTviewApp(menuItems []config.MenuItem, client *reddit.Client, t theme.Theme) *TviewApp {
	ta := &TviewApp{
		app:         tview.NewApplication(),
		pages:       tview.NewPages(),
		menuItems:   menuItems,
		client:      client,
		theme:       t,
		stopRefresh: make(chan struct{}),
	}

	ta.setupUI()
	return ta
}

// SetStartupNotice queues a message to be shown in the status bar on first
// render, e.g. a warning about an unknown theme name in the config.
func (ta *TviewApp) SetStartupNotice(msg string) {
	ta.startupNotice = msg
}

func (ta *TviewApp) setupUI() {
	// Header
	ta.header = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	ta.header.SetBackgroundColor(ta.theme.HeaderBg.TCell)
	ta.header.SetTextColor(ta.theme.HeaderFg.TCell)

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
	ta.commentsView.SetBorderColor(ta.theme.Border.TCell)
	ta.commentsView.SetBorderPadding(0, 0, 1, 1)

	// URL input
	ta.urlInput = tview.NewInputField().
		SetLabel("URL: ").
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetFieldTextColor(ta.theme.Primary.TCell).
		SetLabelColor(ta.theme.Primary.TCell)

	// Filter input
	ta.filterInput = tview.NewInputField().
		SetLabel("/ ").
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetFieldTextColor(ta.theme.Primary.TCell).
		SetLabelColor(ta.theme.Accent.TCell)

	// Status bar
	ta.statusBar = tview.NewTextView().
		SetDynamicColors(true)
	ta.statusBar.SetBackgroundColor(ta.theme.HeaderBg.TCell)
	ta.statusBar.SetTextColor(ta.theme.HeaderFg.TCell)

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
	menuFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(ta.menuView, 0, 2, true).
		AddItem(nil, 0, 1, false)
	menuFlex.SetBackgroundColor(tcell.ColorDefault)
	menuFlex.SetBorder(true)
	menuFlex.SetBorderColor(ta.theme.Border.TCell)
	ta.menuFlex = menuFlex
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
			lines = append(lines, fmt.Sprintf("[%s::b]→ %s[-:-:-]", ta.theme.Accent.Hex, item.Title))
			if item.Description != "" {
				lines = append(lines, fmt.Sprintf("[%s]  %s[-]", ta.theme.Muted.Hex, item.Description))
			}
		} else {
			lines = append(lines, fmt.Sprintf("[%s]  %s[-]", ta.theme.Secondary.Hex, item.Title))
			if item.Description != "" {
				lines = append(lines, fmt.Sprintf("[%s]  %s[-]", ta.theme.Subtle.Hex, item.Description))
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
	threadFlex.SetBorderColor(ta.theme.Border.TCell)
	ta.threadFlex = threadFlex
	ta.pages.AddPage("threads", threadFlex, true, false)
}

func (ta *TviewApp) renderThreadList() {
	ta.threadView.Clear()

	if len(ta.threadsData) == 0 {
		fmt.Fprintf(ta.threadView, "[%s]No threads found[-]", ta.theme.Muted.Hex)
		return
	}

	var lines []string
	for i, thread := range ta.threadsData {
		if i == ta.threadIndex {
			lines = append(lines, fmt.Sprintf("[%s::b]→ %s[-:-:-]", ta.theme.Accent.Hex, thread.Title))
		} else {
			lines = append(lines, fmt.Sprintf("[%s]  %s[-]", ta.theme.Secondary.Hex, thread.Title))
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
	fmt.Fprintf(label, "[%s::b]Enter Reddit Thread URL[-:-:-]", ta.theme.Primary.Hex)

	// Style the input field
	ta.urlInput.SetBackgroundColor(tcell.ColorDefault)
	ta.urlInput.SetFieldBackgroundColor(ta.theme.InputBg.TCell)
	ta.urlInput.SetFieldTextColor(ta.theme.Primary.TCell)
	ta.urlInput.SetLabelColor(ta.theme.Accent.TCell)
	ta.urlInput.SetLabel("→ ")
	ta.urlInput.SetPlaceholder("https://reddit.com/r/...")
	ta.urlInput.SetPlaceholderTextColor(ta.theme.Placeholder.TCell)

	// Hint text
	hint := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	hint.SetBackgroundColor(tcell.ColorDefault)
	fmt.Fprintf(hint, "[%s]Press [%s]Enter[-] to submit  •  [%s]Esc[-] to go back[-]", ta.theme.Muted.Hex, ta.theme.Accent.Hex, ta.theme.Accent.Hex)

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
	innerFlex.SetBorderColor(ta.theme.Border.TCell)
	ta.urlInnerFlex = innerFlex

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

	// Menu page navigation (non-split mode)
	if pageName == "menu" && !ta.splitMode {
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

	// Split mode pane navigation
	if pageName == "comments" && ta.splitMode {
		pane := ta.getActivePane()
		if pane != nil {
			if pane.showingMenu {
				switch event.Key() {
				case tcell.KeyUp:
					ta.paneMenuUp(pane)
					return nil
				case tcell.KeyDown:
					ta.paneMenuDown(pane)
					return nil
				case tcell.KeyEnter:
					ta.paneSelectMenuItem(pane)
					return nil
				case tcell.KeyEscape:
					// Close this pane and exit split mode
					ta.closeSplitMode()
					return nil
				case tcell.KeyRune:
					switch event.Rune() {
					case 'k', 'K':
						ta.paneMenuUp(pane)
						return nil
					case 'j', 'J':
						ta.paneMenuDown(pane)
						return nil
					}
				}
			} else if pane.showingThreads {
				switch event.Key() {
				case tcell.KeyUp:
					ta.paneThreadUp(pane)
					return nil
				case tcell.KeyDown:
					ta.paneThreadDown(pane)
					return nil
				case tcell.KeyEnter:
					ta.paneSelectThread(pane)
					return nil
				case tcell.KeyEscape:
					// Go back to menu in this pane
					pane.showingThreads = false
					pane.showingMenu = true
					ta.rebuildSplitLayout()
					return nil
				case tcell.KeyRune:
					switch event.Rune() {
					case 'k', 'K':
						ta.paneThreadUp(pane)
						return nil
					case 'j', 'J':
						ta.paneThreadDown(pane)
						return nil
					}
				}
			} else {
				// Showing comments in this pane
				switch event.Key() {
				case tcell.KeyEscape:
					// Go back to threads in this pane
					pane.showingThreads = true
					pane.thread = nil
					pane.comments = nil
					// Stop refresh for this pane
					if pane.refreshEnabled {
						pane.refreshEnabled = false
						select {
						case pane.stopRefresh <- struct{}{}:
						default:
						}
					}
					ta.rebuildSplitLayout()
					return nil
				}
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
		case 'h', 'H':
			if pageName == "comments" && !ta.splitMode {
				ta.splitView(tview.FlexRow) // Horizontal split (top/bottom)
				return nil
			}
		case 'v', 'V':
			if pageName == "comments" && !ta.splitMode {
				ta.splitView(tview.FlexColumn) // Vertical split (side by side)
				return nil
			}
		case 't', 'T':
			ta.cycleTheme()
			return nil
		}
	case tcell.KeyTab:
		if pageName == "comments" && ta.splitMode {
			ta.switchActivePane()
			return nil
		}
	}

	return event
}

func (ta *TviewApp) showMenu() {
	ta.updateHeaderWithUpdate("Reddit Stream Console", "Q:Quit  Enter:Select  T:Theme")
	ta.renderMenu()
	ta.pages.SwitchToPage("menu")
	ta.app.SetFocus(ta.menuView)
}

func (ta *TviewApp) updateHeaderWithUpdate(title, keys string) {
	ta.header.Clear()
	fmt.Fprintf(ta.header, " [::b]%s", title)

	ta.statusBar.Clear()
	leftPart := ta.formatKeys(keys)

	if ta.latestVersion != "" {
		_, _, width, _ := ta.statusBar.GetInnerRect()
		updateMsg := fmt.Sprintf("Update available: %s", ta.latestVersion)
		leftLen := len(strings.ReplaceAll(keys, ":", " ")) + 10 // rough estimate
		padding := width - leftLen - len(updateMsg) - 4
		if padding < 2 {
			padding = 2
		}
		fmt.Fprintf(ta.statusBar, " %s%s[%s]%s[-]", leftPart, strings.Repeat(" ", padding), ta.theme.Secondary.Hex, updateMsg)
	} else {
		fmt.Fprintf(ta.statusBar, " %s", leftPart)
	}
}

func (ta *TviewApp) showThreads() {
	title := "Threads"
	if ta.currentMenu != nil {
		title = ta.currentMenu.Title
	}
	ta.updateHeader(title, "Q:Quit  Enter:Open  T:Theme  Esc:Back")
	ta.renderThreadList()
	ta.pages.SwitchToPage("threads")
	ta.app.SetFocus(ta.threadView)
}

func (ta *TviewApp) showComments() {
	title := "Comments"
	if ta.currentThread != nil {
		title = ta.currentThread.Title
	}
	ta.updateHeader(title, "Q:Quit  R:Refresh  /:Filter  H/V:Split  T:Theme  Esc:Back")
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
	fmt.Fprintf(ta.statusBar, " %s", ta.formatKeys(keys))
}

func (ta *TviewApp) setStatus(msg string) {
	ta.statusBar.Clear()
	fmt.Fprintf(ta.statusBar, " %s", msg)
}

// formatKeys formats "Q:Quit  R:Refresh" into styled "[Q] Quit  [R] Refresh"
func (ta *TviewApp) formatKeys(keys string) string {
	parts := strings.Fields(keys)
	var formatted []string
	for _, part := range parts {
		if idx := strings.Index(part, ":"); idx != -1 {
			key := part[:idx]
			desc := part[idx+1:]
			formatted = append(formatted, fmt.Sprintf("[%s][[%s]%s[%s]][-] %s", ta.theme.Accent.Hex, ta.theme.Primary.Hex, key, ta.theme.Accent.Hex, desc))
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

func (ta *TviewApp) Run() error {
	// Set terminal title
	fmt.Print("\033]0;reddit-stream-console\007")

	if ta.startupNotice != "" {
		ta.setStatus(ta.startupNotice)
	}

	// Check for updates in background
	go ta.checkForUpdates()

	return ta.app.Run()
}

// applyTheme re-applies static colours from t to every primitive that
// holds them as state, then re-renders dynamic views so their inline
// markup picks up the new palette.
func (ta *TviewApp) applyTheme(t theme.Theme) {
	ta.theme = t
	if ta.primaryPane != nil {
		ta.primaryPane.theme = t
	}
	if ta.secondaryPane != nil {
		ta.secondaryPane.theme = t
	}

	ta.header.SetBackgroundColor(t.HeaderBg.TCell)
	ta.header.SetTextColor(t.HeaderFg.TCell)
	ta.statusBar.SetBackgroundColor(t.HeaderBg.TCell)
	ta.statusBar.SetTextColor(t.HeaderFg.TCell)

	ta.commentsView.SetBorderColor(t.Border.TCell)
	if ta.menuFlex != nil {
		ta.menuFlex.SetBorderColor(t.Border.TCell)
	}
	if ta.threadFlex != nil {
		ta.threadFlex.SetBorderColor(t.Border.TCell)
	}
	if ta.urlInnerFlex != nil {
		ta.urlInnerFlex.SetBorderColor(t.Border.TCell)
	}

	ta.urlInput.SetFieldBackgroundColor(t.InputBg.TCell)
	ta.urlInput.SetFieldTextColor(t.Primary.TCell)
	ta.urlInput.SetLabelColor(t.Accent.TCell)
	ta.urlInput.SetPlaceholderTextColor(t.Placeholder.TCell)
	ta.filterInput.SetFieldTextColor(t.Primary.TCell)
	ta.filterInput.SetLabelColor(t.Accent.TCell)

	ta.renderMenu()
	ta.renderThreadList()
	if ta.currentThread != nil {
		ta.renderComments()
	}
	if ta.splitMode {
		ta.rebuildSplitLayout()
	}
}

// cycleTheme advances to the next built-in theme, applies it live, and
// persists the choice to app_config.json so it survives restarts.
func (ta *TviewApp) cycleTheme() {
	names := theme.Names()
	if len(names) == 0 {
		return
	}
	idx := 0
	for i, n := range names {
		if n == ta.theme.Name {
			idx = (i + 1) % len(names)
			break
		}
	}
	next := names[idx]
	ta.applyTheme(theme.Get(next))

	if path, err := config.SaveTheme(next); err != nil {
		ta.setStatus(fmt.Sprintf("Theme: %s (save failed: %v)", next, err))
	} else {
		ta.setStatus(fmt.Sprintf("Theme: %s — saved to %s", next, path))
	}
}

// versionGreater returns true when a is numerically newer than b.
// Strings like "1.10.0" and "1.9.0" compare correctly; unknown segments are treated as 0.
func versionGreater(a, b string) bool {
	aParts := strings.SplitN(a, ".", 3)
	bParts := strings.SplitN(b, ".", 3)
	for i := 0; i < 3; i++ {
		var an, bn int
		if i < len(aParts) {
			an, _ = strconv.Atoi(aParts[i])
		}
		if i < len(bParts) {
			bn, _ = strconv.Atoi(bParts[i])
		}
		if an != bn {
			return an > bn
		}
	}
	return false
}

func (ta *TviewApp) checkForUpdates() {
	if Version == "dev" {
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/fenneh/reddit-stream-console/releases/latest")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(Version, "v")

	if versionGreater(latest, current) {
		ta.latestVersion = release.TagName
		ta.app.QueueUpdateDraw(func() {
			// Refresh menu footer if on menu page
			pageName, _ := ta.pages.GetFrontPage()
			if pageName == "menu" {
				ta.showMenu()
			}
		})
	}
}
