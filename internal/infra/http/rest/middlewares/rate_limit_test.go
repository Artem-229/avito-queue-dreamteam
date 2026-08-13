package middlewares

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// setupRateLimitRouter поднимает минимальный роутер с единственным маршрутом
// под лимитером — так каждый тест проверяет ровно поведение
// RateLimitMiddleware, не завися от реальных хендлеров логина/очереди.
// Маршрут с :id — чтобы тестировать и KeyByIP, и KeyByIPAndItem одним и тем
// же роутером.
func setupRateLimitRouter(maxAttempts int, window time.Duration, keyFunc func(c *gin.Context) string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimitMiddleware(maxAttempts, window, keyFunc))
	r.POST("/items/:id/action", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

// doRequest шлёт один POST от заданного IP на заданный item_id и возвращает
// код ответа. IP передаётся через RemoteAddr — именно оттуда его достаёт
// c.ClientIP() внутри middleware, это не заголовок, который клиент мог бы
// подделать.
func doRequest(r *gin.Engine, ip, itemID string) int {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/items/"+itemID+"/action", nil)
	req.RemoteAddr = ip + ":12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

// Happy path: ровно maxAttempts запросов подряд от одного IP — ни один не
// должен быть заблокирован. Проверяет граничное значение изнутри лимита
// (5-й запрос при лимите 5), а не только "заведомо мало запросов".
// Используем KeyByIP — тут проверяется сам лимитер, а не конкретный keyFunc.
func TestRateLimitMiddleware_WithinLimit(t *testing.T) {
	r := setupRateLimitRouter(5, time.Minute, KeyByIP)

	for i := 0; i < 5; i++ {
		code := doRequest(r, "1.2.3.4", "item-1")
		if code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, code)
		}
	}
}

// Ровно на maxAttempts+1 запрос в том же окне — 429. Это и есть основная
// защита: без неё бот мог бы слать неограниченный поток запросов подряд
// (подбор admin-ключа или спам входа в очередь, в зависимости от того, на
// каком маршруте применён middleware).
func TestRateLimitMiddleware_ExceedsLimit(t *testing.T) {
	r := setupRateLimitRouter(3, time.Minute, KeyByIP)

	for i := 0; i < 3; i++ {
		doRequest(r, "5.6.7.8", "item-1")
	}

	code := doRequest(r, "5.6.7.8", "item-1")
	if code != http.StatusTooManyRequests {
		t.Errorf("expected 429 on request exceeding limit, got %d", code)
	}
}

// Разные IP не должны делить один и тот же счётчик — лимит per-IP, а не
// глобальный на весь сервер. Без этой изоляции один "шумный" клиент
// заблокировал бы вообще всех остальных, включая честных пользователей —
// это была бы уже не защита, а самостоятельная DoS-уязвимость.
func TestRateLimitMiddleware_SeparateIPs(t *testing.T) {
	r := setupRateLimitRouter(1, time.Minute, KeyByIP)

	codeA := doRequest(r, "10.0.0.1", "item-1")
	codeB := doRequest(r, "10.0.0.2", "item-1")

	if codeA != http.StatusOK || codeB != http.StatusOK {
		t.Errorf("expected both IPs' first request to succeed independently, got %d and %d", codeA, codeB)
	}
}

// После истечения окна счётчик должен сброситься — блокировка не вечная.
// Проверяет обе половины поведения последовательно: (1) лимит реально
// срабатывает внутри окна (иначе тест ничего бы не доказывал про сброс),
// (2) после ожидания дольше window тот же IP снова пропускается — окно у
// каждого IP персональное и стартует заново от следующего запроса, а не
// привязано к фиксированным границам по часам (см. обсуждение в чате).
func TestRateLimitMiddleware_WindowResets(t *testing.T) {
	r := setupRateLimitRouter(1, 50*time.Millisecond, KeyByIP)

	first := doRequest(r, "9.9.9.9", "item-1")
	second := doRequest(r, "9.9.9.9", "item-1")
	if first != http.StatusOK {
		t.Errorf("expected first request to succeed, got %d", first)
	}
	if second != http.StatusTooManyRequests {
		t.Errorf("expected second request within window to be blocked, got %d", second)
	}

	time.Sleep(60 * time.Millisecond)

	third := doRequest(r, "9.9.9.9", "item-1")
	if third != http.StatusOK {
		t.Errorf("expected request after window reset to succeed, got %d", third)
	}
}

// KeyByIPAndItem: тот же IP, но РАЗНЫЕ товары — не должны делить счётчик.
// Легитимный юзер, быстро пробующий заказать несколько разных дефицитных
// лотов подряд (ровно тот сценарий, ради которого существует очередь —
// "товар только выложили"), не должен упираться в лимит, рассчитанный на
// защиту ОДНОГО конкретного товара от спама фейковыми аккаунтами.
func TestRateLimitMiddleware_KeyByIPAndItem_DifferentItemsIndependent(t *testing.T) {
	r := setupRateLimitRouter(1, time.Minute, KeyByIPAndItem)

	codeItem1 := doRequest(r, "1.1.1.1", "item-1")
	codeItem2 := doRequest(r, "1.1.1.1", "item-2")
	codeItem3 := doRequest(r, "1.1.1.1", "item-3")

	if codeItem1 != http.StatusOK || codeItem2 != http.StatusOK || codeItem3 != http.StatusOK {
		t.Errorf("expected requests to different items from same IP to succeed independently, got %d, %d, %d",
			codeItem1, codeItem2, codeItem3)
	}
}

// KeyByIPAndItem: тот же IP И тот же товар — счётчик общий, лимит всё ещё
// работает. Без этого теста предыдущий (разные товары) ничего бы не
// доказывал про реальную защиту — нужно убедиться, что "разделение по
// товару" не превратилось в "лимита нет вообще".
func TestRateLimitMiddleware_KeyByIPAndItem_SameItemBlocked(t *testing.T) {
	r := setupRateLimitRouter(1, time.Minute, KeyByIPAndItem)

	first := doRequest(r, "2.2.2.2", "item-1")
	second := doRequest(r, "2.2.2.2", "item-1")

	if first != http.StatusOK {
		t.Errorf("expected first request to succeed, got %d", first)
	}
	if second != http.StatusTooManyRequests {
		t.Errorf("expected second request to the SAME item to be blocked, got %d", second)
	}
}
