package app

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/fenneh/reddit-stream-console/internal/reddit"
)

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
				ta.updateHeader(title, "Q:Quit  R:Refresh  /:Filter  H/V:Split  T:Theme  Esc:Back")
			}
			// Sort comments by time (oldest first, newest at bottom)
			sort.Slice(comments, func(i, j int) bool {
				return comments[i].CreatedUTC < comments[j].CreatedUTC
			})
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
	ta.renderCommentsToView(ta.commentsView, ta.comments, ta.commentFilter)
}

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
