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

type unitTypeInput struct {
	UnitTypeName string `json:"unit_type_name"`
	UnitTypeCode string `json:"unit_type_code"`
}

// UnitTypeHandler handles CRUD for public.unit_type_tab.
type UnitTypeHandler struct {
	db *sqlx.DB
}

// NewUnitTypeHandler builds a UnitTypeHandler.
func NewUnitTypeHandler(db *sqlx.DB) *UnitTypeHandler {
	return &UnitTypeHandler{db: db}
}

// Register wires the unit-type routes onto the given group.
func (h *UnitTypeHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/references/unit-types", h.List)
	rg.GET("/references/unit-types/:id", h.Get)
	rg.POST("/references/unit-types", h.Create)
	rg.PUT("/references/unit-types/:id", h.Update)
	rg.DELETE("/references/unit-types/:id", h.Delete)
}

func (h *UnitTypeHandler) List(c *gin.Context) {
	items := []models.UnitType{}
	const q = `SELECT unit_type_id, unit_type_name, unit_type_code
		FROM public.unit_type_tab ORDER BY unit_type_id`
	if err := h.db.Select(&items, q); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *UnitTypeHandler) Get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var item models.UnitType
	const q = `SELECT unit_type_id, unit_type_name, unit_type_code
		FROM public.unit_type_tab WHERE unit_type_id = $1`
	if err := h.db.Get(&item, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "unit type not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *UnitTypeHandler) Create(c *gin.Context) {
	var in unitTypeInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if in.UnitTypeName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unit_type_name is required"})
		return
	}

	var item models.UnitType
	const q = `INSERT INTO public.unit_type_tab (unit_type_name, unit_type_code)
		VALUES ($1, $2)
		RETURNING unit_type_id, unit_type_name, unit_type_code`
	if err := h.db.Get(&item, q, in.UnitTypeName, in.UnitTypeCode); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *UnitTypeHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var in unitTypeInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if in.UnitTypeName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unit_type_name is required"})
		return
	}

	var item models.UnitType
	const q = `UPDATE public.unit_type_tab
		SET unit_type_name = $1, unit_type_code = $2
		WHERE unit_type_id = $3
		RETURNING unit_type_id, unit_type_name, unit_type_code`
	if err := h.db.Get(&item, q, in.UnitTypeName, in.UnitTypeCode, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "unit type not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *UnitTypeHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	res, err := h.db.Exec(`DELETE FROM public.unit_type_tab WHERE unit_type_id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "unit type not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
