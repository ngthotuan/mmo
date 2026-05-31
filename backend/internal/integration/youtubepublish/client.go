// Package youtubepublish publishes videos to YouTube as Shorts via the YouTube
// Data API v3 (OAuth2 + resumable upload). It is separate from the existing
// `youtube` package, which only reads trending videos with an API key.
package youtubepublish

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"mmo/pkg/config"
	"mmo/pkg/httpclient"
)

const youtubeUploadScope = "https://www.googleapis.com/auth/youtube.upload https://www.googleapis.com/auth/youtube.readonly"

type Client struct {
	clientID     string
	clientSecret string
	redirectURL  string
	categoryID   string
	privacy      string
	httpClient   *http.Client
	api          config.YouTubePublishAPIConfig
}

type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"`
}

type UserProfile struct {
	ChannelID string
	Title     string
	CustomURL string
	AvatarURL string
}

func New(cfg config.YouTubePublishConfig) *Client {
	category := cfg.DefaultCategoryID
	if category == "" {
		category = "22" // People & Blogs
	}
	privacy := cfg.PrivacyStatus
	if privacy == "" {
		privacy = "unlisted"
	}
	return &Client{
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		redirectURL:  cfg.RedirectURL,
		categoryID:   category,
		privacy:      privacy,
		httpClient:   httpclient.New("youtube_publish", cfg.HTTPTimeout),
		api:          cfg.API,
	}
}

// AuthURL builds the Google OAuth2 consent URL. access_type=offline + prompt=consent
// are REQUIRED to receive a refresh_token.
func (c *Client) AuthURL(state string) string {
	params := url.Values{
		"client_id":              {c.clientID},
		"redirect_uri":           {c.redirectURL},
		"response_type":          {"code"},
		"scope":                  {youtubeUploadScope},
		"access_type":            {"offline"},
		"prompt":                 {"consent"},
		"include_granted_scopes": {"true"},
		"state":                  {state},
	}
	return c.api.AuthBaseURL + "?" + params.Encode()
}

func (c *Client) ExchangeCode(ctx context.Context, code string) (*Tokens, error) {
	body := url.Values{
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {c.redirectURL},
	}
	return c.tokenRequest(ctx, body)
}

// RefreshToken exchanges a refresh token for a fresh access token. Google does
// NOT return a new refresh_token, so callers must preserve the existing one.
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*Tokens, error) {
	body := url.Values{
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
	}
	return c.tokenRequest(ctx, body)
}

func (c *Client) tokenRequest(ctx context.Context, body url.Values) (*Tokens, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.api.TokenURL, strings.NewReader(body.Encode()))
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
		Tokens
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("youtube token: decode: %w", err)
	}
	if result.Error != "" {
		return nil, fmt.Errorf("youtube token error: %s: %s", result.Error, result.ErrorDescription)
	}
	return &result.Tokens, nil
}

func (c *Client) GetUserInfo(ctx context.Context, accessToken string) (*UserProfile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.api.DataBaseURL+"/channels?part=snippet&mine=true", nil)
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
		Items []struct {
			ID      string `json:"id"`
			Snippet struct {
				Title      string `json:"title"`
				CustomURL  string `json:"customUrl"`
				Thumbnails struct {
					Default struct {
						URL string `json:"url"`
					} `json:"default"`
				} `json:"thumbnails"`
			} `json:"snippet"`
		} `json:"items"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, fmt.Errorf("youtube channels error: %s", result.Error.Message)
	}
	if len(result.Items) == 0 {
		return nil, fmt.Errorf("youtube: no channel for this account")
	}
	it := result.Items[0]
	custom := it.Snippet.CustomURL
	if custom == "" {
		custom = it.ID
	}
	return &UserProfile{
		ChannelID: it.ID,
		Title:     it.Snippet.Title,
		CustomURL: custom,
		AvatarURL: it.Snippet.Thumbnails.Default.URL,
	}, nil
}

type PostVideoRequest struct {
	VideoURL    string
	Title       string
	Description string
	Tags        []string
}

// PostVideo uploads the video at VideoURL to YouTube as a Short via a resumable
// upload, and returns the new video ID. #Shorts is ensured in title/description
// so YouTube classifies the vertical clip as a Short.
func (c *Client) PostVideo(ctx context.Context, accessToken string, r PostVideoRequest) (string, error) {
	title := ensureShorts(truncate(r.Title, 90))
	description := ensureShorts(r.Description)

	meta := map[string]any{
		"snippet": map[string]any{
			"title":       title,
			"description": description,
			"tags":        r.Tags,
			"categoryId":  c.categoryID,
		},
		"status": map[string]any{
			"privacyStatus":           c.privacy,
			"selfDeclaredMadeForKids": false,
		},
	}
	metaJSON, _ := json.Marshal(meta)

	// 1. Initiate resumable session.
	initURL := c.api.UploadBaseURL + "/videos?uploadType=resumable&part=snippet,status"
	initReq, err := http.NewRequestWithContext(ctx, http.MethodPost, initURL, bytes.NewReader(metaJSON))
	if err != nil {
		return "", err
	}
	initReq.Header.Set("Authorization", "Bearer "+accessToken)
	initReq.Header.Set("Content-Type", "application/json; charset=UTF-8")
	initReq.Header.Set("X-Upload-Content-Type", "video/mp4")

	initResp, err := c.httpClient.Do(initReq)
	if err != nil {
		return "", fmt.Errorf("youtube resumable init: %w", err)
	}
	defer initResp.Body.Close()
	if initResp.StatusCode/100 != 2 {
		raw, _ := io.ReadAll(initResp.Body)
		return "", fmt.Errorf("youtube resumable init failed (%d): %s", initResp.StatusCode, string(raw))
	}
	uploadURL := initResp.Header.Get("Location")
	if uploadURL == "" {
		return "", fmt.Errorf("youtube: no resumable upload URL returned")
	}

	// 2. Download the source bytes from R2 (need a known Content-Length for a
	//    single-PUT resumable upload).
	srcReq, err := http.NewRequestWithContext(ctx, http.MethodGet, r.VideoURL, nil)
	if err != nil {
		return "", err
	}
	srcResp, err := c.httpClient.Do(srcReq)
	if err != nil {
		return "", fmt.Errorf("youtube: fetch source video: %w", err)
	}
	defer srcResp.Body.Close()
	if srcResp.StatusCode/100 != 2 {
		return "", fmt.Errorf("youtube: fetch source video failed (%d)", srcResp.StatusCode)
	}
	videoBytes, err := io.ReadAll(srcResp.Body)
	if err != nil {
		return "", fmt.Errorf("youtube: read source video: %w", err)
	}

	// 3. Upload the bytes.
	upReq, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, bytes.NewReader(videoBytes))
	if err != nil {
		return "", err
	}
	upReq.Header.Set("Authorization", "Bearer "+accessToken)
	upReq.Header.Set("Content-Type", "video/mp4")
	upReq.ContentLength = int64(len(videoBytes))

	upResp, err := c.httpClient.Do(upReq)
	if err != nil {
		return "", fmt.Errorf("youtube resumable upload: %w", err)
	}
	defer upResp.Body.Close()

	raw, _ := io.ReadAll(upResp.Body)
	var result struct {
		ID    string `json:"id"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("youtube upload: decode response: %w", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("youtube upload error: %s", result.Error.Message)
	}
	if result.ID == "" {
		return "", fmt.Errorf("youtube upload: no video id returned: %s", string(raw))
	}
	return result.ID, nil
}

type VideoStats struct {
	ViewCount    int64
	LikeCount    int64
	CommentCount int64
}

func (c *Client) GetVideoStats(ctx context.Context, accessToken, videoID string) (*VideoStats, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.api.DataBaseURL+"/videos?part=statistics&id="+url.QueryEscape(videoID), nil)
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
		Items []struct {
			Statistics struct {
				ViewCount    string `json:"viewCount"`
				LikeCount    string `json:"likeCount"`
				CommentCount string `json:"commentCount"`
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
		return nil, fmt.Errorf("youtube stats error: %s", result.Error.Message)
	}
	if len(result.Items) == 0 {
		return &VideoStats{}, nil
	}
	s := result.Items[0].Statistics
	return &VideoStats{
		ViewCount:    parseInt(s.ViewCount),
		LikeCount:    parseInt(s.LikeCount),
		CommentCount: parseInt(s.CommentCount),
	}, nil
}

// WatchURL returns the canonical Shorts watch URL for a video ID.
func WatchURL(videoID string) string {
	return "https://www.youtube.com/shorts/" + videoID
}

func ensureShorts(s string) string {
	if strings.Contains(strings.ToLower(s), "#shorts") {
		return s
	}
	if s == "" {
		return "#Shorts"
	}
	return s + " #Shorts"
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

func parseInt(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}
