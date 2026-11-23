package app

import (
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/fenneh/reddit-stream-console/internal/config"
	"github.com/fenneh/reddit-stream-console/internal/reddit"
	"github.com/fenneh/reddit-stream-console/internal/theme"
)

type CommentPane struct {
	id            string
	view          *tview.TextView
	filterInput   *tview.InputField
	thread        *reddit.Thread
	comments      []reddit.Comment
	commentFilter string
	filterActive  bool
	stopRefresh   chan struct{}

	theme theme.Theme

	showingMenu    bool
	showingThreads bool
	menuIndex      int
	threadIndex    int
	threadsData    []reddit.Thread
	currentMenu    *config.MenuItem
}

func NewCommentPane(id string, t theme.Theme) *CommentPane {
	pane := &CommentPane{
		id:          id,
		theme:       t,
		stopRefresh: make(chan struct{}),
	}

	pane.view = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true).
		SetWordWrap(true)
	pane.view.SetBackgroundColor(tcell.ColorDefault)
	pane.view.SetBorder(true)
	pane.view.SetBorderColor(t.Border.TCell)
	pane.view.SetBorderPadding(0, 0, 1, 1)

	pane.filterInput = tview.NewInputField().
		SetLabel("/ ").
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetFieldTextColor(t.Primary.TCell).
		SetLabelColor(t.Accent.TCell)

	return pane
}

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

func (p *CommentPane) SetActive(active bool) {
	if active {
		p.view.SetBorderColor(p.theme.Border.TCell)
	} else {
		p.view.SetBorderColor(p.theme.InactiveBorder.TCell)
	}
}

func (p *CommentPane) startAutoRefresh(fetch func()) {
	p.stopAutoRefresh()
	stop := make(chan struct{})
	p.stopRefresh = stop

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				fetch()
			case <-stop:
				return
			}
		}
	}()
}

func (p *CommentPane) stopAutoRefresh() {
	if p.stopRefresh != nil {
		close(p.stopRefresh)
		p.stopRefresh = nil
	}
}
