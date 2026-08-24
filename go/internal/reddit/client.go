package reddit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Client struct {
	httpClient   *http.Client
	userAgent    string
	baseURL      string // oauth.reddit.com API base
	tokenURL     string // token endpoint (www.reddit.com)
	clientID     string
	clientSecret string

	tokenMu     sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

// NewClient builds a Reddit API client authenticated with an app-only OAuth
// token. clientID is required; clientSecret is only needed for a "script"
// type app (a Reddit "installed app" has no secret, and clientSecret should
// be left empty in that case). Get credentials at https://www.reddit.com/prefs/apps.
func NewClient(userAgent, clientID, clientSecret string) (*Client, error) {
	if userAgent == "" {
		userAgent = "terminal:reddit-stream-console:v1.0.0 (by github.com/fenneh/reddit-stream-console)"
	}
	if clientID == "" {
		return nil, fmt.Errorf("REDDIT_CLIENT_ID is required - see .env.example")
	}

	return &Client{
		httpClient:   &http.Client{Timeout: 15 * time.Second},
		userAgent:    userAgent,
		baseURL:      "https://oauth.reddit.com",
		tokenURL:     "https://www.reddit.com/api/v1/access_token",
		clientID:     clientID,
		clientSecret: clientSecret,
	}, nil
}

// ensureToken returns a cached app-only OAuth access token, fetching (or
// refreshing an expired) one. Apps with a secret (the "script" type) use the
// standard client_credentials grant; secret-less apps (the "installed app"
// type) use Reddit's installed_client grant instead.
func (c *Client) ensureToken() (string, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.accessToken, nil
	}

	form := url.Values{}
	if c.clientSecret != "" {
		form.Set("grant_type", "client_credentials")
	} else {
		form.Set("grant_type", "https://oauth.reddit.com/grants/installed_client")
		form.Set("device_id", "reddit-stream-console")
	}

	req, err := http.NewRequest(http.MethodPost, c.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", c.userAgent)
	req.SetBasicAuth(c.clientID, c.clientSecret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch token: http %d", resp.StatusCode)
	}

	var payload struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode token: %w", err)
	}
	if payload.AccessToken == "" {
		return "", fmt.Errorf("empty access token in response")
	}

	c.accessToken = payload.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(payload.ExpiresIn-30) * time.Second)
	return c.accessToken, nil
}

func (c *Client) newAuthedRequest(urlStr string) (*http.Request, error) {
	token, err := c.ensureToken()
	if err != nil {
		return nil, fmt.Errorf("authenticate: %w", err)
	}

	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", c.userAgent)
	return req, nil
}

func (c *Client) FetchComments(permalink string) ([]Comment, string, error) {
	clean := strings.Trim(permalink, "/")
	urlStr := fmt.Sprintf("%s/%s?sort=new&limit=200&_=%d", c.baseURL, clean, time.Now().UnixNano())

	req, err := c.newAuthedRequest(urlStr)
	if err != nil {
		return nil, "", fmt.Errorf("build comments request: %w", err)
	}
	req.Header.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	req.Header.Set("Pragma", "no-cache")

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
		urlStr := fmt.Sprintf("%s/r/%s/search?%s", c.baseURL, cfg.Subreddit, query.Encode())

		req, err := c.newAuthedRequest(urlStr)
		if err != nil {
			return nil, fmt.Errorf("build search request: %w", err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetch threads: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("fetch threads: http %d", resp.StatusCode)
		}

		var listing listing
		err = json.NewDecoder(resp.Body).Decode(&listing)
		resp.Body.Close()
		if err != nil {
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
