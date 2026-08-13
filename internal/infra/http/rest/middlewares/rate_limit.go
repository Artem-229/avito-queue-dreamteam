package middlewares

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type attempt struct {
	count     int
	windowEnd time.Time
}

// RateLimitMiddleware — простой лимитер с фиксированным окном (не скользящим):
// не более maxAttempts запросов с одним и тем же ключом за window. Ключ
// вычисляется через keyFunc — это важно: считать лимит по одному IP
// одинаково для всех маршрутов оказалось неверным решением (см. обсуждение
// с командой). Для /admin/login угроза — подбор ключа с одного IP, там
// ключ = сам IP. Для входа в очередь угроза — много РАЗНЫХ user_id с
// одного IP спамящих ОДИН конкретный товар, а не легитимный юзер, быстро
// нажимающий "заказать" на разных дефицитных лотах подряд (ровно тот
// сценарий, ради которого существует этот кейс) — поэтому там ключ
// комбинирует IP + item_id: атака на конкретный товар всё ещё тормозится,
// а покупки на нескольких разных товарах подряд не задевает вообще.
func RateLimitMiddleware(maxAttempts int, window time.Duration, keyFunc func(c *gin.Context) string) gin.HandlerFunc {
	var mu sync.Mutex
	attempts := make(map[string]*attempt)

	return func(c *gin.Context) {
		key := keyFunc(c)
		now := time.Now()

		mu.Lock()
		a, exists := attempts[key]
		if !exists || now.After(a.windowEnd) {
			a = &attempt{count: 0, windowEnd: now.Add(window)}
			attempts[key] = a
		}
		a.count++
		blocked := a.count > maxAttempts
		mu.Unlock()

		if blocked {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, errorBody{
				Code:    "TOO_MANY_REQUESTS",
				Message: "Слишком много попыток, попробуйте позже",
			})
			return
		}

		c.Next()
	}
}

// KeyByIP — лимит считается по одному IP вне зависимости от маршрута.
// Подходит там, где нет естественного "предмета" запроса (например,
// /admin/login — угроза это подбор ключа с одного IP, не привязана к
// конкретному товару).
func KeyByIP(c *gin.Context) string {
	return c.ClientIP()
}

// KeyByIPAndItem — лимит считается отдельно на каждую пару (IP, item_id).
// Подходит для входа в очередь: не наказывает легитимного юзера,
// быстро пробующего разные дефицитные товары, но по-прежнему тормозит
// спам множеством фейковых аккаунтов в ОДИН конкретный товар.
func KeyByIPAndItem(c *gin.Context) string {
	return c.ClientIP() + ":" + c.Param("id")
}
