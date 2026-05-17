package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"mmo/pkg/config"
)

type Client struct {
	apiKey     string
	apiBase    string
	httpClient *http.Client
}

type Video struct {
	VideoID     string   `json:"video_id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	ChannelName string   `json:"channel_name"`
	ViewCount   string   `json:"view_count"`
	Keywords    []string `json:"keywords"`
	SourceURL   string   `json:"source_url"`
}

func New(cfg config.YouTubeConfig) *Client {
	return &Client{
		apiKey:     cfg.APIKey,
		apiBase:    cfg.APIBase,
		httpClient: &http.Client{Timeout: cfg.HTTPTimeout},
	}
}

func (c *Client) FetchTrending(ctx context.Context, regionCode, categoryID string) ([]Video, error) {
	if c.apiKey == "" {
		return nil, nil
	}

	params := url.Values{
		"part":       {"snippet,statistics"},
		"chart":      {"mostPopular"},
		"regionCode": {regionCode},
		"maxResults": {"20"},
		"key":        {c.apiKey},
	}
	if categoryID != "" && categoryID != "0" {
		params.Set("videoCategoryId", categoryID)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.apiBase+"/videos?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("youtube trending request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Items []struct {
			ID      string `json:"id"`
			Snippet struct {
				Title        string   `json:"title"`
				Description  string   `json:"description"`
				ChannelTitle string   `json:"channelTitle"`
				Tags         []string `json:"tags"`
			} `json:"snippet"`
			Statistics struct {
				ViewCount string `json:"viewCount"`
			} `json:"statistics"`
		} `json:"items"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, fmt.Errorf("youtube API error: %s", result.Error.Message)
	}

	videos := make([]Video, 0, len(result.Items))
	for _, item := range result.Items {
		keywords := append([]string{item.Snippet.Title}, item.Snippet.Tags...)
		videos = append(videos, Video{
			VideoID:     item.ID,
			Title:       item.Snippet.Title,
			Description: item.Snippet.Description,
			ChannelName: item.Snippet.ChannelTitle,
			ViewCount:   item.Statistics.ViewCount,
			Keywords:    keywords,
			SourceURL:   "https://www.youtube.com/watch?v=" + item.ID,
		})
	}
	return videos, nil
}
