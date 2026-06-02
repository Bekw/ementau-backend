package handlers

import (
	"database/sql"
	"errors"
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

const materialUploadDir = "./uploads/materials"

// MaterialHandler handles CRUD for public.material_tab.
type MaterialHandler struct {
	db *sqlx.DB
}

// NewMaterialHandler builds a MaterialHandler.
func NewMaterialHandler(db *sqlx.DB) *MaterialHandler {
	return &MaterialHandler{db: db}
}

// Register wires the material routes onto the given group.
func (h *MaterialHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/materials", h.List)
	rg.GET("/materials/:id", h.Get)
	rg.POST("/materials", h.Create)
	rg.PUT("/materials/:id", h.Update)
	rg.DELETE("/materials/:id", h.Delete)
}

const materialColumns = `material_id, material_name, material_code, material_type_id,
	photo_name, photo_url, unit_type_id`

func (h *MaterialHandler) List(c *gin.Context) {
	items := []models.Material{}
	const q = `SELECT ` + materialColumns + ` FROM public.material_tab ORDER BY material_id`
	if err := h.db.Select(&items, q); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *MaterialHandler) Get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var item models.Material
	const q = `SELECT ` + materialColumns + ` FROM public.material_tab WHERE material_id = $1`
	if err := h.db.Get(&item, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "material not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

// savePhoto stores an optional "photo" file locally. Returns ok=false when no file
// was sent. On success it returns the original name and the public URL path.
func (h *MaterialHandler) savePhoto(c *gin.Context) (name, url string, ok bool, err error) {
	fileHeader, ferr := c.FormFile("photo")
	if ferr != nil || fileHeader == nil {
		return "", "", false, nil
	}
	if err := os.MkdirAll(materialUploadDir, 0o755); err != nil {
		return "", "", false, err
	}
	original := filepath.Base(fileHeader.Filename)
	saved := fmt.Sprintf("%d_%s", time.Now().UnixNano(), original)
	if err := c.SaveUploadedFile(fileHeader, filepath.Join(materialUploadDir, saved)); err != nil {
		return "", "", false, err
	}
	// URL path always uses forward slashes regardless of OS.
	return original, "/uploads/materials/" + saved, true, nil
}

func (h *MaterialHandler) Create(c *gin.Context) {
	name := c.PostForm("material_name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "material_name is required"})
		return
	}
	code := c.PostForm("material_code")

	materialTypeID, err := strconv.Atoi(c.PostForm("material_type_id"))
	if err != nil || materialTypeID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "material_type_id is required"})
		return
	}
	unitTypeID, err := strconv.Atoi(c.PostForm("unit_type_id"))
	if err != nil || unitTypeID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unit_type_id is required"})
		return
	}

	photoName, photoURL, _, perr := h.savePhoto(c)
	if perr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save photo"})
		return
	}

	var item models.Material
	const q = `INSERT INTO public.material_tab
		(material_name, material_code, material_type_id, photo_name, photo_url, unit_type_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING ` + materialColumns
	if err := h.db.Get(&item, q, name, code, materialTypeID, photoName, photoURL, unitTypeID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *MaterialHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	name := c.PostForm("material_name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "material_name is required"})
		return
	}
	code := c.PostForm("material_code")

	materialTypeID, err := strconv.Atoi(c.PostForm("material_type_id"))
	if err != nil || materialTypeID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "material_type_id is required"})
		return
	}
	unitTypeID, err := strconv.Atoi(c.PostForm("unit_type_id"))
	if err != nil || unitTypeID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unit_type_id is required"})
		return
	}

	photoName, photoURL, hasPhoto, perr := h.savePhoto(c)
	if perr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save photo"})
		return
	}

	var item models.Material
	var q string
	var args []interface{}
	if hasPhoto {
		q = `UPDATE public.material_tab
			SET material_name = $1, material_code = $2, material_type_id = $3,
				unit_type_id = $4, photo_name = $5, photo_url = $6
			WHERE material_id = $7
			RETURNING ` + materialColumns
		args = []interface{}{name, code, materialTypeID, unitTypeID, photoName, photoURL, id}
	} else {
		// No new photo — leave the existing photo columns untouched.
		q = `UPDATE public.material_tab
			SET material_name = $1, material_code = $2, material_type_id = $3,
				unit_type_id = $4
			WHERE material_id = $5
			RETURNING ` + materialColumns
		args = []interface{}{name, code, materialTypeID, unitTypeID, id}
	}

	if err := h.db.Get(&item, q, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "material not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *MaterialHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	res, err := h.db.Exec(`DELETE FROM public.material_tab WHERE material_id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "material not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
