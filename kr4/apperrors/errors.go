// Package apperrors реализует Task 10.1 и Task 10.2:
//   - кастомные классы исключений (CustomErrorA, CustomErrorB)
//   - middleware-обработчик исключений (аналог @app.exception_handler)
//   - модели ответов об ошибках (ErrorResponse, ValidationErrorResp)
//   - форматирование ошибок валидации
package apperrors

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ── Task 10.1: кастомные классы исключений ────────────────────────────────────

// CustomErrorA — исключение «условие не выполнено», HTTP 400
type CustomErrorA struct{ Message string }

func (e *CustomErrorA) Error() string   { return e.Message }
func (e *CustomErrorA) StatusCode() int { return http.StatusBadRequest }

// CustomErrorB — исключение «ресурс не найден», HTTP 404
type CustomErrorB struct{ Message string }

func (e *CustomErrorB) Error() string   { return e.Message }
func (e *CustomErrorB) StatusCode() int { return http.StatusNotFound }

// ── Модели ответов ─────────────────────────────────────────────────────────────

// ErrorResponse — стандартная модель ошибки (Task 10.1)
type ErrorResponse struct {
	Code    int    `json:"code"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

// ValidationFieldError — ошибка по одному полю
type ValidationFieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrorResp — ответ при ошибках валидации (Task 10.2)
type ValidationErrorResp struct {
	Code   int                    `json:"code"`
	Type   string                 `json:"type"`
	Errors []ValidationFieldError `json:"errors"`
}

// ── Task 10.1: middleware-обработчик исключений ───────────────────────────────
// Аналог @app.exception_handler в FastAPI.
// Регистрируется ПОСЛЕ gin.Recovery(), чтобы его defer выполнился первым (LIFO).
func RecoverMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			r := recover()
			if r == nil {
				return
			}
			switch err := r.(type) {
			case *CustomErrorA:
				c.AbortWithStatusJSON(err.StatusCode(), ErrorResponse{
					Code:    err.StatusCode(),
					Type:    "CustomErrorA",
					Message: err.Message,
				})
			case *CustomErrorB:
				c.AbortWithStatusJSON(err.StatusCode(), ErrorResponse{
					Code:    err.StatusCode(),
					Type:    "CustomErrorB",
					Message: err.Message,
				})
			default:
				c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{
					Code:    http.StatusInternalServerError,
					Type:    "InternalServerError",
					Message: fmt.Sprintf("%v", r),
				})
			}
		}()
		c.Next()
	}
}

// ── Task 10.2: обработка ошибок валидации ─────────────────────────────────────
// Аналог @app.exception_handler(RequestValidationError).
func HandleValidationError(c *gin.Context, err error) {
	var valErrs validator.ValidationErrors
	if errors.As(err, &valErrs) {
		fields := make([]ValidationFieldError, 0, len(valErrs))
		for _, fe := range valErrs {
			fields = append(fields, ValidationFieldError{
				Field:   fe.Field(),
				Message: validationMessage(fe),
			})
		}
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, ValidationErrorResp{
			Code:   http.StatusUnprocessableEntity,
			Type:   "ValidationError",
			Errors: fields,
		})
		return
	}
	c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{
		Code:    http.StatusBadRequest,
		Type:    "BadRequest",
		Message: err.Error(),
	})
}

func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "Поле обязательно для заполнения"
	case "email":
		return "Должен быть действительный адрес e-mail"
	case "gt":
		return fmt.Sprintf("Значение должно быть больше %s", fe.Param())
	case "gte":
		return fmt.Sprintf("Значение должно быть не менее %s", fe.Param())
	case "min":
		return fmt.Sprintf("Минимальная длина: %s символов", fe.Param())
	case "max":
		return fmt.Sprintf("Максимальная длина: %s символов", fe.Param())
	default:
		return fmt.Sprintf("Ошибка валидации тега '%s'", fe.Tag())
	}
}
