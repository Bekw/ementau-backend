package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/ementau/ementau-backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

const projectFileUploadDir = "./uploads/project_files"

// ProjectFileHandler handles file attachments for a project.
type ProjectFileHandler struct {
	db *sqlx.DB
}

// NewProjectFileHandler builds a ProjectFileHandler.
func NewProjectFileHandler(db *sqlx.DB) *ProjectFileHandler {
	return &ProjectFileHandler{db: db}
}

// Register wires the project-file routes onto the given group.
func (h *ProjectFileHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/projects/:id/files", h.List)
	rg.POST("/projects/:id/files", h.Upload)
	rg.DELETE("/projects/:id/files/:file_id", h.Delete)
}

func (h *ProjectFileHandler) List(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	items := []models.ProjectFile{}
	const q = `SELECT project_file_id, project_id, file_name, file_url, employee_id, rowversion
		FROM public.project_file_tab WHERE project_id = $1 ORDER BY rowversion DESC`
	if err := h.db.Select(&items, q, projectID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *ProjectFileHandler) Upload(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	fileHeader, ferr := c.FormFile("file")
	if ferr != nil || fileHeader == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Файл не выбран"})
		return
	}

	employeeID, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user context"})
		return
	}

	if err := os.MkdirAll(projectFileUploadDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create upload directory"})
		return
	}
	original := filepath.Base(fileHeader.Filename)
	saved := fmt.Sprintf("%d_%s", time.Now().UnixNano(), original)
	if err := c.SaveUploadedFile(fileHeader, filepath.Join(projectFileUploadDir, saved)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
		return
	}
	// URL path always uses forward slashes regardless of OS.
	fileURL := "/uploads/project_files/" + saved

	var item models.ProjectFile
	const q = `INSERT INTO public.project_file_tab
		(project_id, file_name, file_url, employee_id, rowversion)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING project_file_id, project_id, file_name, file_url, employee_id, rowversion`
	if err := h.db.Get(&item, q, projectID, original, fileURL, employeeID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *ProjectFileHandler) Delete(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}
	fileID, err := strconv.Atoi(c.Param("file_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file id"})
		return
	}

	res, err := h.db.Exec(
		`DELETE FROM public.project_file_tab WHERE project_file_id = $1 AND project_id = $2`,
		fileID, projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
