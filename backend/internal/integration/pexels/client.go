package pexels

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

type VideoFile struct {
	ID       int    `json:"id"`
	Quality  string `json:"quality"`
	FileType string `json:"file_type"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Link     string `json:"link"`
}

type Video struct {
	ID         int         `json:"id"`
	Width      int         `json:"width"`
	Height     int         `json:"height"`
	Duration   int         `json:"duration"`
	URL        string      `json:"url"`
	VideoFiles []VideoFile `json:"video_files"`
}

type Photo struct {
	ID     int    `json:"id"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	URL    string `json:"url"`
	Src    struct {
		Original string `json:"original"`
		Large    string `json:"large"`
		Medium   string `json:"medium"`
	} `json:"src"`
}

func New(cfg config.PexelsConfig) *Client {
	return &Client{
		apiKey:     cfg.APIKey,
		apiBase:    cfg.APIBase,
		httpClient: &http.Client{Timeout: cfg.HTTPTimeout},
	}
}

func (c *Client) SearchVideos(ctx context.Context, query string, perPage int) ([]Video, error) {
	if c.apiKey == "" {
		return nil, nil
	}
	params := url.Values{
		"query":    {query},
		"per_page": {fmt.Sprintf("%d", perPage)},
		"size":     {"medium"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.apiBase+"/videos/search?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pexels video search: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Videos []Video `json:"videos"`
		Error  string  `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Error != "" {
		return nil, fmt.Errorf("pexels error: %s", result.Error)
	}
	return result.Videos, nil
}

func (c *Client) SearchPhotos(ctx context.Context, query string, perPage int) ([]Photo, error) {
	if c.apiKey == "" {
		return nil, nil
	}
	params := url.Values{
		"query":    {query},
		"per_page": {fmt.Sprintf("%d", perPage)},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.apiBase+"/v1/search?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pexels photo search: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Photos []Photo `json:"photos"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Photos, nil
}

func BestVideoURL(v Video) string {
	for _, f := range v.VideoFiles {
		if f.Quality == "hd" && f.Width <= 1920 {
			return f.Link
		}
	}
	for _, f := range v.VideoFiles {
		if f.Quality == "sd" {
			return f.Link
		}
	}
	if len(v.VideoFiles) > 0 {
		return v.VideoFiles[0].Link
	}
	return ""
}
