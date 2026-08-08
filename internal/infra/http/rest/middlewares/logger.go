package middlewares

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

func LoggerMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger.Info("starting request", "method", c.Request.Method, "path", c.Request.URL.Path)
		c.Next()
	}
}
