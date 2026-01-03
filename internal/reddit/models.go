package reddit

import (
	"encoding/json"
	"strings"
	"time"
)

type Thread struct {
	ID        string
	Title     string
	Permalink string
	Type      string
}

type Comment struct {
	ID            string
	Author        string
	Body          string
	CreatedUTC    float64
	FormattedTime string
	Score         int
	Depth         int
	ParentID      string
}

type ThreadQuery struct {
	Type                string
	Subreddit           string
	Flairs              []string
	MaxAgeHours         int
	Limit               int
	TitleMustContain    []string
	TitleMustNotContain []string
}

func (q ThreadQuery) WithinAge(createdUTC float64) bool {
	if q.MaxAgeHours == 0 {
		return true
	}
	ageSeconds := float64(q.MaxAgeHours * 3600)
	return createdUTC >= (nowUTC() - ageSeconds)
}

func (q ThreadQuery) TitleMatches(title string) bool {
	lower := stringsLower(title)
	for _, phrase := range q.TitleMustContain {
		if !stringsContains(lower, stringsLower(phrase)) {
			return false
		}
	}
	for _, phrase := range q.TitleMustNotContain {
		if stringsContains(lower, stringsLower(phrase)) {
			return false
		}
	}
	return true
}

func nowUTC() float64 {
	return float64(time.Now().Unix())
}

func stringsLower(value string) string {
	return strings.ToLower(value)
}

func stringsContains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}

type listing struct {
	Data listingData `json:"data"`
}

type listingData struct {
	Children []thing `json:"children"`
}

type thing struct {
	Kind string          `json:"kind"`
	Data json.RawMessage `json:"data"`
}

type postData struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Permalink  string  `json:"permalink"`
	CreatedUTC float64 `json:"created_utc"`
}

type redditComment struct {
	ID         string          `json:"id"`
	Author     string          `json:"author"`
	Body       string          `json:"body"`
	CreatedUTC float64         `json:"created_utc"`
	Score      int             `json:"score"`
	ParentID   string          `json:"parent_id"`
	Replies    json.RawMessage `json:"replies"`
}
