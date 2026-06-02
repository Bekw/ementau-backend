package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/ementau/ementau-backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type materialSupplierInput struct {
	SupplierID int `json:"supplier_id"`
}

// MaterialSupplierHandler manages material↔supplier links.
type MaterialSupplierHandler struct {
	db *sqlx.DB
}

// NewMaterialSupplierHandler builds a MaterialSupplierHandler.
func NewMaterialSupplierHandler(db *sqlx.DB) *MaterialSupplierHandler {
	return &MaterialSupplierHandler{db: db}
}

// Register wires the material-supplier link routes onto the given group.
// Uses :id for the material segment to match the existing /materials/:id routes.
func (h *MaterialSupplierHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/materials/:id/suppliers", h.List)
	rg.POST("/materials/:id/suppliers", h.Link)
	rg.DELETE("/materials/:id/suppliers/:supplier_id", h.Unlink)
}

func (h *MaterialSupplierHandler) List(c *gin.Context) {
	materialID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid material id"})
		return
	}

	items := []models.Supplier{}
	const q = `SELECT s.supplier_id, s.supplier_name, s.supplier_phone, s.supplier_address
		FROM public.supplier_tab s
		JOIN public.material_supplier_tab ms ON ms.supplier_id = s.supplier_id
		WHERE ms.material_id = $1
		ORDER BY s.supplier_name`
	if err := h.db.Select(&items, q, materialID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *MaterialSupplierHandler) Link(c *gin.Context) {
	materialID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid material id"})
		return
	}

	var in materialSupplierInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if in.SupplierID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "supplier_id is required"})
		return
	}

	_, err = h.db.Exec(
		`INSERT INTO public.material_supplier_tab (material_id, supplier_id) VALUES ($1, $2)`,
		materialID, in.SupplierID)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "Поставщик уже привязан к материалу"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "linked"})
}

func (h *MaterialSupplierHandler) Unlink(c *gin.Context) {
	materialID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid material id"})
		return
	}
	supplierID, err := strconv.Atoi(c.Param("supplier_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid supplier id"})
		return
	}

	res, err := h.db.Exec(
		`DELETE FROM public.material_supplier_tab WHERE material_id = $1 AND supplier_id = $2`,
		materialID, supplierID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "unlinked"})
}
