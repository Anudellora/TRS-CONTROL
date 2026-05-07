// Task 11.1: модульные тесты трёх эндпоинтов с помощью net/http/httptest.
// Аналог pytest + TestClient из FastAPI.
package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"kr4/handlers"
	"kr4/models"
)

func init() { gin.SetMode(gin.TestMode) }

// newTestRouter создаёт изолированный роутер с чистым хранилищем.
func newTestRouter() (*gin.Engine, *handlers.UserStore) {
	store := handlers.NewUserStore()
	return handlers.NewRouterWithStore(nil, store), store
}

func postJSON(r *gin.Engine, path string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func doRequest(r *gin.Engine, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ── POST /users ───────────────────────────────────────────────────────────────

func TestCreateUser_Success(t *testing.T) {
	r, store := newTestRouter()
	defer store.Reset()

	w := postJSON(r, "/users", map[string]any{
		"username": "alice",
		"age":      25,
		"email":    "alice@example.com",
		"password": "Secret123",
	})

	if w.Code != http.StatusCreated {
		t.Fatalf("ожидали 201, получили %d: %s", w.Code, w.Body)
	}

	var resp models.UserOut
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("не удалось декодировать ответ:", err)
	}
	if resp.ID == 0 {
		t.Error("ID должен быть ненулевым")
	}
	if resp.Username != "alice" {
		t.Errorf("username: ожидали 'alice', получили '%s'", resp.Username)
	}
	if strings.Contains(w.Body.String(), "password") {
		t.Error("пароль не должен присутствовать в ответе")
	}
}

func TestCreateUser_InvalidEmail(t *testing.T) {
	r, store := newTestRouter()
	defer store.Reset()

	w := postJSON(r, "/users", map[string]any{
		"username": "bob",
		"age":      22,
		"email":    "not-an-email",
		"password": "Secret123",
	})

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("ожидали 422, получили %d", w.Code)
	}
}

func TestCreateUser_AgeTooLow(t *testing.T) {
	r, store := newTestRouter()
	defer store.Reset()

	w := postJSON(r, "/users", map[string]any{
		"username": "young",
		"age":      17,
		"email":    "young@example.com",
		"password": "Secret123",
	})

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("ожидали 422 (age ≤ 18), получили %d", w.Code)
	}
}

func TestCreateUser_PasswordTooShort(t *testing.T) {
	r, store := newTestRouter()
	defer store.Reset()

	w := postJSON(r, "/users", map[string]any{
		"username": "charlie",
		"age":      30,
		"email":    "c@example.com",
		"password": "abc",
	})

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("ожидали 422 (пароль < 8 символов), получили %d", w.Code)
	}
}

// ── GET /users/:id ────────────────────────────────────────────────────────────

func TestGetUser_Success(t *testing.T) {
	r, store := newTestRouter()
	defer store.Reset()

	postJSON(r, "/users", map[string]any{
		"username": "dave", "age": 28, "email": "dave@example.com", "password": "Passw0rd1",
	})

	w := doRequest(r, http.MethodGet, "/users/1")
	if w.Code != http.StatusOK {
		t.Fatalf("ожидали 200, получили %d", w.Code)
	}

	var resp models.UserOut
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Username != "dave" {
		t.Errorf("username: ожидали 'dave', получили '%s'", resp.Username)
	}
}

func TestGetUser_NotFound(t *testing.T) {
	r, store := newTestRouter()
	defer store.Reset()

	w := doRequest(r, http.MethodGet, "/users/999")
	if w.Code != http.StatusNotFound {
		t.Errorf("ожидали 404, получили %d", w.Code)
	}
}

// ── DELETE /users/:id ─────────────────────────────────────────────────────────

func TestDeleteUser_Success(t *testing.T) {
	r, store := newTestRouter()
	defer store.Reset()

	postJSON(r, "/users", map[string]any{
		"username": "eve", "age": 35, "email": "eve@example.com", "password": "Passw0rd1",
	})

	w := doRequest(r, http.MethodDelete, "/users/1")
	if w.Code != http.StatusNoContent {
		t.Fatalf("ожидали 204, получили %d", w.Code)
	}
}

func TestDeleteUser_NotFound(t *testing.T) {
	r, store := newTestRouter()
	defer store.Reset()

	w := doRequest(r, http.MethodDelete, "/users/42")
	if w.Code != http.StatusNotFound {
		t.Errorf("ожидали 404, получили %d", w.Code)
	}
}

func TestDeleteUser_DoubleDelete(t *testing.T) {
	r, store := newTestRouter()
	defer store.Reset()

	postJSON(r, "/users", map[string]any{
		"username": "frank", "age": 40, "email": "frank@example.com", "password": "Passw0rd1",
	})
	doRequest(r, http.MethodDelete, "/users/1")

	// Повторное удаление → 404
	w := doRequest(r, http.MethodDelete, "/users/1")
	if w.Code != http.StatusNotFound {
		t.Errorf("ожидали 404 при повторном удалении, получили %d", w.Code)
	}
}

// ── Task 10.1: demo error endpoints ──────────────────────────────────────────

func TestDemoErrorA(t *testing.T) {
	r, _ := newTestRouter()
	w := doRequest(r, http.MethodGet, "/demo/error-a")
	if w.Code != http.StatusBadRequest {
		t.Errorf("CustomErrorA: ожидали 400, получили %d", w.Code)
	}
}

func TestDemoErrorB(t *testing.T) {
	r, _ := newTestRouter()
	w := doRequest(r, http.MethodGet, "/demo/error-b")
	if w.Code != http.StatusNotFound {
		t.Errorf("CustomErrorB: ожидали 404, получили %d", w.Code)
	}
}
