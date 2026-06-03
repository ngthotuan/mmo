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

// AuthSSE accepts a JWT from the Authorization header OR the ?token= query param.
// EventSource API in browsers cannot set custom headers, so SSE endpoints need this.
func AuthSSE(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := ""
		if t, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer "); ok {
			tokenStr = t
		} else {
			tokenStr = c.Query("token")
		}
		if tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, apperr.ErrUnauthorized)
			return
		}

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

// RequireRole aborts with 403 unless the authenticated user has one of the
// given roles. Must run after Auth().
func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(c *gin.Context) {
		claims := GetClaims(c)
		if claims == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, apperr.ErrUnauthorized)
			return
		}
		if _, ok := allowed[claims.Role]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, apperr.ErrForbidden)
			return
		}
		c.Next()
	}
}

// RequireFullAccess allows read-only (GET/HEAD/OPTIONS) requests for everyone but
// restricts mutating requests to roles with full access (admin/member). This is
// how view-only accounts are enforced across the whole API. Must run after Auth().
func RequireFullAccess(hasFullAccess func(role string) bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}
		claims := GetClaims(c)
		if claims == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, apperr.ErrUnauthorized)
			return
		}
		if !hasFullAccess(claims.Role) {
			c.AbortWithStatusJSON(http.StatusForbidden,
				apperr.New(http.StatusForbidden, "your account has view-only access; ask an admin to grant full access"))
			return
		}
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
