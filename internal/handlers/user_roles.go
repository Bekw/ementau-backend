package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/ementau/ementau-backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type userRoleInput struct {
	UserRoleName string `json:"user_role_name"`
	UserRoleCode string `json:"user_role_code"`
}

// UserRoleHandler handles CRUD for admin.user_role_tab.
type UserRoleHandler struct {
	db *sqlx.DB
}

// NewUserRoleHandler builds a UserRoleHandler.
func NewUserRoleHandler(db *sqlx.DB) *UserRoleHandler {
	return &UserRoleHandler{db: db}
}

// Register wires the user-role routes onto the given group.
func (h *UserRoleHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/references/user-roles", h.List)
	rg.GET("/references/user-roles/:id", h.Get)
	rg.POST("/references/user-roles", h.Create)
	rg.PUT("/references/user-roles/:id", h.Update)
	rg.DELETE("/references/user-roles/:id", h.Delete)
}

func (h *UserRoleHandler) List(c *gin.Context) {
	items := []models.UserRole{}
	const q = `SELECT user_role_id, user_role_name, user_role_code
		FROM admin.user_role_tab ORDER BY user_role_id`
	if err := h.db.Select(&items, q); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *UserRoleHandler) Get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var item models.UserRole
	const q = `SELECT user_role_id, user_role_name, user_role_code
		FROM admin.user_role_tab WHERE user_role_id = $1`
	if err := h.db.Get(&item, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user role not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *UserRoleHandler) Create(c *gin.Context) {
	var in userRoleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if in.UserRoleName == "" || in.UserRoleCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_role_name and user_role_code are required"})
		return
	}

	var item models.UserRole
	const q = `INSERT INTO admin.user_role_tab (user_role_name, user_role_code)
		VALUES ($1, $2)
		RETURNING user_role_id, user_role_name, user_role_code`
	if err := h.db.Get(&item, q, in.UserRoleName, in.UserRoleCode); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *UserRoleHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var in userRoleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if in.UserRoleName == "" || in.UserRoleCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_role_name and user_role_code are required"})
		return
	}

	var item models.UserRole
	const q = `UPDATE admin.user_role_tab
		SET user_role_name = $1, user_role_code = $2
		WHERE user_role_id = $3
		RETURNING user_role_id, user_role_name, user_role_code`
	if err := h.db.Get(&item, q, in.UserRoleName, in.UserRoleCode, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user role not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *UserRoleHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	res, err := h.db.Exec(`DELETE FROM admin.user_role_tab WHERE user_role_id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "user role not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
