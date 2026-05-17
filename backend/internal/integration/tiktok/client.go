package tiktok

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"mmo/pkg/config"
)

type Client struct {
	clientKey    string
	clientSecret string
	redirectURL  string
	httpClient   *http.Client
	api          config.TikTokAPIConfig
}

type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"`
	OpenID       string `json:"open_id"`
}

type UserProfile struct {
	OpenID      string `json:"open_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
}

func New(cfg config.TikTokConfig) *Client {
	return &Client{
		clientKey:    cfg.ClientKey,
		clientSecret: cfg.ClientSecret,
		redirectURL:  cfg.RedirectURL,
		httpClient:   &http.Client{Timeout: cfg.HTTPTimeout},
		api:          cfg.API,
	}
}

func (c *Client) AuthURL(state string) string {
	params := url.Values{
		"client_key":    {c.clientKey},
		"scope":         {"user.info.basic,video.upload,video.publish"},
		"response_type": {"code"},
		"redirect_uri":  {c.redirectURL},
		"state":         {state},
	}
	return c.api.AuthBaseURL + "?" + params.Encode()
}

func (c *Client) ExchangeCode(ctx context.Context, code string) (*Tokens, error) {
	body := url.Values{
		"client_key":    {c.clientKey},
		"client_secret": {c.clientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {c.redirectURL},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.api.TokenURL,
		strings.NewReader(body.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var result struct {
		Data  Tokens `json:"data"`
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("decode tiktok token response: %w", err)
	}
	if result.Error.Code != 0 {
		return nil, fmt.Errorf("tiktok token error: %s", result.Error.Message)
	}
	return &result.Data, nil
}

func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*Tokens, error) {
	body := url.Values{
		"client_key":    {c.clientKey},
		"client_secret": {c.clientSecret},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.api.TokenURL,
		strings.NewReader(body.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data  Tokens `json:"data"`
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Error.Code != 0 {
		return nil, fmt.Errorf("tiktok refresh error: %s", result.Error.Message)
	}
	return &result.Data, nil
}

func (c *Client) GetUserInfo(ctx context.Context, accessToken string) (*UserProfile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.api.UserInfoURL+"?fields=open_id,display_name,avatar_url", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			User UserProfile `json:"user"`
		} `json:"data"`
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Error.Code != 0 {
		return nil, fmt.Errorf("tiktok user info error: %s", result.Error.Message)
	}
	return &result.Data.User, nil
}

type PostVideoResult struct {
	PublishID string `json:"publish_id"`
}

type PostVideoStatus struct {
	Status       string `json:"status"`
	PublicPostID string `json:"public_post_id"`
}

type PostVideoRequest struct {
	VideoURL     string
	Caption      string
	ProductLinks []string
}

func (c *Client) PostVideo(ctx context.Context, accessToken string, r PostVideoRequest) (string, error) {
	videoURL := r.VideoURL
	caption := r.Caption

	postInfo := map[string]any{
		"title":                    caption,
		"privacy_level":            "SELF_ONLY",
		"disable_duet":             false,
		"disable_comment":          false,
		"disable_stitch":           false,
		"video_cover_timestamp_ms": 1000,
	}

	if len(r.ProductLinks) > 0 {
		links := make([]map[string]string, len(r.ProductLinks))
		for i, id := range r.ProductLinks {
			links[i] = map[string]string{"id": id}
		}
		postInfo["brand_organic_type"] = "PRODUCT_LINK"
		postInfo["product_links"] = links
	}

	body := map[string]any{
		"post_info": postInfo,
		"source_info": map[string]any{
			"source":    "PULL_FROM_URL",
			"video_url": videoURL,
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("tiktok post video: marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.api.PublishInitURL,
		strings.NewReader(string(payload)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			PublishID string `json:"publish_id"`
		} `json:"data"`
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("tiktok post video: decode response: %w", err)
	}
	if result.Error.Code != "ok" {
		return "", fmt.Errorf("tiktok post video error: %s", result.Error.Message)
	}
	return result.Data.PublishID, nil
}

func (c *Client) GetPublishStatus(ctx context.Context, accessToken, publishID string) (string, error) {
	body := map[string]any{"publish_id": publishID}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("tiktok get publish status: marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.api.PublishStatusURL,
		strings.NewReader(string(payload)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("tiktok get publish status: decode response: %w", err)
	}
	if result.Error.Code != "ok" {
		return "", fmt.Errorf("tiktok publish status error: %s", result.Error.Message)
	}
	return result.Data.Status, nil
}

type VideoStats struct {
	ViewCount    int64 `json:"view_count"`
	LikeCount    int64 `json:"like_count"`
	CommentCount int64 `json:"comment_count"`
	ShareCount   int64 `json:"share_count"`
	PlayTime     int64 `json:"average_time_watched"`
}

func (c *Client) GetVideoStats(ctx context.Context, accessToken, videoID string) (*VideoStats, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.api.VideoQueryURL+"?fields=view_count,like_count,comment_count,share_count", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Videos []struct {
				ID         string     `json:"id"`
				VideoStats VideoStats `json:"statistics"`
			} `json:"videos"`
		} `json:"data"`
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Error.Code != "" && result.Error.Code != "ok" {
		return nil, fmt.Errorf("tiktok stats error: %s", result.Error.Message)
	}
	for _, v := range result.Data.Videos {
		if v.ID == videoID {
			return &v.VideoStats, nil
		}
	}
	return &VideoStats{}, nil
}
