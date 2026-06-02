package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type dashboardTopProject struct {
	ProjectID    int     `db:"project_id"    json:"project_id"`
	ProjectName  string  `db:"project_name"  json:"project_name"`
	IncomeTotal  float64 `db:"income_total"  json:"income_total"`
	ExpenseTotal float64 `db:"expense_total" json:"expense_total"`
}

type dashboardMonthly struct {
	Month   string  `db:"month"   json:"month"`
	Income  float64 `db:"income"  json:"income"`
	Expense float64 `db:"expense" json:"expense"`
}

// DashboardHandler serves aggregated stats for the dashboard.
type DashboardHandler struct {
	db *sqlx.DB
}

// NewDashboardHandler builds a DashboardHandler.
func NewDashboardHandler(db *sqlx.DB) *DashboardHandler {
	return &DashboardHandler{db: db}
}

// Register wires the dashboard route onto the given group.
func (h *DashboardHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/dashboard", h.Get)
}

func (h *DashboardHandler) Get(c *gin.Context) {
	var projectsCount int
	if err := h.db.Get(&projectsCount, `SELECT COUNT(*) FROM public.project_tab`); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var buildingsCount int
	if err := h.db.Get(&buildingsCount, `SELECT COUNT(*) FROM public.building_tab`); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var totals balanceTotals
	const totalsQ = `SELECT
		COALESCE(SUM(CASE WHEN ft.is_invest = true THEN f.amount ELSE 0 END), 0) as income_total,
		COALESCE(SUM(CASE WHEN ft.is_invest = false THEN f.amount ELSE 0 END), 0) as expense_total
		FROM public.finance_tab f
		JOIN public.finance_type_tab ft ON ft.finance_type_id = f.finance_type_id`
	if err := h.db.Get(&totals, totalsQ); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	topProjects := []dashboardTopProject{}
	// Finances link to projects directly now (building optional), so join on project_id.
	const topQ = `SELECT
		p.project_id,
		p.project_name,
		COALESCE(SUM(CASE WHEN ft.is_invest = true THEN f.amount ELSE 0 END), 0) as income_total,
		COALESCE(SUM(CASE WHEN ft.is_invest = false THEN f.amount ELSE 0 END), 0) as expense_total
		FROM public.project_tab p
		LEFT JOIN public.finance_tab f ON f.project_id = p.project_id
		LEFT JOIN public.finance_type_tab ft ON ft.finance_type_id = f.finance_type_id
		GROUP BY p.project_id, p.project_name
		ORDER BY income_total DESC
		LIMIT 5`
	if err := h.db.Select(&topProjects, topQ); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	monthly := []dashboardMonthly{}
	const monthlyQ = `SELECT
		TO_CHAR(f.rowversion, 'YYYY-MM') as month,
		COALESCE(SUM(CASE WHEN ft.is_invest = true THEN f.amount ELSE 0 END), 0) as income,
		COALESCE(SUM(CASE WHEN ft.is_invest = false THEN f.amount ELSE 0 END), 0) as expense
		FROM public.finance_tab f
		JOIN public.finance_type_tab ft ON ft.finance_type_id = f.finance_type_id
		WHERE f.rowversion >= NOW() - INTERVAL '6 months'
		GROUP BY TO_CHAR(f.rowversion, 'YYYY-MM')
		ORDER BY month ASC`
	if err := h.db.Select(&monthly, monthlyQ); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"projects_count":  projectsCount,
		"buildings_count": buildingsCount,
		"income_total":    totals.IncomeTotal,
		"expense_total":   totals.ExpenseTotal,
		"net_total":       totals.IncomeTotal - totals.ExpenseTotal,
		"top_projects":    topProjects,
		"monthly_stats":   monthly,
	})
}
