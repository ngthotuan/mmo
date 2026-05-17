package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	apperr "mmo/pkg/errors"
)

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

const claimsKey = "claims"

func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, apperr.ErrUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, apperr.ErrInvalidToken
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, apperr.ErrInvalidToken)
			return
		}

		c.Set(claimsKey, claims)
		c.Next()
	}
}

func GetClaims(c *gin.Context) *Claims {
	v, _ := c.Get(claimsKey)
	claims, _ := v.(*Claims)
	return claims
}

func GetUserID(c *gin.Context) string {
	if claims := GetClaims(c); claims != nil {
		return claims.UserID
	}
	return ""
}
