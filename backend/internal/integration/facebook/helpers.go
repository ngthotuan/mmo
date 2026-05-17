package facebook

import (
	"context"
	"net/http"
)

func newFBRequest(ctx context.Context, method, url string) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, method, url, nil)
}
