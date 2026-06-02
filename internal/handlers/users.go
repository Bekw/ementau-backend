package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/ementau/ementau-backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

type userInput struct {
	FIO        string `json:"fio"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	UserRoleID int    `json:"user_role_id"`
	PhoneNum   string `json:"phone_num"`
}

// UserHandler handles CRUD for admin.user_tab.
type UserHandler struct {
	db *sqlx.DB
}

// NewUserHandler builds a UserHandler.
func NewUserHandler(db *sqlx.DB) *UserHandler {
	return &UserHandler{db: db}
}

// Register wires the user routes onto the given group.
func (h *UserHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/users", h.List)
	rg.GET("/users/:id", h.Get)
	rg.POST("/users", h.Create)
	rg.PUT("/users/:id", h.Update)
	rg.DELETE("/users/:id", h.Delete)
}

func (h *UserHandler) List(c *gin.Context) {
	users := []models.User{}
	const q = `SELECT user_id, fio, email, user_role_id, phone_num, rowversion
		FROM admin.user_tab ORDER BY user_id`
	if err := h.db.Select(&users, q); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, users)
}

func (h *UserHandler) Get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var u models.User
	const q = `SELECT user_id, fio, email, user_role_id, phone_num, rowversion
		FROM admin.user_tab WHERE user_id = $1`
	if err := h.db.Get(&u, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, u)
}

func (h *UserHandler) Create(c *gin.Context) {
	var in userInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if in.FIO == "" || in.Email == "" || in.Password == "" || in.UserRoleID == 0 || in.PhoneNum == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fio, email, password, user_role_id and phone_num are required"})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	var u models.User
	const q = `INSERT INTO admin.user_tab (fio, email, password, user_role_id, phone_num)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING user_id, fio, email, password, user_role_id, phone_num, rowversion`
	if err := h.db.Get(&u, q, in.FIO, in.Email, string(hashed), in.UserRoleID, in.PhoneNum); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, u)
}

func (h *UserHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var in userInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if in.FIO == "" || in.Email == "" || in.UserRoleID == 0 || in.PhoneNum == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fio, email, user_role_id and phone_num are required"})
		return
	}

	var u models.User
	var q string
	args := []interface{}{in.FIO, in.Email, in.UserRoleID, in.PhoneNum}

	if in.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
			return
		}
		q = `UPDATE admin.user_tab
			SET fio = $1, email = $2, user_role_id = $3, phone_num = $4, password = $5
			WHERE user_id = $6
			RETURNING user_id, fio, email, password, user_role_id, phone_num, rowversion`
		args = append(args, string(hashed), id)
	} else {
		q = `UPDATE admin.user_tab
			SET fio = $1, email = $2, user_role_id = $3, phone_num = $4
			WHERE user_id = $5
			RETURNING user_id, fio, email, password, user_role_id, phone_num, rowversion`
		args = append(args, id)
	}

	if err := h.db.Get(&u, q, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, u)
}

func (h *UserHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	res, err := h.db.Exec(`DELETE FROM admin.user_tab WHERE user_id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
