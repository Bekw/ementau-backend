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

const financeUploadDir = "./uploads/finances"

const financeColumns = `finance_id, finance_type_id, finance_description, finance_file_name,
	finance_file_url, employee_id, rowversion, project_id, building_id, amount`

// balanceTotals is scanned from the balance aggregate queries.
type balanceTotals struct {
	IncomeTotal  float64 `db:"income_total"`
	ExpenseTotal float64 `db:"expense_total"`
}

// FinanceHandler handles finances (scoped to a building or directly to a project)
// and balance aggregates.
type FinanceHandler struct {
	db *sqlx.DB
}

// NewFinanceHandler builds a FinanceHandler.
func NewFinanceHandler(db *sqlx.DB) *FinanceHandler {
	return &FinanceHandler{db: db}
}

// Register wires the finance and balance routes onto the given group.
func (h *FinanceHandler) Register(rg *gin.RouterGroup) {
	// Building-scoped.
	rg.GET("/buildings/:building_id/finances", h.List)
	rg.POST("/buildings/:building_id/finances", h.Create)
	rg.DELETE("/buildings/:building_id/finances/:finance_id", h.Delete)
	rg.GET("/buildings/:building_id/balance", h.BuildingBalance)
	// Project-scoped.
	rg.GET("/projects/:id/finances", h.ListByProject)
	rg.POST("/projects/:id/finances", h.CreateForProject)
	rg.DELETE("/projects/:id/finances/:finance_id", h.DeleteByProject)
	rg.GET("/projects/:id/balance", h.ProjectBalance)
	// Attach/replace a receipt file on an existing finance record.
	rg.PUT("/finances/:finance_id/file", h.UpdateFile)
}

// parseFinanceForm reads the shared multipart fields used by both create paths.
func parseFinanceForm(c *gin.Context) (financeTypeID int, description string, amount float64, ok bool) {
	financeTypeID, err := strconv.Atoi(c.PostForm("finance_type_id"))
	if err != nil || financeTypeID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "finance_type_id is required"})
		return 0, "", 0, false
	}
	description = c.PostForm("finance_description")
	if description == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "finance_description is required"})
		return 0, "", 0, false
	}
	if raw := c.PostForm("amount"); raw != "" {
		amount, err = strconv.ParseFloat(raw, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
			return 0, "", 0, false
		}
	}
	return financeTypeID, description, amount, true
}

// saveFinanceFile stores an optional "file" upload locally; returns empty strings if none.
func saveFinanceFile(c *gin.Context) (name, url string, err error) {
	fileHeader, ferr := c.FormFile("file")
	if ferr != nil || fileHeader == nil {
		return "", "", nil
	}
	if err := os.MkdirAll(financeUploadDir, 0o755); err != nil {
		return "", "", err
	}
	original := filepath.Base(fileHeader.Filename)
	saved := fmt.Sprintf("%d_%s", time.Now().UnixNano(), original)
	if err := c.SaveUploadedFile(fileHeader, filepath.Join(financeUploadDir, saved)); err != nil {
		return "", "", err
	}
	// URL path always uses forward slashes regardless of OS.
	return original, "/uploads/finances/" + saved, nil
}

func (h *FinanceHandler) List(c *gin.Context) {
	buildingID, err := strconv.Atoi(c.Param("building_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid building id"})
		return
	}

	items := []models.Finance{}
	const q = `SELECT ` + financeColumns + `
		FROM public.finance_tab WHERE building_id = $1 ORDER BY rowversion DESC`
	if err := h.db.Select(&items, q, buildingID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *FinanceHandler) ListByProject(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	items := []models.Finance{}
	const q = `SELECT ` + financeColumns + `
		FROM public.finance_tab WHERE project_id = $1 ORDER BY rowversion DESC`
	if err := h.db.Select(&items, q, projectID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *FinanceHandler) Create(c *gin.Context) {
	buildingID, err := strconv.Atoi(c.Param("building_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid building id"})
		return
	}

	financeTypeID, description, amount, ok := parseFinanceForm(c)
	if !ok {
		return
	}

	employeeID, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user context"})
		return
	}

	// A building-level finance still belongs to the building's project.
	var projectID int
	if err := h.db.Get(&projectID,
		`SELECT project_id FROM public.building_tab WHERE building_id = $1`, buildingID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	fileName, fileURL, err := saveFinanceFile(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
		return
	}

	var item models.Finance
	const q = `INSERT INTO public.finance_tab
		(finance_type_id, finance_description, finance_file_name, finance_file_url, employee_id, project_id, building_id, amount, rowversion)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		RETURNING ` + financeColumns
	if err := h.db.Get(&item, q,
		financeTypeID, description, fileName, fileURL,
		employeeID, projectID, buildingID, amount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *FinanceHandler) CreateForProject(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	financeTypeID, description, amount, ok := parseFinanceForm(c)
	if !ok {
		return
	}

	employeeID, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user context"})
		return
	}

	fileName, fileURL, err := saveFinanceFile(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
		return
	}

	// Project-direct finance — no building.
	var item models.Finance
	const q = `INSERT INTO public.finance_tab
		(finance_type_id, finance_description, finance_file_name, finance_file_url, employee_id, project_id, building_id, amount, rowversion)
		VALUES ($1, $2, $3, $4, $5, $6, NULL, $7, NOW())
		RETURNING ` + financeColumns
	if err := h.db.Get(&item, q,
		financeTypeID, description, fileName, fileURL,
		employeeID, projectID, amount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *FinanceHandler) Delete(c *gin.Context) {
	buildingID, err := strconv.Atoi(c.Param("building_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid building id"})
		return
	}
	financeID, err := strconv.Atoi(c.Param("finance_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid finance id"})
		return
	}

	res, err := h.db.Exec(
		`DELETE FROM public.finance_tab WHERE finance_id = $1 AND building_id = $2`,
		financeID, buildingID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "finance not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *FinanceHandler) DeleteByProject(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}
	financeID, err := strconv.Atoi(c.Param("finance_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid finance id"})
		return
	}

	res, err := h.db.Exec(
		`DELETE FROM public.finance_tab WHERE finance_id = $1 AND project_id = $2`,
		financeID, projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "finance not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// UpdateFile attaches or replaces the receipt file on an existing finance record.
func (h *FinanceHandler) UpdateFile(c *gin.Context) {
	financeID, err := strconv.Atoi(c.Param("finance_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid finance id"})
		return
	}

	// The file is required for this endpoint.
	if _, ferr := c.FormFile("file"); ferr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Файл не выбран"})
		return
	}

	fileName, fileURL, err := saveFinanceFile(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
		return
	}

	res, err := h.db.Exec(
		`UPDATE public.finance_tab SET finance_file_name = $1, finance_file_url = $2 WHERE finance_id = $3`,
		fileName, fileURL, financeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "finance not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"finance_file_name": fileName,
		"finance_file_url":  fileURL,
	})
}

func (h *FinanceHandler) BuildingBalance(c *gin.Context) {
	buildingID, err := strconv.Atoi(c.Param("building_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid building id"})
		return
	}

	var b balanceTotals
	const q = `SELECT
		COALESCE(SUM(CASE WHEN ft.is_invest = true THEN f.amount ELSE 0 END), 0) as income_total,
		COALESCE(SUM(CASE WHEN ft.is_invest = false THEN f.amount ELSE 0 END), 0) as expense_total
		FROM public.finance_tab f
		JOIN public.finance_type_tab ft ON ft.finance_type_id = f.finance_type_id
		WHERE f.building_id = $1`
	if err := h.db.Get(&b, q, buildingID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"income_total":  b.IncomeTotal,
		"expense_total": b.ExpenseTotal,
		"net_total":     b.IncomeTotal - b.ExpenseTotal,
	})
}

func (h *FinanceHandler) ProjectBalance(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	var b balanceTotals
	const q = `SELECT
		COALESCE(SUM(CASE WHEN ft.is_invest = true THEN f.amount ELSE 0 END), 0) as income_total,
		COALESCE(SUM(CASE WHEN ft.is_invest = false THEN f.amount ELSE 0 END), 0) as expense_total
		FROM public.finance_tab f
		JOIN public.finance_type_tab ft ON ft.finance_type_id = f.finance_type_id
		WHERE f.project_id = $1`
	if err := h.db.Get(&b, q, projectID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"income_total":  b.IncomeTotal,
		"expense_total": b.ExpenseTotal,
		"net_total":     b.IncomeTotal - b.ExpenseTotal,
	})
}
