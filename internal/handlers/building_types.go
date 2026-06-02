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

type buildingTypeInput struct {
	BuildingTypeName string `json:"building_type_name"`
	BuildingTypeCode string `json:"building_type_code"`
}

// BuildingTypeHandler handles CRUD for public.building_type_tab.
type BuildingTypeHandler struct {
	db *sqlx.DB
}

// NewBuildingTypeHandler builds a BuildingTypeHandler.
func NewBuildingTypeHandler(db *sqlx.DB) *BuildingTypeHandler {
	return &BuildingTypeHandler{db: db}
}

// Register wires the building-type routes onto the given group.
func (h *BuildingTypeHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/references/building-types", h.List)
	rg.GET("/references/building-types/:id", h.Get)
	rg.POST("/references/building-types", h.Create)
	rg.PUT("/references/building-types/:id", h.Update)
	rg.DELETE("/references/building-types/:id", h.Delete)
}

func (h *BuildingTypeHandler) List(c *gin.Context) {
	items := []models.BuildingType{}
	const q = `SELECT building_type_id, building_type_name, building_type_code
		FROM public.building_type_tab ORDER BY building_type_id`
	if err := h.db.Select(&items, q); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *BuildingTypeHandler) Get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var item models.BuildingType
	const q = `SELECT building_type_id, building_type_name, building_type_code
		FROM public.building_type_tab WHERE building_type_id = $1`
	if err := h.db.Get(&item, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "building type not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *BuildingTypeHandler) Create(c *gin.Context) {
	var in buildingTypeInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if in.BuildingTypeName == "" || in.BuildingTypeCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "building_type_name and building_type_code are required"})
		return
	}

	var item models.BuildingType
	const q = `INSERT INTO public.building_type_tab (building_type_name, building_type_code)
		VALUES ($1, $2)
		RETURNING building_type_id, building_type_name, building_type_code`
	if err := h.db.Get(&item, q, in.BuildingTypeName, in.BuildingTypeCode); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *BuildingTypeHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var in buildingTypeInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if in.BuildingTypeName == "" || in.BuildingTypeCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "building_type_name and building_type_code are required"})
		return
	}

	var item models.BuildingType
	const q = `UPDATE public.building_type_tab
		SET building_type_name = $1, building_type_code = $2
		WHERE building_type_id = $3
		RETURNING building_type_id, building_type_name, building_type_code`
	if err := h.db.Get(&item, q, in.BuildingTypeName, in.BuildingTypeCode, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "building type not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *BuildingTypeHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	res, err := h.db.Exec(`DELETE FROM public.building_type_tab WHERE building_type_id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "building type not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
