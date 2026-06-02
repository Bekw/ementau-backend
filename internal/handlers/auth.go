package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/ementau/ementau-backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
)

type loginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshInput struct {
	RefreshToken string `json:"refresh_token"`
}

// AuthHandler handles login and token refresh.
type AuthHandler struct {
	db     *sqlx.DB
	secret string
}

// NewAuthHandler builds an AuthHandler.
func NewAuthHandler(db *sqlx.DB, secret string) *AuthHandler {
	return &AuthHandler{db: db, secret: secret}
}

// Register wires the public auth routes onto the given group.
func (h *AuthHandler) Register(rg *gin.RouterGroup) {
	rg.POST("/auth/login", h.Login)
	rg.POST("/auth/refresh", h.Refresh)
}

// Login authenticates a user and issues access + refresh tokens.
func (h *AuthHandler) Login(c *gin.Context) {
	var in loginInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if in.Email == "" || in.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email and password are required"})
		return
	}

	var u models.User
	if err := h.db.Get(&u, `SELECT * FROM admin.user_tab WHERE email = $1`, in.Email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(in.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	accessToken, err := h.generateAccessToken(u.UserID, u.UserRoleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate access token"})
		return
	}
	refreshToken, err := h.generateRefreshToken(u.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user": gin.H{
			"user_id":      u.UserID,
			"fio":          u.FIO,
			"email":        u.Email,
			"phone_num":    u.PhoneNum,
			"user_role_id": u.UserRoleID,
		},
	})
}

// Refresh validates a refresh token and issues a new access token.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var in refreshInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if in.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token is required"})
		return
	}

	token, err := jwt.Parse(in.RefreshToken, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(h.secret), nil
	})
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
		return
	}
	if t, _ := claims["type"].(string); t != "refresh" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token type"})
		return
	}

	uidFloat, ok := claims["user_id"].(float64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
		return
	}
	userID := int(uidFloat)

	// Look up the current role so the new access token reflects up-to-date data.
	var userRoleID int
	if err := h.db.Get(&userRoleID, `SELECT user_role_id FROM admin.user_tab WHERE user_id = $1`, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	accessToken, err := h.generateAccessToken(userID, userRoleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate access token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"access_token": accessToken})
}

func (h *AuthHandler) generateAccessToken(userID, userRoleID int) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"user_id":      userID,
		"user_role_id": userRoleID,
		"type":         "access",
		"iat":          now.Unix(),
		"exp":          now.Add(accessTokenTTL).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(h.secret))
}

func (h *AuthHandler) generateRefreshToken(userID int) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"user_id": userID,
		"type":    "refresh",
		"iat":     now.Unix(),
		"exp":     now.Add(refreshTokenTTL).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(h.secret))
}
