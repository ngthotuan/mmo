package reddit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"mmo/pkg/config"
)

type Client struct {
	apiBase    string
	httpClient *http.Client
}

type Post struct {
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	URL       string   `json:"url"`
	Score     int      `json:"score"`
	Subreddit string   `json:"subreddit"`
	Keywords  []string `json:"keywords"`
}

func New(cfg config.RedditConfig) *Client {
	return &Client{
		apiBase:    cfg.APIBase,
		httpClient: &http.Client{Timeout: cfg.HTTPTimeout},
	}
}

func (c *Client) FetchTopPosts(ctx context.Context, subreddit, timeWindow string, limit int) ([]Post, error) {
	if limit <= 0 || limit > 25 {
		limit = 10
	}
	url := fmt.Sprintf("%s/r/%s/top.json?t=%s&limit=%d", c.apiBase, subreddit, timeWindow, limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "AutoContent/1.0 (content aggregation bot)")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("reddit request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("reddit rate limited")
	}

	var result struct {
		Data struct {
			Children []struct {
				Data struct {
					Title     string `json:"title"`
					Selftext  string `json:"selftext"`
					URL       string `json:"url"`
					Score     int    `json:"score"`
					Subreddit string `json:"subreddit"`
				} `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	posts := make([]Post, 0, len(result.Data.Children))
	for _, child := range result.Data.Children {
		d := child.Data
		keywords := extractKeywords(d.Title)
		posts = append(posts, Post{
			Title:     d.Title,
			Body:      truncate(d.Selftext, 500),
			URL:       "https://reddit.com" + d.URL,
			Score:     d.Score,
			Subreddit: d.Subreddit,
			Keywords:  keywords,
		})
	}
	return posts, nil
}

func extractKeywords(title string) []string {
	words := strings.Fields(title)
	seen := map[string]bool{}
	var kw []string
	for _, w := range words {
		w = strings.ToLower(strings.Trim(w, ".,!?\"'()[]"))
		if len(w) > 4 && !seen[w] {
			seen[w] = true
			kw = append(kw, w)
		}
	}
	if len(kw) > 5 {
		return kw[:5]
	}
	return kw
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
