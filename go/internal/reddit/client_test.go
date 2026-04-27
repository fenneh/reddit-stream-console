package reddit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// redirectTransport rewrites all outbound requests to the given test server,
// so we can exercise HTTP logic without hitting reddit.com.
type redirectTransport struct {
	srv *httptest.Server
}

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := *req
	clonedURL := *req.URL
	clonedURL.Scheme = "http"
	clonedURL.Host = t.srv.Listener.Addr().String()
	cloned.URL = &clonedURL
	return http.DefaultTransport.RoundTrip(&cloned)
}

func newTestClient(srv *httptest.Server) *Client {
	return &Client{
		httpClient: &http.Client{Transport: &redirectTransport{srv: srv}},
		userAgent:  "test",
	}
}

// — normalizePermalink —

func TestNormalizePermalink(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"/r/soccer/comments/abc123/match_thread/", "/r/soccer/comments/abc123/match_thread"},
		{"r/soccer/comments/abc123/match_thread", "/r/soccer/comments/abc123/match_thread"},
		{"https://www.reddit.com/r/soccer/comments/abc123/match_thread/", "/r/soccer/comments/abc123/match_thread"},
		{"https://www.reddit.com/r/soccer/comments/abc123/match_thread.json", "/r/soccer/comments/abc123/match_thread"},
		{"/r/FantasyPL/comments/xyz789/", "/r/FantasyPL/comments/xyz789"},
	}
	for _, tc := range cases {
		got, err := normalizePermalink(tc.input)
		if err != nil {
			t.Errorf("normalizePermalink(%q) unexpected error: %v", tc.input, err)
			continue
		}
		if got != tc.want {
			t.Errorf("normalizePermalink(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestNormalizePermalinkEmpty(t *testing.T) {
	for _, input := range []string{"", "   "} {
		if _, err := normalizePermalink(input); err == nil {
			t.Errorf("expected error for input %q", input)
		}
	}
}

// — extractThreadID —

func TestExtractThreadID(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"/r/soccer/comments/abc123/match_thread", "abc123"},
		{"/r/FantasyPL/comments/xyz789/", "xyz789"},
		{"/r/soccer/comments/abc123", "abc123"},
		{"not/a/valid/path", ""},
		{"", ""},
	}
	for _, tc := range cases {
		got := extractThreadID(tc.input)
		if got != tc.want {
			t.Errorf("extractThreadID(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// — ThreadQuery methods —

func TestWithinAge(t *testing.T) {
	now := float64(time.Now().Unix())
	q := ThreadQuery{MaxAgeHours: 2}

	if !q.WithinAge(now - 3600) {
		t.Error("post from 1h ago should be within 2h limit")
	}
	if q.WithinAge(now - 10800) {
		t.Error("post from 3h ago should be outside 2h limit")
	}
}

func TestWithinAgeZeroMeansUnlimited(t *testing.T) {
	q := ThreadQuery{MaxAgeHours: 0}
	if !q.WithinAge(0) {
		t.Error("MaxAgeHours=0 should always return true")
	}
}

func TestTitleMatches(t *testing.T) {
	q := ThreadQuery{
		TitleMustContain:    []string{"match thread"},
		TitleMustNotContain: []string{"post match"},
	}
	cases := []struct {
		title string
		want  bool
	}{
		{"Match Thread: Arsenal vs Chelsea", true},
		{"Post Match Thread: Arsenal vs Chelsea", false},
		{"Arsenal vs Chelsea", false},
		{"MATCH THREAD: Liverpool vs City", true},
	}
	for _, tc := range cases {
		got := q.TitleMatches(tc.title)
		if got != tc.want {
			t.Errorf("TitleMatches(%q) = %v, want %v", tc.title, got, tc.want)
		}
	}
}

// — small helpers —

func TestFallback(t *testing.T) {
	if fallback("", "default") != "default" {
		t.Error("expected fallback for empty string")
	}
	if fallback("value", "default") != "value" {
		t.Error("expected original value when non-empty")
	}
}

func TestFormatTimestamp(t *testing.T) {
	if formatTimestamp(0) != "" {
		t.Error("expected empty string for zero timestamp")
	}
	if got := formatTimestamp(1700000000); len(got) == 0 {
		t.Error("expected non-empty formatted timestamp")
	}
}

// — extractPost —

func TestExtractPost(t *testing.T) {
	postJSON, _ := json.Marshal(postData{ID: "abc123", Title: "Match Thread"})
	l := listing{Data: listingData{Children: []thing{{Kind: "t3", Data: postJSON}}}}

	id, title := extractPost(l)
	if id != "abc123" {
		t.Errorf("extractPost id = %q, want %q", id, "abc123")
	}
	if title != "Match Thread" {
		t.Errorf("extractPost title = %q, want %q", title, "Match Thread")
	}
}

func TestExtractPostEmptyListing(t *testing.T) {
	id, title := extractPost(listing{})
	if id != "" || title != "" {
		t.Error("expected empty id and title for empty listing")
	}
}

func TestExtractPostWrongKind(t *testing.T) {
	l := listing{Data: listingData{Children: []thing{{Kind: "t1", Data: json.RawMessage(`{}`)}}}}
	id, _ := extractPost(l)
	if id != "" {
		t.Error("expected empty id for non-t3 kind")
	}
}

// — processComment —

func TestProcessComment(t *testing.T) {
	c := NewClient("test")
	raw, _ := json.Marshal(redditComment{
		ID:       "c1",
		Author:   "alice",
		Body:     "hello",
		Score:    3,
		ParentID: "t3_post1",
		Replies:  json.RawMessage(`""`),
	})

	var out []Comment
	c.processComment(raw, "post1", 0, &out)

	if len(out) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(out))
	}
	got := out[0]
	if got.Author != "alice" || got.Body != "hello" || got.Score != 3 {
		t.Errorf("unexpected comment fields: %+v", got)
	}
	if got.ParentID != "" {
		t.Errorf("top-level comment ParentID should be empty, got %q", got.ParentID)
	}
}

func TestProcessCommentDeletedSkipped(t *testing.T) {
	c := NewClient("test")
	for _, body := range []string{"[deleted]", "[removed]"} {
		raw, _ := json.Marshal(redditComment{ID: "c1", Author: "x", Body: body, ParentID: "t3_post1"})
		var out []Comment
		c.processComment(raw, "post1", 0, &out)
		if len(out) != 0 {
			t.Errorf("expected %q comment to be skipped", body)
		}
	}
}

func TestProcessCommentWrongParentSkipped(t *testing.T) {
	c := NewClient("test")
	raw, _ := json.Marshal(redditComment{ID: "c1", Author: "x", Body: "hi", ParentID: "t3_other"})
	var out []Comment
	c.processComment(raw, "post1", 0, &out)
	if len(out) != 0 {
		t.Error("expected comment with mismatched parent to be skipped at depth 0")
	}
}

func TestProcessCommentWithReplies(t *testing.T) {
	c := NewClient("test")

	replyJSON, _ := json.Marshal(redditComment{
		ID:       "c2",
		Author:   "bob",
		Body:     "reply",
		ParentID: "t1_c1",
		Replies:  json.RawMessage(`""`),
	})
	replyListing, _ := json.Marshal(listing{
		Data: listingData{Children: []thing{{Kind: "t1", Data: replyJSON}}},
	})

	raw, _ := json.Marshal(redditComment{
		ID:       "c1",
		Author:   "alice",
		Body:     "hello",
		ParentID: "t3_post1",
		Replies:  replyListing,
	})

	var out []Comment
	c.processComment(raw, "post1", 0, &out)

	if len(out) != 2 {
		t.Fatalf("expected 2 comments (parent + reply), got %d", len(out))
	}
	if out[0].Depth != 0 || out[1].Depth != 1 {
		t.Errorf("unexpected depths: %d, %d", out[0].Depth, out[1].Depth)
	}
	if out[1].ParentID != "c1" {
		t.Errorf("reply ParentID = %q, want %q", out[1].ParentID, "c1")
	}
}

// — FetchComments (HTTP) —

func buildCommentsPayload(postID, title, commentBody string) []byte {
	postJSON, _ := json.Marshal(postData{
		ID:        postID,
		Title:     title,
		Permalink: "/r/test/comments/" + postID + "/thread/",
	})
	commentJSON, _ := json.Marshal(redditComment{
		ID:       "c1",
		Author:   "user1",
		Body:     commentBody,
		Score:    1,
		ParentID: "t3_" + postID,
		Replies:  json.RawMessage(`""`),
	})
	payload := []listing{
		{Data: listingData{Children: []thing{{Kind: "t3", Data: postJSON}}}},
		{Data: listingData{Children: []thing{{Kind: "t1", Data: commentJSON}}}},
	}
	b, _ := json.Marshal(payload)
	return b
}

func TestFetchComments(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(buildCommentsPayload("abc123", "Match Thread", "Great goal!"))
	}))
	defer srv.Close()

	comments, title, err := newTestClient(srv).FetchComments("/r/test/comments/abc123/thread/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if title != "Match Thread" {
		t.Errorf("title = %q, want %q", title, "Match Thread")
	}
	if len(comments) != 1 || comments[0].Body != "Great goal!" {
		t.Errorf("unexpected comments: %+v", comments)
	}
}

func TestFetchCommentsHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	_, _, err := newTestClient(srv).FetchComments("/r/test/comments/abc123/thread/")
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
}

// — FindThreads (HTTP) —

func buildSearchPayload(postID, title string) []byte {
	postJSON, _ := json.Marshal(postData{
		ID:         postID,
		Title:      title,
		Permalink:  "/r/soccer/comments/" + postID + "/",
		CreatedUTC: float64(time.Now().Unix()),
	})
	l := listing{Data: listingData{Children: []thing{{Kind: "t3", Data: postJSON}}}}
	b, _ := json.Marshal(l)
	return b
}

func TestFindThreads(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(buildSearchPayload("abc123", "Match Thread: Test vs Test"))
	}))
	defer srv.Close()

	threads, err := newTestClient(srv).FindThreads(ThreadQuery{
		Type:      "match",
		Subreddit: "soccer",
		Flairs:    []string{"match thread"},
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(threads) != 1 || threads[0].ID != "abc123" {
		t.Errorf("unexpected threads: %+v", threads)
	}
}

func TestFindThreadsTitleFilter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(buildSearchPayload("abc123", "Post Match Thread: Test vs Test"))
	}))
	defer srv.Close()

	threads, err := newTestClient(srv).FindThreads(ThreadQuery{
		Type:                "match",
		Subreddit:           "soccer",
		Flairs:              []string{"match thread"},
		Limit:               10,
		TitleMustNotContain: []string{"post match"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(threads) != 0 {
		t.Errorf("expected filtered thread to be excluded, got %+v", threads)
	}
}
