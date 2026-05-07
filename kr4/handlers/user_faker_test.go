// Task 11.2: асинхронные тесты с Faker.
//
// В Go нет pytest-asyncio, но есть t.Parallel() для конкурентного запуска.
// Аналог httpx.AsyncClient + ASGITransport — net/http/httptest.Server + http.Client,
// который позволяет обращаться к приложению без реального запуска сервера.
// Для генерации данных используется github.com/brianvoe/gofakeit/v6 (аналог Python Faker).
package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"kr4/handlers"
	"kr4/models"
)

// validUserPayload генерирует валидные данные пользователя через Faker.
func validUserPayload() map[string]any {
	return map[string]any{
		"username": gofakeit.Username(),
		"age":      gofakeit.IntRange(19, 80),
		"email":    gofakeit.Email(),
		// пароль: строчные + заглавные + цифры, длина 10 (между 8 и 16)
		"password": gofakeit.Password(true, true, true, false, false, 10),
		"phone":    gofakeit.Phone(),
	}
}

// TestFaker_CreateUser — создание пользователя с данными от Faker.
func TestFaker_CreateUser(t *testing.T) {
	t.Parallel()

	store := handlers.NewUserStore()
	r := handlers.NewRouterWithStore(nil, store)
	defer store.Reset()

	body, _ := json.Marshal(validUserPayload())
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("ожидали 201, получили %d: %s", w.Code, w.Body)
	}

	var resp models.UserOut
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.ID == 0 {
		t.Error("ID должен быть ненулевым")
	}
}

// TestFaker_GetExistingUser — получение существующего пользователя (200).
func TestFaker_GetExistingUser(t *testing.T) {
	t.Parallel()

	store := handlers.NewUserStore()
	r := handlers.NewRouterWithStore(nil, store)
	defer store.Reset()

	postJSON(r, "/users", validUserPayload())

	w := doRequest(r, http.MethodGet, "/users/1")
	if w.Code != http.StatusOK {
		t.Errorf("ожидали 200, получили %d", w.Code)
	}
}

// TestFaker_GetNonExistentUser — запрос несуществующего пользователя (404).
func TestFaker_GetNonExistentUser(t *testing.T) {
	t.Parallel()

	store := handlers.NewUserStore()
	r := handlers.NewRouterWithStore(nil, store)
	defer store.Reset()

	w := doRequest(r, http.MethodGet, "/users/9999")
	if w.Code != http.StatusNotFound {
		t.Errorf("ожидали 404, получили %d", w.Code)
	}
}

// TestFaker_DeleteUser — удаление пользователя (204).
func TestFaker_DeleteUser(t *testing.T) {
	t.Parallel()

	store := handlers.NewUserStore()
	r := handlers.NewRouterWithStore(nil, store)
	defer store.Reset()

	postJSON(r, "/users", validUserPayload())

	w := doRequest(r, http.MethodDelete, "/users/1")
	if w.Code != http.StatusNoContent {
		t.Fatalf("ожидали 204, получили %d", w.Code)
	}
}

// TestFaker_DeleteThenGetReturns404 — после удаления пользователь недоступен (404).
func TestFaker_DeleteThenGetReturns404(t *testing.T) {
	t.Parallel()

	store := handlers.NewUserStore()
	r := handlers.NewRouterWithStore(nil, store)
	defer store.Reset()

	postJSON(r, "/users", validUserPayload())
	doRequest(r, http.MethodDelete, "/users/1")

	w := doRequest(r, http.MethodGet, "/users/1")
	if w.Code != http.StatusNotFound {
		t.Errorf("ожидали 404 после удаления, получили %d", w.Code)
	}
}

// TestFaker_DoubleDelete — повторное удаление возвращает 404.
func TestFaker_DoubleDelete(t *testing.T) {
	t.Parallel()

	store := handlers.NewUserStore()
	r := handlers.NewRouterWithStore(nil, store)
	defer store.Reset()

	postJSON(r, "/users", validUserPayload())
	doRequest(r, http.MethodDelete, "/users/1")

	w := doRequest(r, http.MethodDelete, "/users/1")
	if w.Code != http.StatusNotFound {
		t.Errorf("ожидали 404, получили %d", w.Code)
	}
}

// TestFaker_ValidationErrors — граничные значения, вызывающие ошибки валидации.
func TestFaker_ValidationErrors(t *testing.T) {
	t.Parallel()

	store := handlers.NewUserStore()
	r := handlers.NewRouterWithStore(nil, store)
	defer store.Reset()

	cases := []struct {
		name    string
		payload map[string]any
	}{
		{"age=18 (not > 18)", map[string]any{"username": gofakeit.Username(), "age": 18, "email": gofakeit.Email(), "password": "Valid1234"}},
		{"age=0", map[string]any{"username": gofakeit.Username(), "age": 0, "email": gofakeit.Email(), "password": "Valid1234"}},
		{"bad email", map[string]any{"username": gofakeit.Username(), "age": 25, "email": "not-email", "password": "Valid1234"}},
		{"short password", map[string]any{"username": gofakeit.Username(), "age": 25, "email": gofakeit.Email(), "password": "abc"}},
		{"long password", map[string]any{"username": gofakeit.Username(), "age": 25, "email": gofakeit.Email(), "password": "ThisIsLongPass123"}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := postJSON(r, "/users", tc.payload)
			if w.Code != http.StatusUnprocessableEntity {
				t.Errorf("[%s] ожидали 422, получили %d: %s", tc.name, w.Code, w.Body)
			}
		})
	}
}

// TestFaker_ConcurrentCreation — аналог «асинхронного» тестирования:
// несколько горутин одновременно создают пользователей, проверяем отсутствие гонок.
func TestFaker_ConcurrentCreation(t *testing.T) {
	store := handlers.NewUserStore()
	r := handlers.NewRouterWithStore(nil, store)
	defer store.Reset()

	// Аналог httpx.AsyncClient — реальный HTTP-сервер через httptest.Server
	srv := httptest.NewServer(r)
	defer srv.Close()

	const n = 10
	var wg sync.WaitGroup
	wg.Add(n)
	errs := make(chan error, n)

	for i := range n {
		i := i
		go func() {
			defer wg.Done()
			payload, _ := json.Marshal(validUserPayload())
			resp, err := http.Post(
				fmt.Sprintf("%s/users", srv.URL),
				"application/json",
				bytes.NewReader(payload),
			)
			if err != nil {
				errs <- fmt.Errorf("горутина %d: %w", i, err)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusCreated {
				errs <- fmt.Errorf("горутина %d: ожидали 201, получили %d", i, resp.StatusCode)
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}
}

// TestFaker_WithContext — демонстрация использования context.Context,
// аналог отмены запроса в httpx.AsyncClient.
func TestFaker_WithContext(t *testing.T) {
	t.Parallel()

	store := handlers.NewUserStore()
	r := handlers.NewRouterWithStore(nil, store)
	defer store.Reset()
	srv := httptest.NewServer(r)
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	payload, _ := json.Marshal(validUserPayload())
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/users", srv.URL), bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("ожидали 201, получили %d", resp.StatusCode)
	}
}
