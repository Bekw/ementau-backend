package handlers

import (
	"net/http"

	"github.com/ementau/ementau-backend/internal/config"
	"github.com/ementau/ementau-backend/internal/middleware"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// NewRouter builds the Gin engine with all routes registered.
func NewRouter(cfg *config.Config, db *sqlx.DB) *gin.Engine {
	r := gin.Default()

	// CORS — allow the frontend dev server to call the API.
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Serve locally-stored uploads (e.g. finance attachments).
	r.Static("/uploads", "./uploads")

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Public API v1 routes (no auth required).
	api := r.Group("/api/v1")
	authH := NewAuthHandler(db, cfg.JWTSecret)
	api.POST("/auth/login", authH.Login)
	api.POST("/auth/refresh", authH.Refresh)

	// Protected API v1 routes — require a valid access token.
	protected := r.Group("/api/v1")
	protected.Use(middleware.Auth(cfg.JWTSecret))

	NewUserHandler(db).Register(protected)
	NewUserRoleHandler(db).Register(protected)
	NewMaterialTypeHandler(db).Register(protected)
	NewBuildingTypeHandler(db).Register(protected)
	NewProjectTypeHandler(db).Register(protected)
	NewFinanceTypeHandler(db).Register(protected)
	NewProjectHandler(db).Register(protected)
	NewBuildingHandler(db).Register(protected)
	NewFinanceHandler(db).Register(protected)
	NewDashboardHandler(db).Register(protected)
	NewUnitTypeHandler(db).Register(protected)
	NewMaterialHandler(db).Register(protected)
	NewSupplyRequestHandler(db).Register(protected)
	NewSupplierHandler(db).Register(protected)
	NewMaterialSupplierHandler(db).Register(protected)
	NewProjectFileHandler(db).Register(protected)

	return r
}
