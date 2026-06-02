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

type supplierInput struct {
	SupplierName    string `json:"supplier_name"`
	SupplierPhone   string `json:"supplier_phone"`
	SupplierAddress string `json:"supplier_address"`
}

// SupplierHandler handles CRUD for public.supplier_tab.
type SupplierHandler struct {
	db *sqlx.DB
}

// NewSupplierHandler builds a SupplierHandler.
func NewSupplierHandler(db *sqlx.DB) *SupplierHandler {
	return &SupplierHandler{db: db}
}

// Register wires the supplier routes onto the given group.
func (h *SupplierHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/references/suppliers", h.List)
	rg.GET("/references/suppliers/:id", h.Get)
	rg.POST("/references/suppliers", h.Create)
	rg.PUT("/references/suppliers/:id", h.Update)
	rg.DELETE("/references/suppliers/:id", h.Delete)
}

const supplierColumns = `supplier_id, supplier_name, supplier_phone, supplier_address`

func (h *SupplierHandler) List(c *gin.Context) {
	items := []models.Supplier{}
	const q = `SELECT ` + supplierColumns + ` FROM public.supplier_tab ORDER BY supplier_id`
	if err := h.db.Select(&items, q); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *SupplierHandler) Get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var item models.Supplier
	const q = `SELECT ` + supplierColumns + ` FROM public.supplier_tab WHERE supplier_id = $1`
	if err := h.db.Get(&item, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "supplier not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *SupplierHandler) Create(c *gin.Context) {
	var in supplierInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if in.SupplierName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "supplier_name is required"})
		return
	}

	var item models.Supplier
	const q = `INSERT INTO public.supplier_tab (supplier_name, supplier_phone, supplier_address)
		VALUES ($1, $2, $3)
		RETURNING ` + supplierColumns
	if err := h.db.Get(&item, q, in.SupplierName, in.SupplierPhone, in.SupplierAddress); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *SupplierHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var in supplierInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if in.SupplierName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "supplier_name is required"})
		return
	}

	var item models.Supplier
	const q = `UPDATE public.supplier_tab
		SET supplier_name = $1, supplier_phone = $2, supplier_address = $3
		WHERE supplier_id = $4
		RETURNING ` + supplierColumns
	if err := h.db.Get(&item, q, in.SupplierName, in.SupplierPhone, in.SupplierAddress, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "supplier not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *SupplierHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	res, err := h.db.Exec(`DELETE FROM public.supplier_tab WHERE supplier_id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "supplier not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
