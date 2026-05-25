package vnexpress

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	httpClient *http.Client
}

type Trend struct {
	Title       string
	Description string
	Keywords    []string
	SourceURL   string
	Category    string
}

// RSS feed categories mapped to niches
var rssFeeds = map[string]string{
	"tin-moi-nhat": "https://vnexpress.net/rss/tin-moi-nhat.rss",
	"kinh-doanh":   "https://vnexpress.net/rss/kinh-doanh.rss",
	"cong-nghe":    "https://vnexpress.net/rss/cong-nghe.rss",
	"the-thao":     "https://vnexpress.net/rss/the-thao.rss",
	"giai-tri":     "https://vnexpress.net/rss/giai-tri.rss",
}

type rssItem struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Link        string `xml:"link"`
	PubDate     string `xml:"pubDate"`
}

type rssFeed struct {
	Items []rssItem `xml:"channel>item"`
}

func New(timeout time.Duration) *Client {
	if timeout == 0 {
		timeout = 15 * time.Second
	}
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
	}
}

// FetchTrending fetches top articles from VnExpress RSS feeds.
// perFeed controls how many items to take from each category feed.
func (c *Client) FetchTrending(ctx context.Context, perFeed int) ([]Trend, error) {
	if perFeed <= 0 {
		perFeed = 5
	}
	var all []Trend
	for category, url := range rssFeeds {
		items, err := c.fetchFeed(ctx, url)
		if err != nil {
			continue
		}
		limit := perFeed
		if len(items) < limit {
			limit = len(items)
		}
		for _, item := range items[:limit] {
			title := cleanText(item.Title)
			desc := cleanText(item.Description)
			if title == "" {
				continue
			}
			all = append(all, Trend{
				Title:       title,
				Description: desc,
				Keywords:    extractKeywords(title, desc),
				SourceURL:   item.Link,
				Category:    category,
			})
		}
	}
	return all, nil
}

func (c *Client) fetchFeed(ctx context.Context, url string) ([]rssItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; AutoContent/1.0)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vnexpress RSS returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var feed rssFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, err
	}
	return feed.Items, nil
}

// cleanText strips HTML tags and extra whitespace from RSS content.
func cleanText(s string) string {
	// Remove CDATA wrappers
	s = strings.TrimPrefix(s, "<![CDATA[")
	s = strings.TrimSuffix(s, "]]>")
	// Strip HTML tags
	for strings.Contains(s, "<") {
		start := strings.Index(s, "<")
		end := strings.Index(s, ">")
		if end < start {
			break
		}
		s = s[:start] + " " + s[end+1:]
	}
	return strings.Join(strings.Fields(s), " ")
}

// extractKeywords pulls meaningful words from title and description for the keywords field.
func extractKeywords(title, desc string) []string {
	combined := title + " " + desc
	words := strings.Fields(combined)
	seen := map[string]bool{}
	var kw []string
	for _, w := range words {
		w = strings.Trim(w, ".,!?;:\"'()[]")
		if len([]rune(w)) > 3 && !seen[w] {
			seen[w] = true
			kw = append(kw, w)
		}
		if len(kw) >= 10 {
			break
		}
	}
	return kw
}
