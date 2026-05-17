package tiktok

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func newShopRequest(ctx context.Context, method, url, token string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// ShopProduct represents a product from TikTok Shop.
type ShopProduct struct {
	ProductID     string `json:"product_id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Price         string `json:"price"`
	Currency      string `json:"currency"`
	CoverImageURL string `json:"cover_image_url"`
	ProductURL    string `json:"product_url"`
	Status        string `json:"status"`
}

// ListShopProducts fetches products from TikTok Shop using the shop access token.
// shopToken is obtained through TikTok Shop seller authorization.
func (c *Client) ListShopProducts(ctx context.Context, shopToken string, page, pageSize int) ([]ShopProduct, int, error) {
	if shopToken == "" {
		return nil, 0, fmt.Errorf("tiktok shop: access token required")
	}
	params := url.Values{
		"page_number": {fmt.Sprintf("%d", page)},
		"page_size":   {fmt.Sprintf("%d", pageSize)},
	}
	req, err := newShopRequest(ctx, "GET", c.api.ShopBaseURL+"/products?"+params.Encode(), shopToken)
	if err != nil {
		return nil, 0, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Products []ShopProduct `json:"products"`
			Total    int           `json:"total"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, 0, fmt.Errorf("tiktok shop: decode response: %w", err)
	}
	if result.Code != 0 {
		return nil, 0, fmt.Errorf("tiktok shop error %d: %s", result.Code, result.Message)
	}
	return result.Data.Products, result.Data.Total, nil
}
