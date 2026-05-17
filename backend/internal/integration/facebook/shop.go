package facebook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// CatalogProduct represents a product from a Facebook Product Catalog.
type CatalogProduct struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       string `json:"price"`
	Currency    string `json:"currency"`
	ImageURL    string `json:"image_url"`
	URL         string `json:"url"`
	Availability string `json:"availability"`
}

// ListCatalogProducts fetches products from a Facebook Business Product Catalog.
// pageToken is a page or user access token with catalog_management permission.
// catalogID is the Facebook Product Catalog ID stored in channel metadata.
func (c *Client) ListCatalogProducts(ctx context.Context, pageToken, catalogID string, limit int) ([]CatalogProduct, error) {
	if catalogID == "" {
		return nil, fmt.Errorf("facebook shop: catalog ID required")
	}
	params := url.Values{
		"fields":       {"id,name,description,price,currency,image_url,url,availability"},
		"limit":        {fmt.Sprintf("%d", limit)},
		"access_token": {pageToken},
	}
	req, err := newFBRequest(ctx, "GET",
		fmt.Sprintf("%s/%s/products?%s", c.graphBaseURL, catalogID, params.Encode()))
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data  []CatalogProduct `json:"data"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("facebook shop: decode response: %w", err)
	}
	if result.Error.Message != "" {
		return nil, fmt.Errorf("facebook shop error: %s", result.Error.Message)
	}
	return result.Data, nil
}
