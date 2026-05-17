package middleware

import (
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CORS(frontendURL string) gin.HandlerFunc {
	cfg := cors.Config{
		AllowOriginFunc: func(origin string) bool {
			if origin == frontendURL {
				return true
			}
			// Allow any localhost / 127.0.0.1 origin (any port) for local dev
			return strings.HasPrefix(origin, "http://localhost") ||
				strings.HasPrefix(origin, "http://127.0.0.1")
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}
	return cors.New(cfg)
}
