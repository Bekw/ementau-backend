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

type materialTypeInput struct {
	MaterialTypeName string `json:"material_type_name"`
	MaterialTypeCode string `json:"material_type_code"`
}

// MaterialTypeHandler handles CRUD for public.material_type_tab.
type MaterialTypeHandler struct {
	db *sqlx.DB
}

// NewMaterialTypeHandler builds a MaterialTypeHandler.
func NewMaterialTypeHandler(db *sqlx.DB) *MaterialTypeHandler {
	return &MaterialTypeHandler{db: db}
}

// Register wires the material-type routes onto the given group.
func (h *MaterialTypeHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/references/material-types", h.List)
	rg.GET("/references/material-types/:id", h.Get)
	rg.POST("/references/material-types", h.Create)
	rg.PUT("/references/material-types/:id", h.Update)
	rg.DELETE("/references/material-types/:id", h.Delete)
}

func (h *MaterialTypeHandler) List(c *gin.Context) {
	items := []models.MaterialType{}
	const q = `SELECT material_type_id, material_type_name, material_type_code
		FROM public.material_type_tab ORDER BY material_type_id`
	if err := h.db.Select(&items, q); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *MaterialTypeHandler) Get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var item models.MaterialType
	const q = `SELECT material_type_id, material_type_name, material_type_code
		FROM public.material_type_tab WHERE material_type_id = $1`
	if err := h.db.Get(&item, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "material type not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *MaterialTypeHandler) Create(c *gin.Context) {
	var in materialTypeInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if in.MaterialTypeName == "" || in.MaterialTypeCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "material_type_name and material_type_code are required"})
		return
	}

	var item models.MaterialType
	const q = `INSERT INTO public.material_type_tab (material_type_name, material_type_code)
		VALUES ($1, $2)
		RETURNING material_type_id, material_type_name, material_type_code`
	if err := h.db.Get(&item, q, in.MaterialTypeName, in.MaterialTypeCode); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *MaterialTypeHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var in materialTypeInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if in.MaterialTypeName == "" || in.MaterialTypeCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "material_type_name and material_type_code are required"})
		return
	}

	var item models.MaterialType
	const q = `UPDATE public.material_type_tab
		SET material_type_name = $1, material_type_code = $2
		WHERE material_type_id = $3
		RETURNING material_type_id, material_type_name, material_type_code`
	if err := h.db.Get(&item, q, in.MaterialTypeName, in.MaterialTypeCode, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "material type not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *MaterialTypeHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	res, err := h.db.Exec(`DELETE FROM public.material_type_tab WHERE material_type_id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "material type not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
