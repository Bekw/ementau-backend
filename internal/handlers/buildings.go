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

type buildingInput struct {
	BuildingName    string `json:"building_name"`
	BuildingTypeID  int    `json:"building_type_id"`
	BuildingAddress string `json:"building_address"`
}

type buildingUpdateInput struct {
	BuildingAddress string  `json:"building_address"`
	DateStart       *string `json:"date_start"`
	DateEnd         *string `json:"date_end"`
}

// BuildingHandler handles CRUD for public.building_tab, scoped to a project.
type BuildingHandler struct {
	db *sqlx.DB
}

// NewBuildingHandler builds a BuildingHandler.
func NewBuildingHandler(db *sqlx.DB) *BuildingHandler {
	return &BuildingHandler{db: db}
}

// Register wires the building routes onto the given group.
// The project segment uses :id to stay consistent with the existing /projects/:id
// routes — Gin requires the same wildcard name at the same path position.
func (h *BuildingHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/projects/:id/buildings", h.List)
	rg.POST("/projects/:id/buildings", h.Create)
	rg.PUT("/projects/:id/buildings/:building_id", h.Update)
	rg.DELETE("/projects/:id/buildings/:building_id", h.Delete)
	// Standalone fetch by id — needed by the building card page, which only
	// knows the building id (the project id comes from the returned record).
	rg.GET("/buildings/:building_id", h.GetByID)
	// All buildings across every project — for the standalone buildings list page.
	rg.GET("/buildings", h.ListAll)
}

func (h *BuildingHandler) ListAll(c *gin.Context) {
	items := []models.Building{}
	const q = `SELECT building_id, building_name, building_type_id, building_address,
		employee_id, date_start, date_end, rowversion, project_id
		FROM public.building_tab ORDER BY building_id DESC`
	if err := h.db.Select(&items, q); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *BuildingHandler) GetByID(c *gin.Context) {
	buildingID, err := strconv.Atoi(c.Param("building_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid building id"})
		return
	}

	var item models.Building
	const q = `SELECT building_id, building_name, building_type_id, building_address,
		employee_id, date_start, date_end, rowversion, project_id
		FROM public.building_tab WHERE building_id = $1`
	if err := h.db.Get(&item, q, buildingID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "building not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *BuildingHandler) List(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	items := []models.Building{}
	const q = `SELECT building_id, building_name, building_type_id, building_address,
		employee_id, date_start, date_end, rowversion, project_id
		FROM public.building_tab WHERE project_id = $1 ORDER BY building_id`
	if err := h.db.Select(&items, q, projectID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *BuildingHandler) Create(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	var in buildingInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if in.BuildingName == "" || in.BuildingTypeID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "building_name and building_type_id are required"})
		return
	}

	// employee_id comes from the authenticated user (set by the JWT middleware).
	employeeID, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user context"})
		return
	}

	var item models.Building
	const q = `INSERT INTO public.building_tab
		(building_name, building_type_id, building_address, employee_id, project_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING building_id, building_name, building_type_id, building_address,
			employee_id, date_start, date_end, rowversion, project_id`
	if err := h.db.Get(&item, q,
		in.BuildingName, in.BuildingTypeID, in.BuildingAddress, employeeID, projectID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *BuildingHandler) Update(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}
	buildingID, err := strconv.Atoi(c.Param("building_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid building id"})
		return
	}

	var in buildingUpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Treat empty-string dates as NULL so they don't fail the timestamp cast.
	if in.DateStart != nil && *in.DateStart == "" {
		in.DateStart = nil
	}
	if in.DateEnd != nil && *in.DateEnd == "" {
		in.DateEnd = nil
	}

	// Only address and the two dates are editable here. Empty dates become NULL.
	var item models.Building
	const q = `UPDATE public.building_tab
		SET building_address = $1, date_start = $2, date_end = $3
		WHERE building_id = $4 AND project_id = $5
		RETURNING building_id, building_name, building_type_id, building_address,
			employee_id, date_start, date_end, rowversion, project_id`
	if err := h.db.Get(&item, q, in.BuildingAddress, in.DateStart, in.DateEnd, buildingID, projectID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "building not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *BuildingHandler) Delete(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}
	buildingID, err := strconv.Atoi(c.Param("building_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid building id"})
		return
	}

	res, err := h.db.Exec(
		`DELETE FROM public.building_tab WHERE building_id = $1 AND project_id = $2`,
		buildingID, projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "building not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
