package middlewares

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"avito-queue/internal/infra/http/rest/session"
)

const UserIDKey = "userID"

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// SessionAuthMiddleware пускает дальше только с валидным подписанным токеном.
// Неподписанного пути (вроде голого X-User-Id) быть не должно: безопасность
// определяется самым слабым принимаемым доказательством личности.
//
// Токен принимается из двух мест и проверяется одинаково: заголовком
// Authorization: Bearer (им пользуется переключатель тестовых аккаунтов — кука
// httpOnly, JS её не читает) или кукой session для обычного входа. Заголовок
// проверяется первым, чтобы явно выбранный аккаунт побеждал фоновую куку.
func SessionAuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := tokenFromRequest(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorBody{
				Code:    "UNAUTHORIZED",
				Message: "Требуется авторизация: выполните вход",
			})
			return
		}

		userID, err := session.Parse(token, secret, time.Now())
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorBody{
				Code:    "INVALID_SESSION",
				Message: "Сессия недействительна или истекла, выполните вход заново",
			})
			return
		}

		c.Set(UserIDKey, userID)
		c.Next()
	}
}

func tokenFromRequest(c *gin.Context) (string, bool) {
	if raw := c.GetHeader("Authorization"); raw != "" {
		if token, found := strings.CutPrefix(raw, "Bearer "); found {
			if token = strings.TrimSpace(token); token != "" {
				return token, true
			}
		}
	}

	if cookie, err := c.Cookie(session.CookieName); err == nil && cookie != "" {
		return cookie, true
	}

	return "", false
}
