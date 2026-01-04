package reddit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	httpClient *http.Client
	userAgent  string
}

func NewClient(userAgent string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		userAgent:  userAgent,
	}
}

func (c *Client) FetchComments(permalink string) ([]Comment, string, error) {
	clean := strings.Trim(permalink, "/")
	urlStr := fmt.Sprintf("https://www.reddit.com/%s.json", clean)

	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, "", fmt.Errorf("build comments request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("fetch comments: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("fetch comments: http %d", resp.StatusCode)
	}

	var payload []listing
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, "", fmt.Errorf("decode comments: %w", err)
	}
	if len(payload) < 2 {
		return nil, "", fmt.Errorf("comments payload missing")
	}

	postID, postTitle := extractPost(payload[0])
	if postID == "" {
		return nil, "", fmt.Errorf("missing post id")
	}

	comments := make([]Comment, 0, 256)
	for _, thing := range payload[1].Data.Children {
		if thing.Kind != "t1" {
			continue
		}
		c.processComment(thing.Data, postID, 0, &comments)
	}

	return comments, postTitle, nil
}

func (c *Client) FindThreads(cfg ThreadQuery) ([]Thread, error) {
	threads := make([]Thread, 0, 64)

	for _, flair := range cfg.Flairs {
		query := url.Values{}
		query.Set("q", fmt.Sprintf("flair:\"%s\"", flair))
		query.Set("sort", "new")
		query.Set("t", "week")
		query.Set("limit", fmt.Sprintf("%d", cfg.Limit))
		query.Set("restrict_sr", "1")
		urlStr := fmt.Sprintf("https://www.reddit.com/r/%s/search.json?%s", cfg.Subreddit, query.Encode())

		req, err := http.NewRequest(http.MethodGet, urlStr, nil)
		if err != nil {
			return nil, fmt.Errorf("build search request: %w", err)
		}
		req.Header.Set("User-Agent", c.userAgent)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetch threads: %w", err)
		}
		if resp.Body != nil {
			defer resp.Body.Close()
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("fetch threads: http %d", resp.StatusCode)
		}

		var listing listing
		if err := json.NewDecoder(resp.Body).Decode(&listing); err != nil {
			return nil, fmt.Errorf("decode threads: %w", err)
		}

		for _, thing := range listing.Data.Children {
			if thing.Kind != "t3" {
				continue
			}
			var post postData
			if err := json.Unmarshal(thing.Data, &post); err != nil {
				continue
			}
			if !cfg.WithinAge(post.CreatedUTC) {
				continue
			}
			if !cfg.TitleMatches(post.Title) {
				continue
			}

			threads = append(threads, Thread{
				ID:        post.ID,
				Title:     post.Title,
				Permalink: post.Permalink,
				Type:      cfg.Type,
			})
		}

		if len(threads) > 0 {
			break
		}
	}

	return threads, nil
}

func (c *Client) ThreadFromURL(input string) (Thread, error) {
	permalink, err := normalizePermalink(input)
	if err != nil {
		return Thread{}, err
	}

	comments, title, err := c.FetchComments(permalink)
	if err != nil {
		return Thread{}, err
	}
	_ = comments

	threadID := extractThreadID(permalink)
	if threadID == "" {
		return Thread{}, fmt.Errorf("invalid thread id")
	}

	return Thread{
		ID:        threadID,
		Title:     title,
		Permalink: permalink,
		Type:      "url_input",
	}, nil
}

func normalizePermalink(input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", fmt.Errorf("empty url")
	}

	if strings.HasPrefix(trimmed, "http") {
		parsed, err := url.Parse(trimmed)
		if err != nil {
			return "", fmt.Errorf("parse url: %w", err)
		}
		trimmed = parsed.Path
	}

	trimmed = strings.TrimSuffix(trimmed, ".json")
	trimmed = strings.TrimSuffix(trimmed, "/")
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	return trimmed, nil
}

func extractThreadID(permalink string) string {
	parts := strings.Split(strings.Trim(permalink, "/"), "/")
	if len(parts) >= 4 && parts[0] == "r" && parts[2] == "comments" {
		return parts[3]
	}
	return ""
}

func extractPost(listing listing) (string, string) {
	if len(listing.Data.Children) == 0 {
		return "", ""
	}
	thing := listing.Data.Children[0]
	if thing.Kind != "t3" {
		return "", ""
	}
	var post postData
	if err := json.Unmarshal(thing.Data, &post); err != nil {
		return "", ""
	}
	return post.ID, post.Title
}

func (c *Client) processComment(raw json.RawMessage, postID string, depth int, out *[]Comment) {
	var comment redditComment
	if err := json.Unmarshal(raw, &comment); err != nil {
		return
	}
	if comment.Body == "[deleted]" || comment.Body == "[removed]" {
		return
	}

	parentFullname := "t3_" + postID
	if depth == 0 && comment.ParentID != parentFullname {
		return
	}

	parentID := strings.TrimPrefix(comment.ParentID, "t1_")
	if strings.HasPrefix(comment.ParentID, "t3_") {
		parentID = ""
	}
	*out = append(*out, Comment{
		ID:            comment.ID,
		Author:        fallback(comment.Author, "[deleted]"),
		Body:          comment.Body,
		CreatedUTC:    comment.CreatedUTC,
		FormattedTime: formatTimestamp(comment.CreatedUTC),
		Score:         comment.Score,
		Depth:         depth,
		ParentID:      parentID,
	})

	if len(comment.Replies) == 0 || string(comment.Replies) == "\"\"" {
		return
	}

	var replyListing listing
	if err := json.Unmarshal(comment.Replies, &replyListing); err != nil {
		return
	}
	for _, child := range replyListing.Data.Children {
		if child.Kind != "t1" {
			continue
		}
		c.processComment(child.Data, postID, depth+1, out)
	}
}

func formatTimestamp(ts float64) string {
	if ts == 0 {
		return ""
	}
	return time.Unix(int64(ts), 0).Local().Format("2006-01-02 15:04:05")
}

func fallback(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
