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

type projectTypeInput struct {
	ProjectTypeName string `json:"project_type_name"`
	ProjectTypeCode string `json:"project_type_code"`
}

// ProjectTypeHandler handles CRUD for public.project_type_tab.
type ProjectTypeHandler struct {
	db *sqlx.DB
}

// NewProjectTypeHandler builds a ProjectTypeHandler.
func NewProjectTypeHandler(db *sqlx.DB) *ProjectTypeHandler {
	return &ProjectTypeHandler{db: db}
}

// Register wires the project-type routes onto the given group.
func (h *ProjectTypeHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/references/project-types", h.List)
	rg.GET("/references/project-types/:id", h.Get)
	rg.POST("/references/project-types", h.Create)
	rg.PUT("/references/project-types/:id", h.Update)
	rg.DELETE("/references/project-types/:id", h.Delete)
}

func (h *ProjectTypeHandler) List(c *gin.Context) {
	items := []models.ProjectType{}
	const q = `SELECT project_type_id, project_type_name, project_type_code
		FROM public.project_type_tab ORDER BY project_type_id`
	if err := h.db.Select(&items, q); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *ProjectTypeHandler) Get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var item models.ProjectType
	const q = `SELECT project_type_id, project_type_name, project_type_code
		FROM public.project_type_tab WHERE project_type_id = $1`
	if err := h.db.Get(&item, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "project type not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *ProjectTypeHandler) Create(c *gin.Context) {
	var in projectTypeInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if in.ProjectTypeName == "" || in.ProjectTypeCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_type_name and project_type_code are required"})
		return
	}

	var item models.ProjectType
	const q = `INSERT INTO public.project_type_tab (project_type_name, project_type_code)
		VALUES ($1, $2)
		RETURNING project_type_id, project_type_name, project_type_code`
	if err := h.db.Get(&item, q, in.ProjectTypeName, in.ProjectTypeCode); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *ProjectTypeHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var in projectTypeInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if in.ProjectTypeName == "" || in.ProjectTypeCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_type_name and project_type_code are required"})
		return
	}

	var item models.ProjectType
	const q = `UPDATE public.project_type_tab
		SET project_type_name = $1, project_type_code = $2
		WHERE project_type_id = $3
		RETURNING project_type_id, project_type_name, project_type_code`
	if err := h.db.Get(&item, q, in.ProjectTypeName, in.ProjectTypeCode, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "project type not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *ProjectTypeHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	res, err := h.db.Exec(`DELETE FROM public.project_type_tab WHERE project_type_id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "project type not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
