package facebook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"mmo/pkg/config"
	"mmo/pkg/httpclient"
)

type Client struct {
	appID        string
	appSecret    string
	redirectURL  string
	authBaseURL  string
	tokenURL     string
	graphBaseURL string
	httpTimeout  time.Duration
	httpClient   *http.Client
}

type Tokens struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type Page struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	AccessToken string `json:"access_token"`
	Picture     string `json:"-"`
}

func New(cfg config.FacebookConfig) *Client {
	return &Client{
		appID:        cfg.AppID,
		appSecret:    cfg.AppSecret,
		redirectURL:  cfg.RedirectURL,
		authBaseURL:  cfg.API.AuthBaseURL,
		tokenURL:     cfg.API.TokenURL,
		graphBaseURL: cfg.API.GraphBaseURL,
		httpTimeout:  cfg.HTTPTimeout,
		httpClient:   httpclient.New("facebook", cfg.HTTPTimeout),
	}
}

func (c *Client) AuthURL(state string) string {
	params := url.Values{
		"client_id":     {c.appID},
		"redirect_uri":  {c.redirectURL},
		"scope":         {"pages_manage_posts,pages_read_engagement,pages_show_list"},
		"response_type": {"code"},
		"state":         {state},
	}
	return c.authBaseURL + "?" + params.Encode()
}

func (c *Client) ExchangeCode(ctx context.Context, code string) (*Tokens, error) {
	params := url.Values{
		"client_id":     {c.appID},
		"client_secret": {c.appSecret},
		"redirect_uri":  {c.redirectURL},
		"code":          {code},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.tokenURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tokens Tokens
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		return nil, err
	}
	if tokens.AccessToken == "" {
		return nil, fmt.Errorf("facebook exchange code failed: empty access token")
	}
	return &tokens, nil
}

func (c *Client) GetLongLivedToken(ctx context.Context, shortLived string) (string, error) {
	params := url.Values{
		"grant_type":        {"fb_exchange_token"},
		"client_id":         {c.appID},
		"client_secret":     {c.appSecret},
		"fb_exchange_token": {shortLived},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.tokenURL+"?"+params.Encode(), nil)
	if err != nil {
		return "", err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.AccessToken, nil
}

func (c *Client) ListPages(ctx context.Context, userAccessToken string) ([]Page, error) {
	params := url.Values{
		"fields":       {"id,name,access_token"},
		"access_token": {userAccessToken},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.graphBaseURL+"/me/accounts?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data []Page `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

func (c *Client) GetPageToken(ctx context.Context, userAccessToken, pageID string) (*Page, error) {
	pages, err := c.ListPages(ctx, userAccessToken)
	if err != nil {
		return nil, err
	}
	for i := range pages {
		if pages[i].ID == pageID {
			pic, _ := c.getPagePicture(ctx, pageID, pages[i].AccessToken)
			pages[i].Picture = pic
			return &pages[i], nil
		}
	}
	return nil, fmt.Errorf("page %s not found in user's pages", pageID)
}

func (c *Client) getPagePicture(ctx context.Context, pageID, pageToken string) (string, error) {
	params := url.Values{
		"fields":       {"picture"},
		"access_token": {pageToken},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/%s?%s", c.graphBaseURL, pageID, params.Encode()), nil)
	if err != nil {
		return "", err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Picture struct {
			Data struct {
				URL string `json:"url"`
			} `json:"data"`
		} `json:"picture"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	return result.Picture.Data.URL, nil
}

type PostVideoRequest struct {
	PageID      string
	VideoURL    string
	Description string
	ProductURLs []string
}

func (c *Client) PostVideo(ctx context.Context, pageToken string, r PostVideoRequest) (string, error) {
	desc := r.Description
	if len(r.ProductURLs) > 0 {
		desc += "\n\n🛍️ Shop:\n"
		for _, u := range r.ProductURLs {
			desc += u + "\n"
		}
	}

	params := url.Values{
		"file_url":     {r.VideoURL},
		"description":  {desc},
		"access_token": {pageToken},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/%s/videos?%s", c.graphBaseURL, r.PageID, params.Encode()), nil)
	if err != nil {
		return "", err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.ID == "" {
		return "", fmt.Errorf("facebook post video: empty post ID")
	}
	return result.ID, nil
}

type VideoStats struct {
	Views    int64 `json:"views"`
	Likes    int64 `json:"likes"`
	Comments int64 `json:"comments"`
	Shares   int64 `json:"shares"`
	Reach    int64 `json:"reach"`
}

func (c *Client) GetVideoStats(ctx context.Context, pageToken, postID string) (*VideoStats, error) {
	params := url.Values{
		"fields":       {"insights.metric(post_impressions,post_engaged_users,post_clicks)"},
		"access_token": {pageToken},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/%s?%s", c.graphBaseURL, postID, params.Encode()), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)

	return &VideoStats{}, nil
}
