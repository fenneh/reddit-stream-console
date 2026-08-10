package app

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/fenneh/reddit-stream-console/internal/reddit"
)

// splitView creates a split view with the current thread in primary pane
// and menu in the secondary pane
func (ta *TviewApp) splitView(direction int) {
	if ta.splitMode {
		return // Already in split mode
	}

	// Stop global auto-refresh first
	ta.stopAutoRefresh()

	ta.splitMode = true
	ta.splitDirection = direction

	// Create primary pane from current state
	ta.primaryPane = NewCommentPane("primary", ta.theme)
	ta.primaryPane.thread = ta.currentThread
	ta.primaryPane.comments = ta.comments
	ta.primaryPane.commentFilter = ta.commentFilter

	// Create secondary pane for menu
	ta.secondaryPane = NewCommentPane("secondary", ta.theme)
	ta.secondaryPane.showingMenu = true

	// Start auto-refresh for primary pane
	ta.startAutoRefreshForPane(ta.primaryPane)

	// Set secondary as active (where menu will appear)
	ta.activePaneID = "secondary"
	ta.primaryPane.SetActive(false)
	ta.secondaryPane.SetActive(true)

	// Rebuild the layout
	ta.rebuildSplitLayout()
}

func (ta *TviewApp) rebuildSplitLayout() {
	splitFlex := tview.NewFlex().SetDirection(ta.splitDirection)

	// Build primary pane content
	primaryContent := ta.buildPaneContent(ta.primaryPane)
	secondaryContent := ta.buildPaneContent(ta.secondaryPane)

	splitFlex.AddItem(primaryContent, 0, 1, ta.activePaneID == "primary")
	splitFlex.AddItem(secondaryContent, 0, 1, ta.activePaneID == "secondary")

	ta.pages.AddPage("comments", splitFlex, true, true)
	ta.updateSplitHeader()
}

func (ta *TviewApp) buildPaneContent(pane *CommentPane) *tview.Flex {
	flex := tview.NewFlex().SetDirection(tview.FlexRow)

	if pane.showingMenu {
		// Show menu in this pane
		menuView := tview.NewTextView().
			SetDynamicColors(true).
			SetTextAlign(tview.AlignCenter)
		menuView.SetBackgroundColor(tcell.ColorDefault)
		menuView.SetBorder(true)
		if pane.id == ta.activePaneID {
			menuView.SetBorderColor(ta.theme.Border.TCell)
		} else {
			menuView.SetBorderColor(ta.theme.InactiveBorder.TCell)
		}

		var lines []string
		lines = append(lines, "")
		for i, item := range ta.menuItems {
			if item.Type == "separator" {
				lines = append(lines, "")
				continue
			}
			if i == pane.menuIndex {
				lines = append(lines, fmt.Sprintf("[%s::b]→ %s[-:-:-]", ta.theme.Accent.Hex, item.Title))
			} else {
				lines = append(lines, fmt.Sprintf("[%s]  %s[-]", ta.theme.Secondary.Hex, item.Title))
			}
		}
		fmt.Fprint(menuView, strings.Join(lines, "\n"))
		flex.AddItem(menuView, 0, 1, true)
	} else if pane.showingThreads {
		threadView := tview.NewTextView().
			SetDynamicColors(true).
			SetScrollable(true).
			SetTextAlign(tview.AlignCenter)
		threadView.SetBackgroundColor(tcell.ColorDefault)
		threadView.SetBorder(true)
		if pane.id == ta.activePaneID {
			threadView.SetBorderColor(ta.theme.Border.TCell)
		} else {
			threadView.SetBorderColor(ta.theme.InactiveBorder.TCell)
		}

		var lines []string
		for i, thread := range pane.threadsData {
			if i == pane.threadIndex {
				lines = append(lines, fmt.Sprintf("[%s::b]→ %s[-:-:-]", ta.theme.Accent.Hex, thread.Title))
			} else {
				lines = append(lines, fmt.Sprintf("[%s]  %s[-]", ta.theme.Secondary.Hex, thread.Title))
			}
		}
		fmt.Fprint(threadView, strings.Join(lines, "\n"))
		flex.AddItem(threadView, 0, 1, true)
	} else {
		// Show comments
		pane.view.Clear()
		ta.renderCommentsToView(pane.view, pane.comments, pane.commentFilter)
		pane.view.ScrollToEnd()
		flex.AddItem(pane.view, 0, 1, true)
	}

	return flex
}

func (ta *TviewApp) renderCommentsToView(view *tview.TextView, comments []reddit.Comment, filter string) {
	_, _, width, _ := view.GetInnerRect()
	if width <= 0 {
		// Estimate width based on terminal size when view not yet drawn
		_, _, termWidth, _ := ta.mainFlex.GetInnerRect()
		if ta.splitMode && ta.splitDirection == tview.FlexColumn {
			width = (termWidth / 2) - 4 // Side by side, account for borders
		} else if ta.splitMode {
			width = termWidth - 4 // Stacked
		} else {
			width = termWidth - 4
		}
		if width <= 0 {
			width = 80
		}
	}

	filterLower := strings.ToLower(strings.TrimSpace(filter))
	roots := buildCommentTree(comments, filterLower)

	var walk func(nodes []*commentNode, depth int)
	walk = func(nodes []*commentNode, depth int) {
		for _, node := range nodes {
			indent := strings.Repeat("  ", depth)
			arrow := ""
			if depth > 0 {
				arrow = fmt.Sprintf("[%s]→[-] ", ta.theme.Accent.Hex)
			}

			header := fmt.Sprintf("%s%s[%s::b]%s[-:-:-] [%s]•[-] [%s]%d points[-] [%s]•[-] [%s]%s[-]",
				indent, arrow,
				ta.theme.Primary.Hex, node.comment.Author,
				ta.theme.Subtle.Hex,
				ta.theme.Secondary.Hex, node.comment.Score,
				ta.theme.Subtle.Hex,
				ta.theme.Border.Hex, node.comment.FormattedTime)
			fmt.Fprintln(view, header)

			bodyIndent := indent
			if depth > 0 {
				bodyIndent = indent + "  "
			}

			bodyWidth := width - len(bodyIndent) - 2
			if bodyWidth < 20 {
				bodyWidth = 20
			}

			for _, paragraph := range strings.Split(node.comment.Body, "\n") {
				if strings.TrimSpace(paragraph) == "" {
					fmt.Fprintln(view)
					continue
				}
				wrappedLines := wrapText(paragraph, bodyWidth)
				for _, line := range wrappedLines {
					fmt.Fprintf(view, "%s%s\n", bodyIndent, line)
				}
			}
			fmt.Fprintln(view)

			if len(node.children) > 0 {
				walk(node.children, depth+1)
			}
		}
	}

	walk(roots, 0)
}

func (ta *TviewApp) switchActivePane() {
	if !ta.splitMode || ta.secondaryPane == nil {
		return
	}

	if ta.activePaneID == "primary" {
		ta.activePaneID = "secondary"
		ta.primaryPane.SetActive(false)
		ta.secondaryPane.SetActive(true)
	} else {
		ta.activePaneID = "primary"
		ta.primaryPane.SetActive(true)
		ta.secondaryPane.SetActive(false)
	}

	ta.rebuildSplitLayout()
	ta.updateSplitHeader()
}

func (ta *TviewApp) updateSplitHeader() {
	var title string
	if ta.activePaneID == "primary" && ta.primaryPane.thread != nil {
		title = fmt.Sprintf("[1] %s", ta.primaryPane.thread.Title)
	} else if ta.activePaneID == "secondary" {
		if ta.secondaryPane.showingMenu {
			title = "[2] Select Thread"
		} else if ta.secondaryPane.thread != nil {
			title = fmt.Sprintf("[2] %s", ta.secondaryPane.thread.Title)
		}
	}

	ta.header.Clear()
	fmt.Fprintf(ta.header, " [::b]%s", title)

	ta.statusBar.Clear()
	keys := "Q:Quit  R:Refresh  /:Filter  Tab:Switch  Esc:Close"
	fmt.Fprintf(ta.statusBar, " %s", ta.formatKeys(keys))
}

func (ta *TviewApp) getActivePane() *CommentPane {
	if ta.activePaneID == "secondary" && ta.secondaryPane != nil {
		return ta.secondaryPane
	}
	return ta.primaryPane
}

func (ta *TviewApp) closeSplitMode() {
	if !ta.splitMode {
		return
	}

	// Stop refresh on both panes if running
	if ta.primaryPane != nil && ta.primaryPane.refreshEnabled {
		ta.primaryPane.refreshEnabled = false
		select {
		case ta.primaryPane.stopRefresh <- struct{}{}:
		default:
		}
	}
	if ta.secondaryPane != nil && ta.secondaryPane.refreshEnabled {
		ta.secondaryPane.refreshEnabled = false
		select {
		case ta.secondaryPane.stopRefresh <- struct{}{}:
		default:
		}
	}

	// Keep primary pane state as current state
	if ta.primaryPane != nil && ta.primaryPane.thread != nil {
		ta.currentThread = ta.primaryPane.thread
		ta.comments = ta.primaryPane.comments
		ta.commentFilter = ta.primaryPane.commentFilter
	}

	ta.splitMode = false
	ta.primaryPane = nil
	ta.secondaryPane = nil
	ta.activePaneID = ""

	// Rebuild single pane comments page (replace the split layout)
	ta.buildCommentsPage()

	// Re-render comments to the original view
	ta.renderComments()
	ta.commentsView.ScrollToEnd()

	// Restart auto-refresh for single mode
	ta.startAutoRefresh()

	ta.showComments()
}

func (ta *TviewApp) paneMenuUp(pane *CommentPane) {
	orig := pane.menuIndex
	for {
		pane.menuIndex--
		if pane.menuIndex < 0 {
			pane.menuIndex = len(ta.menuItems) - 1
		}
		if pane.menuIndex == orig {
			break
		}
		if ta.menuItems[pane.menuIndex].Type != "separator" {
			break
		}
	}
	ta.rebuildSplitLayout()
}

func (ta *TviewApp) paneMenuDown(pane *CommentPane) {
	orig := pane.menuIndex
	for {
		pane.menuIndex++
		if pane.menuIndex >= len(ta.menuItems) {
			pane.menuIndex = 0
		}
		if pane.menuIndex == orig {
			break
		}
		if ta.menuItems[pane.menuIndex].Type != "separator" {
			break
		}
	}
	ta.rebuildSplitLayout()
}

func (ta *TviewApp) paneSelectMenuItem(pane *CommentPane) {
	if pane.menuIndex < 0 || pane.menuIndex >= len(ta.menuItems) {
		return
	}

	item := ta.menuItems[pane.menuIndex]
	if item.Type == "separator" {
		return
	}

	if item.Type == "url_input" {
		// URL input not supported in split mode for now
		return
	}

	pane.currentMenu = &item
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
			pane.threadsData = threads
			pane.threadIndex = 0
			pane.showingMenu = false
			pane.showingThreads = true
			ta.rebuildSplitLayout()
		})
	}()
}

func (ta *TviewApp) paneThreadUp(pane *CommentPane) {
	if len(pane.threadsData) == 0 {
		return
	}
	pane.threadIndex--
	if pane.threadIndex < 0 {
		pane.threadIndex = len(pane.threadsData) - 1
	}
	ta.rebuildSplitLayout()
}

func (ta *TviewApp) paneThreadDown(pane *CommentPane) {
	if len(pane.threadsData) == 0 {
		return
	}
	pane.threadIndex++
	if pane.threadIndex >= len(pane.threadsData) {
		pane.threadIndex = 0
	}
	ta.rebuildSplitLayout()
}

func (ta *TviewApp) paneSelectThread(pane *CommentPane) {
	if pane.threadIndex < 0 || pane.threadIndex >= len(pane.threadsData) {
		return
	}

	thread := pane.threadsData[pane.threadIndex]
	pane.thread = &thread
	pane.comments = nil
	pane.commentFilter = ""
	pane.showingThreads = false
	pane.showingMenu = false

	ta.setStatus("Loading comments...")
	ta.app.ForceDraw()

	go func() {
		comments, title, err := ta.client.FetchComments(thread.Permalink)
		ta.app.QueueUpdateDraw(func() {
			if err != nil {
				ta.setStatus(fmt.Sprintf("Error: %v", err))
				return
			}
			if title != "" {
				pane.thread.Title = title
			}
			// Sort comments by time
			sort.Slice(comments, func(i, j int) bool {
				return comments[i].CreatedUTC < comments[j].CreatedUTC
			})
			pane.comments = comments
			ta.rebuildSplitLayout()
			ta.startAutoRefreshForPane(pane)
		})
	}()
}

func (ta *TviewApp) startAutoRefreshForPane(pane *CommentPane) {
	if pane == nil || pane.thread == nil {
		return
	}

	// Stop existing refresh
	if pane.refreshEnabled {
		pane.refreshEnabled = false
		select {
		case pane.stopRefresh <- struct{}{}:
		default:
		}
	}

	pane.refreshEnabled = true
	pane.stopRefresh = make(chan struct{})

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if pane.refreshEnabled && pane.thread != nil {
					ta.loadCommentsForPane(pane)
				}
			case <-pane.stopRefresh:
				return
			}
		}
	}()
}

func (ta *TviewApp) loadCommentsForPane(pane *CommentPane) {
	if pane == nil || pane.thread == nil {
		return
	}

	go func() {
		comments, title, err := ta.client.FetchComments(pane.thread.Permalink)
		ta.app.QueueUpdateDraw(func() {
			if err != nil {
				return
			}
			if title != "" {
				pane.thread.Title = title
			}
			sort.Slice(comments, func(i, j int) bool {
				return comments[i].CreatedUTC < comments[j].CreatedUTC
			})
			pane.comments = comments
			if ta.splitMode {
				ta.rebuildSplitLayout()
			}
		})
	}()
}
