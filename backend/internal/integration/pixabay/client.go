package pixabay

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"mmo/pkg/config"
	"mmo/pkg/httpclient"
)

type Client struct {
	apiKey     string
	apiBase    string
	httpClient *http.Client
}

type VideoHit struct {
	ID      int    `json:"id"`
	PageURL string `json:"pageURL"`
	Duration int   `json:"duration"`
	Videos  struct {
		Large  VideoStream `json:"large"`
		Medium VideoStream `json:"medium"`
		Small  VideoStream `json:"small"`
	} `json:"videos"`
	Tags string `json:"tags"`
}

type VideoStream struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Size   int    `json:"size"`
}

type ImageHit struct {
	ID            int    `json:"id"`
	WebformatURL  string `json:"webformatURL"`
	LargeImageURL string `json:"largeImageURL"`
	ImageWidth    int    `json:"imageWidth"`
	ImageHeight   int    `json:"imageHeight"`
	Tags          string `json:"tags"`
}

func New(cfg config.PixabayConfig) *Client {
	return &Client{
		apiKey:     cfg.APIKey,
		apiBase:    cfg.APIBase,
		httpClient: httpclient.New("pixabay", cfg.HTTPTimeout),
	}
}

func (c *Client) SearchVideos(ctx context.Context, query string, perPage int) ([]VideoHit, error) {
	if c.apiKey == "" {
		return nil, nil
	}
	params := url.Values{
		"key":        {c.apiKey},
		"q":          {query},
		"per_page":   {fmt.Sprintf("%d", perPage)},
		"video_type": {"film"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.apiBase+"/videos/?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pixabay video search: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Hits []VideoHit `json:"hits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Hits, nil
}

func (c *Client) SearchImages(ctx context.Context, query string, perPage int) ([]ImageHit, error) {
	if c.apiKey == "" {
		return nil, nil
	}
	params := url.Values{
		"key":         {c.apiKey},
		"q":           {query},
		"per_page":    {fmt.Sprintf("%d", perPage)},
		"image_type":  {"photo"},
		"orientation": {"vertical"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.apiBase+"/?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pixabay image search: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Hits []ImageHit `json:"hits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Hits, nil
}
