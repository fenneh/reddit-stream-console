package app

import (
	"strings"
	"testing"

	"github.com/fenneh/reddit-stream-console/internal/reddit"
	"github.com/fenneh/reddit-stream-console/internal/theme"
)

// — wrapText —

func TestWrapTextNonPositiveWidth(t *testing.T) {
	for _, w := range []int{0, -1} {
		got := wrapText("hello world", w)
		if len(got) != 1 || got[0] != "hello world" {
			t.Errorf("wrapText(width=%d) = %v, want [\"hello world\"]", w, got)
		}
	}
}

func TestWrapTextEmpty(t *testing.T) {
	got := wrapText("", 10)
	if len(got) != 0 {
		t.Errorf("wrapText(\"\") = %v, want []", got)
	}
}

func TestWrapTextFitsOnOneLine(t *testing.T) {
	got := wrapText("hello world", 11)
	if len(got) != 1 || got[0] != "hello world" {
		t.Errorf("wrapText fits = %v, want [\"hello world\"]", got)
	}
}

func TestWrapTextWrapsAtWordBoundary(t *testing.T) {
	// "hello" (5) + " world" (6) = 11 > 10, so wraps
	got := wrapText("hello world foo", 10)
	want := []string{"hello", "world foo"}
	if len(got) != len(want) {
		t.Fatalf("wrapText = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d: %q, want %q", i, got[i], want[i])
		}
	}
}

func TestWrapTextSingleLongWord(t *testing.T) {
	got := wrapText("superlongword", 5)
	if len(got) != 1 || got[0] != "superlongword" {
		t.Errorf("wrapText single long word = %v, want [\"superlongword\"]", got)
	}
}

// — buildCommentTree —

func TestBuildCommentTreeRoots(t *testing.T) {
	comments := []reddit.Comment{
		{ID: "c1", Author: "alice", Body: "first root"},
		{ID: "c2", Author: "bob", Body: "second root"},
	}
	roots := buildCommentTree(comments, "")
	if len(roots) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(roots))
	}
}

func TestBuildCommentTreeChildAttached(t *testing.T) {
	comments := []reddit.Comment{
		{ID: "c1", Author: "alice", Body: "root", ParentID: ""},
		{ID: "c2", Author: "bob", Body: "reply", ParentID: "c1"},
	}
	roots := buildCommentTree(comments, "")
	if len(roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(roots))
	}
	if len(roots[0].children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(roots[0].children))
	}
	if roots[0].children[0].comment.Author != "bob" {
		t.Errorf("child author = %q, want \"bob\"", roots[0].children[0].comment.Author)
	}
}

func TestBuildCommentTreeOrphanBecomesRoot(t *testing.T) {
	comments := []reddit.Comment{
		{ID: "c2", Author: "bob", Body: "reply", ParentID: "missing"},
	}
	roots := buildCommentTree(comments, "")
	if len(roots) != 1 {
		t.Errorf("expected orphan to become root, got %d roots", len(roots))
	}
}

func TestBuildCommentTreeFilterByBody(t *testing.T) {
	comments := []reddit.Comment{
		{ID: "c1", Author: "alice", Body: "hello world"},
		{ID: "c2", Author: "bob", Body: "goodbye world"},
	}
	roots := buildCommentTree(comments, "hello")
	if len(roots) != 1 {
		t.Fatalf("expected 1 root after filter, got %d", len(roots))
	}
	if roots[0].comment.Author != "alice" {
		t.Errorf("filtered root author = %q, want \"alice\"", roots[0].comment.Author)
	}
}

func TestBuildCommentTreeFilterByAuthor(t *testing.T) {
	comments := []reddit.Comment{
		{ID: "c1", Author: "alice", Body: "some text"},
		{ID: "c2", Author: "bob", Body: "other text"},
	}
	roots := buildCommentTree(comments, "bob")
	if len(roots) != 1 || roots[0].comment.Author != "bob" {
		t.Errorf("expected bob after author filter, got %v", roots)
	}
}

func TestBuildCommentTreeFilterExcludesAll(t *testing.T) {
	comments := []reddit.Comment{
		{ID: "c1", Author: "alice", Body: "hello"},
	}
	roots := buildCommentTree(comments, "zzz")
	if len(roots) != 0 {
		t.Errorf("expected empty result, got %d roots", len(roots))
	}
}

// — formatKeys —

func TestFormatKeysWrapsKeyAndDescription(t *testing.T) {
	ta := &TviewApp{theme: theme.Default()}
	got := ta.formatKeys("Q:Quit")
	if !strings.Contains(got, "Q") || !strings.Contains(got, "Quit") {
		t.Errorf("formatKeys(%q) = %q, want it to contain key and description", "Q:Quit", got)
	}
}

func TestFormatKeysJoinsMultipleEntries(t *testing.T) {
	ta := &TviewApp{theme: theme.Default()}
	got := ta.formatKeys("Q:Quit R:Refresh")
	if !strings.Contains(got, "Quit") || !strings.Contains(got, "Refresh") {
		t.Errorf("formatKeys(%q) = %q, want both entries present", "Q:Quit R:Refresh", got)
	}
	if idx1, idx2 := strings.Index(got, "Quit"), strings.Index(got, "Refresh"); idx1 >= idx2 {
		t.Errorf("formatKeys(%q) = %q, want Quit before Refresh", "Q:Quit R:Refresh", got)
	}
}

func TestFormatKeysPassesThroughEntryWithoutColon(t *testing.T) {
	ta := &TviewApp{theme: theme.Default()}
	got := ta.formatKeys("standalone")
	if got != "standalone" {
		t.Errorf("formatKeys(%q) = %q, want unchanged passthrough", "standalone", got)
	}
}

func TestFormatKeysEmptyInput(t *testing.T) {
	ta := &TviewApp{theme: theme.Default()}
	if got := ta.formatKeys(""); got != "" {
		t.Errorf("formatKeys(\"\") = %q, want \"\"", got)
	}
}

func TestBuildCommentTreeNestedChildren(t *testing.T) {
	comments := []reddit.Comment{
		{ID: "c1", Author: "a", Body: "root", ParentID: ""},
		{ID: "c2", Author: "b", Body: "child", ParentID: "c1"},
		{ID: "c3", Author: "c", Body: "grandchild", ParentID: "c2"},
	}
	roots := buildCommentTree(comments, "")
	if len(roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(roots))
	}
	child := roots[0].children
	if len(child) != 1 {
		t.Fatalf("expected 1 child, got %d", len(child))
	}
	grandchild := child[0].children
	if len(grandchild) != 1 || grandchild[0].comment.Author != "c" {
		t.Errorf("unexpected grandchild: %v", grandchild)
	}
}
