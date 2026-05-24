package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"mmo/pkg/logger"
)

func Logger() gin.HandlerFunc {
	log := logger.Get()
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		status := c.Writer.Status()
		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("query", c.Request.URL.RawQuery),
			zap.Int("status", status),
			zap.Duration("latency", time.Since(start)),
			zap.Int("size", c.Writer.Size()),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		}
		if errMsg := c.Errors.ByType(gin.ErrorTypePrivate).String(); errMsg != "" {
			fields = append(fields, zap.String("error", errMsg))
		}
		switch {
		case status >= 500:
			log.Error("request", fields...)
		case status >= 400:
			log.Warn("request", fields...)
		default:
			log.Info("request", fields...)
		}
	}
}
