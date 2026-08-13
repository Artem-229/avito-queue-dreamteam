package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// setupTestRouter поднимает минимальный роутер с единственным защищённым
// маршрутом — так каждый тест ниже проверяет ровно поведение middleware,
// не завися от реальных хендлеров каталога.
func setupTestRouter(secretKey string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AdminAuthMiddleware(secretKey))
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

// TestValidateAdminKey проверяет саму функцию сравнения в изоляции от HTTP —
// это единственное место, где реально сравнивается ключ (и middleware,
// и AdminLogin вызывают именно её), поэтому здесь разобраны все граничные
// случаи: совпадение, несовпадение, и оба варианта пустой строки отдельно
// (пустой ввод юзера vs админка вообще не настроена на сервере).
func TestValidateAdminKey(t *testing.T) {
	cases := []struct {
		name     string
		provided string
		secret   string
		want     bool
	}{
		{"matching keys", "abc123", "abc123", true},
		{"mismatched keys", "wrong", "abc123", false},
		{"empty provided", "", "abc123", false},
		{"empty secret (admin not configured)", "abc123", "", false},
		{"both empty", "", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ValidateAdminKey(tc.provided, tc.secret)
			if got != tc.want {
				t.Errorf("ValidateAdminKey(%q, %q) = %v, want %v", tc.provided, tc.secret, got, tc.want)
			}
		})
	}
}

// Happy path: верный ключ в заголовке → middleware пропускает запрос
// дальше по цепочке (c.Next()), хендлер реально вызывается и отвечает 200.
func TestAdminAuthMiddleware_ValidKey(t *testing.T) {
	r := setupTestRouter("supersecret")

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("X-Admin-Key", "supersecret")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// Ключ передан, но не совпадает с настроенным на сервере → 403, а не 401:
// заголовок технически присутствует и распознан, просто не даёт прав —
// это осознанный отказ, а не "не авторизован вовсе".
func TestAdminAuthMiddleware_WrongKey(t *testing.T) {
	r := setupTestRouter("supersecret")

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("X-Admin-Key", "wrongkey")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

// Заголовок X-Admin-Key вообще отсутствует в запросе (не пустая строка,
// а именно не передан) — тот самый случай прямого обхода по ссылке без
// всякой попытки авторизоваться, который мы разбирали ещё на примере CHK-02.
func TestAdminAuthMiddleware_NoKey(t *testing.T) {
	r := setupTestRouter("supersecret")

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

// Сервер поднят без ADMIN_SECRET_KEY в окружении (secretKey == "") — даже
// если юзер пришлёт правильно выглядящий заголовок, middleware обязан
// вернуть явный 503 "функция не настроена", а не 403 "неверный ключ": это
// разные ситуации для того, кто читает логи/дебажит окружение, и разная
// диагностика ускоряет починку на проде.
func TestAdminAuthMiddleware_NotConfigured(t *testing.T) {
	r := setupTestRouter("") // сервер без настроенного ADMIN_SECRET_KEY

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("X-Admin-Key", "anything")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}
