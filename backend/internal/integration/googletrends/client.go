package googletrends

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"mmo/pkg/config"
)

type Client struct {
	apiBase    string
	httpClient *http.Client
}

type Trend struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Keywords    []string `json:"keywords"`
	Score       float64  `json:"score"`
	SourceURL   string   `json:"source_url"`
}

func New(cfg config.GoogleTrendsConfig) *Client {
	return &Client{
		apiBase:    cfg.APIBase,
		httpClient: &http.Client{Timeout: cfg.HTTPTimeout},
	}
}

func (c *Client) FetchDailyTrends(ctx context.Context, geo string) ([]Trend, error) {
	url := fmt.Sprintf("%s?hl=en-US&tz=-60&geo=%s&ns=15", c.apiBase, geo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; AutoContent/1.0)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google trends request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google trends request: unexpected status %d", resp.StatusCode)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	body := strings.TrimPrefix(string(raw), ")]}'")
	body = strings.TrimSpace(body)

	if strings.HasPrefix(body, "<") {
		return nil, fmt.Errorf("google trends returned HTML (likely blocked or CAPTCHA)")
	}

	var data struct {
		Default struct {
			TrendingSearchesDays []struct {
				TrendingSearches []struct {
					Title struct {
						Query       string `json:"query"`
						ExploreLink string `json:"exploreLink"`
					} `json:"title"`
					FormattedTraffic string `json:"formattedTraffic"`
					Articles         []struct {
						Title  string `json:"title"`
						URL    string `json:"url"`
						Source string `json:"source"`
					} `json:"articles"`
					RelatedQueries []struct {
						Query string `json:"query"`
					} `json:"relatedQueries"`
				} `json:"trendingSearches"`
			} `json:"trendingSearchesDays"`
		} `json:"default"`
	}

	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return nil, fmt.Errorf("parse google trends: %w", err)
	}

	var trends []Trend
	if len(data.Default.TrendingSearchesDays) == 0 {
		return trends, nil
	}
	for _, ts := range data.Default.TrendingSearchesDays[0].TrendingSearches {
		keywords := []string{ts.Title.Query}
		for _, rq := range ts.RelatedQueries {
			keywords = append(keywords, rq.Query)
		}

		desc := ""
		sourceURL := ""
		if len(ts.Articles) > 0 {
			desc = ts.Articles[0].Title
			sourceURL = ts.Articles[0].URL
		}

		trends = append(trends, Trend{
			Title:       ts.Title.Query,
			Description: desc,
			Keywords:    keywords,
			SourceURL:   sourceURL,
		})
	}
	return trends, nil
}
