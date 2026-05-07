package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"kr4/apperrors"
	"kr4/models"
)

type ProductHandler struct{ db *gorm.DB }

func NewProductHandler(db *gorm.DB) *ProductHandler { return &ProductHandler{db: db} }

// POST /products
func (h *ProductHandler) Create(c *gin.Context) {
	var input models.ProductInput
	if err := c.ShouldBindJSON(&input); err != nil {
		apperrors.HandleValidationError(c, err)
		return
	}
	p := models.Product{
		Title:       input.Title,
		Price:       input.Price,
		Count:       input.Count,
		Description: input.Description,
	}
	if err := h.db.Create(&p).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apperrors.ErrorResponse{
			Code: 500, Type: "DBError", Message: err.Error(),
		})
		return
	}
	c.JSON(http.StatusCreated, p)
}

// GET /products
func (h *ProductHandler) List(c *gin.Context) {
	var products []models.Product
	if err := h.db.Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apperrors.ErrorResponse{
			Code: 500, Type: "DBError", Message: err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, products)
}

// GET /products/:id
func (h *ProductHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apperrors.ErrorResponse{
			Code: 400, Type: "BadRequest", Message: "Неверный ID",
		})
		return
	}
	var p models.Product
	if err := h.db.First(&p, id).Error; err != nil {
		// Task 10.1: бросаем кастомное исключение CustomErrorB
		panic(&apperrors.CustomErrorB{Message: "Продукт не найден"})
	}
	c.JSON(http.StatusOK, p)
}
