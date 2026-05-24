package httpclient

import (
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
	"mmo/pkg/logger"
)

// LoggingTransport wraps http.RoundTripper and logs every outbound request.
type LoggingTransport struct {
	Service string
	Base    http.RoundTripper
}

func (t *LoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	resp, err := base.RoundTrip(req)
	latency := time.Since(start)

	fields := []zap.Field{
		zap.String("service", t.Service),
		zap.String("method", req.Method),
		zap.String("url", sanitizeURL(req)),
		zap.Duration("latency", latency),
	}
	if err != nil {
		fields = append(fields, zap.Error(err))
		logger.Get().Warn("outbound http error", fields...)
		return resp, err
	}
	fields = append(fields, zap.Int("status", resp.StatusCode))
	if resp.StatusCode >= 400 {
		logger.Get().Warn("outbound http", fields...)
	} else {
		logger.Get().Info("outbound http", fields...)
	}
	return resp, nil
}

// New returns an *http.Client with logging transport for the given service name.
func New(service string, timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: &LoggingTransport{Service: service},
	}
}

// sanitizeURL returns the URL without query parameters that may contain secrets.
func sanitizeURL(req *http.Request) string {
	u := *req.URL
	return fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, u.Path)
}
