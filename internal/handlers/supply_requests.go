package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ementau/ementau-backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

const (
	supplyRequestColumns = `request_id, building_id, status, employee_id, comment, rowversion, parent_request_id`
	supplyItemColumns    = `item_id, request_id, material_id, quantity, comment, price,
		received_status, received_quantity, supplier_id, unit_type_id`
)

type supplyItemInput struct {
	MaterialID int     `json:"material_id"`
	Quantity   float64 `json:"quantity"`
	Comment    string  `json:"comment"`
	UnitTypeID *int    `json:"unit_type_id"`
}

type createSupplyRequestInput struct {
	Comment string            `json:"comment"`
	Items   []supplyItemInput `json:"items"`
}

type setPriceInput struct {
	ItemID     int     `json:"item_id"`
	Price      float64 `json:"price"`
	SupplierID int     `json:"supplier_id"`
}

type receiveItemInput struct {
	ItemID           int     `json:"item_id"`
	ReceivedStatus   string  `json:"received_status"`
	ReceivedQuantity float64 `json:"received_quantity"`
}

// SupplyRequestHandler handles the supply request workflow.
type SupplyRequestHandler struct {
	db *sqlx.DB
}

// NewSupplyRequestHandler builds a SupplyRequestHandler.
func NewSupplyRequestHandler(db *sqlx.DB) *SupplyRequestHandler {
	return &SupplyRequestHandler{db: db}
}

// Register wires the supply request routes onto the given group.
func (h *SupplyRequestHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/supply-requests", h.ListAll)
	rg.GET("/buildings/:building_id/supply-requests", h.List)
	rg.POST("/buildings/:building_id/supply-requests", h.Create)
	rg.GET("/supply-requests/:request_id", h.Get)
	rg.PUT("/supply-requests/:request_id/items", h.UpdateItems)
	rg.PUT("/supply-requests/:request_id/prices", h.SetPrices)
	rg.PUT("/supply-requests/:request_id/receive", h.Receive)
	rg.POST("/supply-requests/:request_id/restock", h.CreateRestock)
	rg.DELETE("/supply-requests/:request_id", h.Delete)
}

func (h *SupplyRequestHandler) ListAll(c *gin.Context) {
	status := c.Query("status")
	buildingID, _ := strconv.Atoi(c.Query("building_id")) // 0 (or invalid) = no filter

	items := []models.SupplyRequest{}
	const q = `SELECT ` + supplyRequestColumns + `
		FROM public.supply_request_tab
		WHERE ($1 = '' OR status = $1) AND ($2 = 0 OR building_id = $2)
		ORDER BY rowversion DESC`
	if err := h.db.Select(&items, q, status, buildingID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *SupplyRequestHandler) List(c *gin.Context) {
	buildingID, err := strconv.Atoi(c.Param("building_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid building id"})
		return
	}

	items := []models.SupplyRequest{}
	const q = `SELECT ` + supplyRequestColumns + `
		FROM public.supply_request_tab WHERE building_id = $1 ORDER BY rowversion DESC`
	if err := h.db.Select(&items, q, buildingID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *SupplyRequestHandler) Get(c *gin.Context) {
	requestID, err := strconv.Atoi(c.Param("request_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request id"})
		return
	}

	var req models.SupplyRequest
	const reqQ = `SELECT ` + supplyRequestColumns + `
		FROM public.supply_request_tab WHERE request_id = $1`
	if err := h.db.Get(&req, reqQ, requestID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "supply request not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	items := []models.SupplyRequestItem{}
	const itemsQ = `SELECT ` + supplyItemColumns + `
		FROM public.supply_request_item_tab WHERE request_id = $1 ORDER BY item_id`
	if err := h.db.Select(&items, itemsQ, requestID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var hasRestock bool
	if err := h.db.Get(&hasRestock,
		`SELECT COUNT(*) > 0 FROM public.supply_request_tab WHERE parent_request_id = $1`,
		requestID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"request": req, "items": items, "has_restock": hasRestock})
}

func (h *SupplyRequestHandler) Create(c *gin.Context) {
	buildingID, err := strconv.Atoi(c.Param("building_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid building id"})
		return
	}

	var in createSupplyRequestInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	employeeID, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user context"})
		return
	}

	tx, err := h.db.Beginx()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	var requestID int
	const insertReq = `INSERT INTO public.supply_request_tab
		(building_id, status, employee_id, comment, rowversion)
		VALUES ($1, 'created', $2, $3, NOW())
		RETURNING request_id`
	if err := tx.Get(&requestID, insertReq, buildingID, employeeID, in.Comment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := insertSupplyItems(tx, requestID, in.Items); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"request_id": requestID})
}

func (h *SupplyRequestHandler) UpdateItems(c *gin.Context) {
	requestID, err := strconv.Atoi(c.Param("request_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request id"})
		return
	}

	var in createSupplyRequestInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	status, err := h.requestStatus(requestID)
	if err != nil {
		h.respondStatusErr(c, err)
		return
	}
	if status != "created" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "заявку можно изменять только в статусе 'created'"})
		return
	}

	tx, err := h.db.Beginx()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		`UPDATE public.supply_request_tab SET comment = $1 WHERE request_id = $2`,
		in.Comment, requestID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if _, err := tx.Exec(
		`DELETE FROM public.supply_request_item_tab WHERE request_id = $1`, requestID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := insertSupplyItems(tx, requestID, in.Items); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

func (h *SupplyRequestHandler) SetPrices(c *gin.Context) {
	requestID, err := strconv.Atoi(c.Param("request_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request id"})
		return
	}

	var in struct {
		Items []setPriceInput `json:"items"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	status, err := h.requestStatus(requestID)
	if err != nil {
		h.respondStatusErr(c, err)
		return
	}
	if status != "created" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "цены можно задать только в статусе 'created'"})
		return
	}

	// Every submitted item must belong to this request.
	existing, err := h.requestItemIDs(requestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	for _, item := range in.Items {
		if !existing[item.ItemID] {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Позиция #%d не принадлежит заявке", item.ItemID)})
			return
		}
		if item.SupplierID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Выберите поставщика для каждой позиции"})
			return
		}
	}

	tx, err := h.db.Beginx()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	for _, item := range in.Items {
		if _, err := tx.Exec(
			`UPDATE public.supply_request_item_tab SET price = $1, supplier_id = $2 WHERE item_id = $3 AND request_id = $4`,
			item.Price, item.SupplierID, item.ItemID, requestID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if _, err := tx.Exec(
		`UPDATE public.supply_request_tab SET status = 'sent' WHERE request_id = $1`, requestID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "sent"})
}

func (h *SupplyRequestHandler) Receive(c *gin.Context) {
	requestID, err := strconv.Atoi(c.Param("request_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request id"})
		return
	}

	var in struct {
		Items []receiveItemInput `json:"items"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req models.SupplyRequest
	const reqQ = `SELECT ` + supplyRequestColumns + `
		FROM public.supply_request_tab WHERE request_id = $1`
	if err := h.db.Get(&req, reqQ, requestID); err != nil {
		h.respondStatusErr(c, err)
		return
	}
	if req.Status != "sent" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "принять можно только заявку в статусе 'sent'"})
		return
	}

	// Every submitted item must belong to this request, and every item of the
	// request must be marked — otherwise the status flip would hide a no-op update.
	existing, err := h.requestItemIDs(requestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	submitted := make(map[int]bool, len(in.Items))
	for _, item := range in.Items {
		if !existing[item.ItemID] {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Позиция #%d не принадлежит заявке", item.ItemID)})
			return
		}
		submitted[item.ItemID] = true
	}
	if len(submitted) != len(existing) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Не все позиции размечены при приёмке"})
		return
	}

	tx, err := h.db.Beginx()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	for _, item := range in.Items {
		if _, err := tx.Exec(
			`UPDATE public.supply_request_item_tab
				SET received_status = $1, received_quantity = $2
				WHERE item_id = $3 AND request_id = $4`,
			item.ReceivedStatus, item.ReceivedQuantity, item.ItemID, requestID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	if _, err := tx.Exec(
		`UPDATE public.supply_request_tab SET status = 'received' WHERE request_id = $1`, requestID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "received"})
}

func (h *SupplyRequestHandler) CreateRestock(c *gin.Context) {
	sourceID, err := strconv.Atoi(c.Param("request_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request id"})
		return
	}

	employeeID, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user context"})
		return
	}

	var src models.SupplyRequest
	const srcQ = `SELECT ` + supplyRequestColumns + `
		FROM public.supply_request_tab WHERE request_id = $1`
	if err := h.db.Get(&src, srcQ, sourceID); err != nil {
		h.respondStatusErr(c, err)
		return
	}

	missing := []models.SupplyRequestItem{}
	const missingQ = `SELECT ` + supplyItemColumns + `
		FROM public.supply_request_item_tab
		WHERE request_id = $1 AND received_status IN ('shortage', 'not_delivered')
		ORDER BY item_id`
	if err := h.db.Select(&missing, missingQ, sourceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(missing) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Нет недостающих позиций"})
		return
	}

	// Reject a second restock for the same source request.
	var childCount int
	if err := h.db.Get(&childCount,
		`SELECT COUNT(*) FROM public.supply_request_tab WHERE parent_request_id = $1`,
		sourceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if childCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Дозаказ по этой заявке уже создан"})
		return
	}

	// Compute the missing quantity per item.
	newItems := make([]supplyItemInput, 0, len(missing))
	for _, it := range missing {
		qty := it.Quantity
		if it.ReceivedStatus != nil && *it.ReceivedStatus == "shortage" {
			received := 0.0
			if it.ReceivedQuantity != nil {
				received = *it.ReceivedQuantity
			}
			qty = it.Quantity - received
		}
		newItems = append(newItems, supplyItemInput{
			MaterialID: it.MaterialID,
			Quantity:   qty,
			Comment:    it.Comment,
			UnitTypeID: it.UnitTypeID,
		})
	}

	tx, err := h.db.Beginx()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	var newID int
	const insertReq = `INSERT INTO public.supply_request_tab
		(building_id, status, employee_id, comment, parent_request_id, rowversion)
		VALUES ($1, 'created', $2, $3, $4, NOW())
		RETURNING request_id`
	comment := fmt.Sprintf("Дозаказ по заявке #%d", sourceID)
	if err := tx.Get(&newID, insertReq, src.BuildingID, employeeID, comment, sourceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := insertSupplyItems(tx, newID, newItems); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"request_id": newID})
}

func (h *SupplyRequestHandler) Delete(c *gin.Context) {
	requestID, err := strconv.Atoi(c.Param("request_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request id"})
		return
	}

	status, err := h.requestStatus(requestID)
	if err != nil {
		h.respondStatusErr(c, err)
		return
	}
	if status != "created" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "удалить можно только заявку в статусе 'created'"})
		return
	}

	// Items cascade via FK.
	if _, err := h.db.Exec(`DELETE FROM public.supply_request_tab WHERE request_id = $1`, requestID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// requestItemIDs returns the set of item_ids that belong to the given request.
func (h *SupplyRequestHandler) requestItemIDs(requestID int) (map[int]bool, error) {
	ids := []int{}
	if err := h.db.Select(&ids,
		`SELECT item_id FROM public.supply_request_item_tab WHERE request_id = $1`, requestID); err != nil {
		return nil, err
	}
	set := make(map[int]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set, nil
}

// requestStatus returns the current status of a request, or sql.ErrNoRows if absent.
func (h *SupplyRequestHandler) requestStatus(requestID int) (string, error) {
	var status string
	err := h.db.Get(&status, `SELECT status FROM public.supply_request_tab WHERE request_id = $1`, requestID)
	return status, err
}

// respondStatusErr maps a lookup error to 404 (missing) or 500.
func (h *SupplyRequestHandler) respondStatusErr(c *gin.Context, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "supply request not found"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

// insertSupplyItems inserts the given items for a request within a transaction.
func insertSupplyItems(tx *sqlx.Tx, requestID int, items []supplyItemInput) error {
	const q = `INSERT INTO public.supply_request_item_tab
		(request_id, material_id, quantity, comment, unit_type_id)
		VALUES ($1, $2, $3, $4, $5)`
	for _, item := range items {
		if _, err := tx.Exec(q, requestID, item.MaterialID, item.Quantity, item.Comment, item.UnitTypeID); err != nil {
			return err
		}
	}
	return nil
}
