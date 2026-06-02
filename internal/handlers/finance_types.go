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

type financeTypeInput struct {
	FinanceTypeName string `json:"finance_type_name"`
	FinanceTypeCode string `json:"finance_type_code"`
	IsInvest        bool   `json:"is_invest"`
}

// FinanceTypeHandler handles CRUD for public.finance_type_tab.
type FinanceTypeHandler struct {
	db *sqlx.DB
}

// NewFinanceTypeHandler builds a FinanceTypeHandler.
func NewFinanceTypeHandler(db *sqlx.DB) *FinanceTypeHandler {
	return &FinanceTypeHandler{db: db}
}

// Register wires the finance-type routes onto the given group.
func (h *FinanceTypeHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/references/finance-types", h.List)
	rg.GET("/references/finance-types/:id", h.Get)
	rg.POST("/references/finance-types", h.Create)
	rg.PUT("/references/finance-types/:id", h.Update)
	rg.DELETE("/references/finance-types/:id", h.Delete)
}

func (h *FinanceTypeHandler) List(c *gin.Context) {
	items := []models.FinanceType{}
	const q = `SELECT finance_type_id, finance_type_name, finance_type_code, is_invest
		FROM public.finance_type_tab ORDER BY finance_type_id`
	if err := h.db.Select(&items, q); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *FinanceTypeHandler) Get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var item models.FinanceType
	const q = `SELECT finance_type_id, finance_type_name, finance_type_code, is_invest
		FROM public.finance_type_tab WHERE finance_type_id = $1`
	if err := h.db.Get(&item, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "finance type not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *FinanceTypeHandler) Create(c *gin.Context) {
	var in financeTypeInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Code is optional; only the name is required.
	if in.FinanceTypeName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "finance_type_name is required"})
		return
	}

	var item models.FinanceType
	const q = `INSERT INTO public.finance_type_tab (finance_type_name, finance_type_code, is_invest)
		VALUES ($1, $2, $3)
		RETURNING finance_type_id, finance_type_name, finance_type_code, is_invest`
	if err := h.db.Get(&item, q, in.FinanceTypeName, in.FinanceTypeCode, in.IsInvest); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *FinanceTypeHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var in financeTypeInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Code is optional; only the name is required.
	if in.FinanceTypeName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "finance_type_name is required"})
		return
	}

	var item models.FinanceType
	const q = `UPDATE public.finance_type_tab
		SET finance_type_name = $1, finance_type_code = $2, is_invest = $3
		WHERE finance_type_id = $4
		RETURNING finance_type_id, finance_type_name, finance_type_code, is_invest`
	if err := h.db.Get(&item, q, in.FinanceTypeName, in.FinanceTypeCode, in.IsInvest, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "finance type not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *FinanceTypeHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	res, err := h.db.Exec(`DELETE FROM public.finance_type_tab WHERE finance_type_id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "finance type not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
