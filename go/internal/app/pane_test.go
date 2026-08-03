package app

import (
	"testing"

	"github.com/fenneh/reddit-stream-console/internal/config"
	"github.com/fenneh/reddit-stream-console/internal/reddit"
	"github.com/fenneh/reddit-stream-console/internal/theme"
)

// — NewCommentPane —

func TestNewCommentPaneSetsID(t *testing.T) {
	pane := NewCommentPane("left", theme.Default())
	if pane.id != "left" {
		t.Errorf("id = %q, want \"left\"", pane.id)
	}
}

func TestNewCommentPaneBuildsViews(t *testing.T) {
	pane := NewCommentPane("right", theme.Default())
	if pane.view == nil {
		t.Error("view is nil")
	}
	if pane.filterInput == nil {
		t.Error("filterInput is nil")
	}
	if pane.stopRefresh == nil {
		t.Error("stopRefresh channel is nil")
	}
}

// — Clear —

func TestClearResetsState(t *testing.T) {
	pane := NewCommentPane("left", theme.Default())
	pane.thread = &reddit.Thread{ID: "t1"}
	pane.comments = []reddit.Comment{{ID: "c1"}}
	pane.commentFilter = "hello"
	pane.filterActive = true
	pane.showingMenu = true
	pane.showingThreads = true
	pane.menuIndex = 3
	pane.threadIndex = 2
	pane.threadsData = []reddit.Thread{{ID: "t2"}}
	pane.currentMenu = &config.MenuItem{}

	pane.Clear()

	if pane.thread != nil {
		t.Error("thread not cleared")
	}
	if pane.comments != nil {
		t.Error("comments not cleared")
	}
	if pane.commentFilter != "" {
		t.Error("commentFilter not cleared")
	}
	if pane.filterActive {
		t.Error("filterActive not cleared")
	}
	if pane.showingMenu {
		t.Error("showingMenu not cleared")
	}
	if pane.showingThreads {
		t.Error("showingThreads not cleared")
	}
	if pane.menuIndex != 0 {
		t.Error("menuIndex not reset")
	}
	if pane.threadIndex != 0 {
		t.Error("threadIndex not reset")
	}
	if pane.threadsData != nil {
		t.Error("threadsData not cleared")
	}
	if pane.currentMenu != nil {
		t.Error("currentMenu not cleared")
	}
}

// — SetActive —

func TestSetActiveTrueUsesBorderColor(t *testing.T) {
	th := theme.Default()
	pane := NewCommentPane("left", th)
	pane.SetActive(true)
	if pane.view.GetBorderColor() != th.Border.TCell {
		t.Error("border color not set to active border color")
	}
}

func TestSetActiveFalseUsesInactiveBorderColor(t *testing.T) {
	th := theme.Default()
	pane := NewCommentPane("left", th)
	pane.SetActive(false)
	if pane.view.GetBorderColor() != th.InactiveBorder.TCell {
		t.Error("border color not set to inactive border color")
	}
}
