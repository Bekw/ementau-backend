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

type projectInput struct {
	ProjectName   string `json:"project_name"`
	ProjectTypeID int    `json:"project_type_id"`
}

// ProjectHandler handles CRUD for public.project_tab.
type ProjectHandler struct {
	db *sqlx.DB
}

// NewProjectHandler builds a ProjectHandler.
func NewProjectHandler(db *sqlx.DB) *ProjectHandler {
	return &ProjectHandler{db: db}
}

// Register wires the project routes onto the given group.
func (h *ProjectHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/projects", h.List)
	rg.GET("/projects/:id", h.Get)
	rg.POST("/projects", h.Create)
	rg.PUT("/projects/:id", h.Update)
	rg.DELETE("/projects/:id", h.Delete)
}

func (h *ProjectHandler) List(c *gin.Context) {
	items := []models.Project{}
	const q = `SELECT project_id, project_name, project_type_id, employee_id, rowversion
		FROM public.project_tab ORDER BY project_id`
	if err := h.db.Select(&items, q); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *ProjectHandler) Get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var item models.Project
	const q = `SELECT project_id, project_name, project_type_id, employee_id, rowversion
		FROM public.project_tab WHERE project_id = $1`
	if err := h.db.Get(&item, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *ProjectHandler) Create(c *gin.Context) {
	var in projectInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if in.ProjectName == "" || in.ProjectTypeID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_name and project_type_id are required"})
		return
	}

	// employee_id comes from the authenticated user (set by the JWT middleware).
	employeeID, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user context"})
		return
	}

	var item models.Project
	const q = `INSERT INTO public.project_tab (project_name, project_type_id, employee_id)
		VALUES ($1, $2, $3)
		RETURNING project_id, project_name, project_type_id, employee_id, rowversion`
	if err := h.db.Get(&item, q, in.ProjectName, in.ProjectTypeID, employeeID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *ProjectHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var in projectInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if in.ProjectName == "" || in.ProjectTypeID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_name and project_type_id are required"})
		return
	}

	// Only project_name and project_type_id are mutable; employee_id and rowversion stay as-is.
	var item models.Project
	const q = `UPDATE public.project_tab
		SET project_name = $1, project_type_id = $2
		WHERE project_id = $3
		RETURNING project_id, project_name, project_type_id, employee_id, rowversion`
	if err := h.db.Get(&item, q, in.ProjectName, in.ProjectTypeID, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *ProjectHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	res, err := h.db.Exec(`DELETE FROM public.project_tab WHERE project_id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// currentUserID reads the user_id stashed in the context by the JWT middleware.
// JWT numeric claims decode as float64, so c.GetInt would yield 0 — convert explicitly.
func currentUserID(c *gin.Context) (int, bool) {
	v, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	switch id := v.(type) {
	case float64:
		return int(id), true
	case int:
		return id, true
	default:
		return 0, false
	}
}
