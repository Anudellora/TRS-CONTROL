package handlers

import (
	"net/http"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	"kr4/apperrors"
	"kr4/models"
)

// UserStore — in-memory хранилище пользователей (Tasks 10.2, 11.1, 11.2).
// Аналог dict[int, dict] из примера на Python.
type UserStore struct {
	mu     sync.Mutex
	data   map[int]models.UserOut
	nextID int
}

func NewUserStore() *UserStore {
	return &UserStore{data: make(map[int]models.UserOut), nextID: 1}
}

// Reset очищает хранилище — используется для изоляции состояния в тестах.
func (s *UserStore) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = make(map[int]models.UserOut)
	s.nextID = 1
}

type UserHandler struct{ store *UserStore }

func NewUserHandler(store *UserStore) *UserHandler { return &UserHandler{store: store} }

// POST /users  — Task 10.2: валидация входных данных
func (h *UserHandler) Create(c *gin.Context) {
	var input models.UserIn
	if err := c.ShouldBindJSON(&input); err != nil {
		// Task 10.2: кастомный обработчик ошибок валидации
		apperrors.HandleValidationError(c, err)
		return
	}

	phone := input.Phone
	if phone == "" {
		phone = "Unknown"
	}

	h.store.mu.Lock()
	defer h.store.mu.Unlock()

	id := h.store.nextID
	h.store.nextID++

	user := models.UserOut{
		ID:       id,
		Username: input.Username,
		Age:      input.Age,
		Email:    input.Email,
		Phone:    phone,
	}
	h.store.data[id] = user
	c.JSON(http.StatusCreated, user)
}

// GET /users/:id
func (h *UserHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apperrors.ErrorResponse{Code: 400, Type: "BadRequest", Message: "Неверный ID"})
		return
	}

	h.store.mu.Lock()
	defer h.store.mu.Unlock()

	user, ok := h.store.data[id]
	if !ok {
		// Task 10.1: кастомное исключение «ресурс не найден»
		panic(&apperrors.CustomErrorB{Message: "Пользователь не найден"})
	}
	c.JSON(http.StatusOK, user)
}

// DELETE /users/:id
func (h *UserHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apperrors.ErrorResponse{Code: 400, Type: "BadRequest", Message: "Неверный ID"})
		return
	}

	h.store.mu.Lock()
	defer h.store.mu.Unlock()

	if _, ok := h.store.data[id]; !ok {
		panic(&apperrors.CustomErrorB{Message: "Пользователь не найден"})
	}
	delete(h.store.data, id)
	c.Status(http.StatusNoContent)
}
