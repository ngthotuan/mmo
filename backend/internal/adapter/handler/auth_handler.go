package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	apperr "mmo/pkg/errors"
	"mmo/pkg/logger"
	"mmo/pkg/middleware"
)

type AuthHandler struct {
	db         *sqlx.DB
	jwtSecret  string
	accessTTL  time.Duration
	refreshTTL time.Duration
	log        *zap.Logger
}

func NewAuthHandler(db *sqlx.DB, jwtSecret string, accessTTL, refreshTTL time.Duration) *AuthHandler {
	return &AuthHandler{
		db:         db,
		jwtSecret:  jwtSecret,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		log:        logger.Get(),
	}
}

type registerRequest struct {
	Name     string `json:"name"     binding:"required,min=2"`
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type loginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // seconds
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apperr.WithDetail(http.StatusBadRequest, "validation error", err.Error()))
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		h.log.Error("register: bcrypt failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, apperr.ErrInternalServer)
		return
	}

	var userID uuid.UUID
	err = h.db.QueryRowContext(c.Request.Context(),
		`INSERT INTO users (email, password_hash, name) VALUES ($1, $2, $3) RETURNING id`,
		req.Email, string(hash), req.Name,
	).Scan(&userID)
	if err != nil {
		if isUniqueViolation(err) {
			c.JSON(http.StatusConflict, apperr.ErrConflict)
			return
		}
		h.log.Error("register: db insert failed", zap.String("email", req.Email), zap.Error(err))
		c.JSON(http.StatusInternalServerError, apperr.ErrInternalServer)
		return
	}

	tokens, err := h.generateTokens(userID.String(), req.Email, "owner")
	if err != nil {
		h.log.Error("register: generate tokens failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, apperr.ErrInternalServer)
		return
	}

	c.JSON(http.StatusCreated, tokens)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apperr.WithDetail(http.StatusBadRequest, "validation error", err.Error()))
		return
	}

	var user struct {
		ID           uuid.UUID `db:"id"`
		PasswordHash string    `db:"password_hash"`
		Role         string    `db:"role"`
	}
	if err := h.db.GetContext(c.Request.Context(), &user,
		`SELECT id, password_hash, role FROM users WHERE email = $1`, req.Email,
	); err != nil {
		c.JSON(http.StatusUnauthorized, apperr.ErrInvalidCredential)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, apperr.ErrInvalidCredential)
		return
	}

	tokens, err := h.generateTokens(user.ID.String(), req.Email, user.Role)
	if err != nil {
		h.log.Error("login: generate tokens failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, apperr.ErrInternalServer)
		return
	}

	c.JSON(http.StatusOK, tokens)
}

func (h *AuthHandler) Me(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, apperr.ErrUnauthorized)
		return
	}

	var user struct {
		ID        uuid.UUID `db:"id"         json:"id"`
		Email     string    `db:"email"       json:"email"`
		Name      string    `db:"name"        json:"name"`
		Role      string    `db:"role"        json:"role"`
		CreatedAt time.Time `db:"created_at"  json:"created_at"`
	}
	if err := h.db.GetContext(c.Request.Context(), &user,
		`SELECT id, email, name, role, created_at FROM users WHERE id = $1`, claims.UserID,
	); err != nil {
		c.JSON(http.StatusNotFound, apperr.ErrNotFound)
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, apperr.ErrUnauthorized)
		return
	}
	var body struct {
		Name string `json:"name" binding:"required,min=2"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}
	if _, err := h.db.ExecContext(c.Request.Context(),
		`UPDATE users SET name=$1, updated_at=NOW() WHERE id=$2`, body.Name, claims.UserID,
	); err != nil {
		h.log.Error("update profile: db failed", zap.String("user_id", claims.UserID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, apperr.ErrInternalServer)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "profile updated"})
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, apperr.ErrUnauthorized)
		return
	}
	var body struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password"     binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}

	var hash string
	if err := h.db.QueryRowContext(c.Request.Context(),
		`SELECT password_hash FROM users WHERE id=$1`, claims.UserID,
	).Scan(&hash); err != nil {
		h.log.Error("change password: fetch hash failed", zap.String("user_id", claims.UserID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, apperr.ErrInternalServer)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(body.CurrentPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, apperr.New(http.StatusUnauthorized, "current password is incorrect"))
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(body.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		h.log.Error("change password: bcrypt failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, apperr.ErrInternalServer)
		return
	}
	if _, err := h.db.ExecContext(c.Request.Context(),
		`UPDATE users SET password_hash=$1, updated_at=NOW() WHERE id=$2`, string(newHash), claims.UserID,
	); err != nil {
		h.log.Error("change password: db update failed", zap.String("user_id", claims.UserID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, apperr.ErrInternalServer)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "password changed"})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var body struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, apperr.ErrBadRequest)
		return
	}

	claims := &middleware.Claims{}
	token, err := jwt.ParseWithClaims(body.RefreshToken, claims, func(t *jwt.Token) (any, error) {
		return []byte(h.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, apperr.ErrInvalidToken)
		return
	}

	tokens, err := h.generateTokens(claims.UserID, claims.Email, claims.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apperr.ErrInternalServer)
		return
	}
	c.JSON(http.StatusOK, tokens)
}

func (h *AuthHandler) generateTokens(userID, email, role string) (*tokenResponse, error) {
	now := time.Now()

	accessClaims := &middleware.Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(h.accessTTL)),
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(h.jwtSecret))
	if err != nil {
		return nil, err
	}

	refreshClaims := &middleware.Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(h.refreshTTL)),
		},
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(h.jwtSecret))
	if err != nil {
		return nil, err
	}

	return &tokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(h.accessTTL.Seconds()),
	}, nil
}

func isUniqueViolation(err error) bool {
	return err != nil && len(err.Error()) > 0 &&
		(contains(err.Error(), "unique") || contains(err.Error(), "duplicate"))
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
