package app

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/fenneh/reddit-stream-console/internal/config"
	"github.com/fenneh/reddit-stream-console/internal/reddit"
)

// CommentPane represents a single pane that can display comments or menu
type CommentPane struct {
	id            string
	view          *tview.TextView
	filterInput   *tview.InputField
	thread        *reddit.Thread
	comments      []reddit.Comment
	commentFilter string
	filterActive  bool
	refreshEnabled bool
	stopRefresh   chan struct{}

	// State tracking for what's displayed in this pane
	showingMenu    bool
	showingThreads bool
	menuIndex      int
	threadIndex    int
	threadsData    []reddit.Thread
	currentMenu    *config.MenuItem
}

// NewCommentPane creates a new comment pane with the given ID
func NewCommentPane(id string) *CommentPane {
	pane := &CommentPane{
		id:          id,
		stopRefresh: make(chan struct{}),
	}

	pane.view = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true).
		SetWordWrap(true)
	pane.view.SetBackgroundColor(tcell.ColorDefault)
	pane.view.SetBorder(true)
	pane.view.SetBorderColor(tealTview)
	pane.view.SetBorderPadding(0, 0, 1, 1)

	pane.filterInput = tview.NewInputField().
		SetLabel("/ ").
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetFieldTextColor(warmCreamTview).
		SetLabelColor(warmOrangeTview)

	return pane
}

// Clear resets the pane state
func (p *CommentPane) Clear() {
	p.thread = nil
	p.comments = nil
	p.commentFilter = ""
	p.filterActive = false
	p.showingMenu = false
	p.showingThreads = false
	p.menuIndex = 0
	p.threadIndex = 0
	p.threadsData = nil
	p.currentMenu = nil
	p.view.Clear()
}

// SetActive updates the pane's border to indicate active/inactive state
func (p *CommentPane) SetActive(active bool) {
	if active {
		p.view.SetBorderColor(tealTview)
	} else {
		p.view.SetBorderColor(tcell.NewRGBColor(80, 80, 80))
	}
}
