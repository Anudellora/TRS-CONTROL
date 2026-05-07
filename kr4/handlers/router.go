package handlers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"kr4/apperrors"
)

// NewRouter создаёт gin-роутер для production-запуска (с подключённой БД).
func NewRouter(db *gorm.DB) *gin.Engine {
	store := NewUserStore()
	return newEngine(db, store)
}

// NewRouterWithStore создаёт роутер с переданным хранилищем — используется в тестах.
func NewRouterWithStore(db *gorm.DB, store *UserStore) *gin.Engine {
	return newEngine(db, store)
}

func newEngine(db *gorm.DB, store *UserStore) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger())
	// RecoverMiddleware зарегистрирован ПОСЛЕ gin.Recovery (LIFO → его defer срабатывает первым)
	r.Use(apperrors.RecoverMiddleware())

	// ── Task 9.1: продукты (CRUD через GORM + миграции) ──────────────────────
	if db != nil {
		ph := NewProductHandler(db)
		r.POST("/products", ph.Create)
		r.GET("/products", ph.List)
		r.GET("/products/:id", ph.GetByID)
	}

	// ── Tasks 10.2 / 11.1 / 11.2: пользователи (in-memory) ──────────────────
	uh := NewUserHandler(store)
	r.POST("/users", uh.Create)
	r.GET("/users/:id", uh.GetByID)
	r.DELETE("/users/:id", uh.Delete)

	// ── Task 10.1: демо-эндпоинты кастомных исключений ───────────────────────
	// GET /demo/error-a?condition=  → CustomErrorA (400)
	r.GET("/demo/error-a", func(c *gin.Context) {
		if c.Query("condition") == "" {
			panic(&apperrors.CustomErrorA{Message: "Обязательное условие не выполнено"})
		}
		c.JSON(200, gin.H{"message": "Условие выполнено"})
	})

	// GET /demo/error-b?id=  → CustomErrorB (404)
	r.GET("/demo/error-b", func(c *gin.Context) {
		if c.Query("id") == "" {
			panic(&apperrors.CustomErrorB{Message: "Ресурс с указанным ID не найден"})
		}
		c.JSON(200, gin.H{"message": "Ресурс найден", "id": c.Query("id")})
	})

	return r
}
