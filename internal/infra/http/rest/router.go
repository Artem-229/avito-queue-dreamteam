package rest

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"avito-queue/internal/config"
	"avito-queue/internal/infra/http/rest/handlers"
	"avito-queue/internal/infra/http/rest/middlewares"
)

func NewRouter(h *handlers.Handlers, conf *config.Config, logger *slog.Logger) *gin.Engine {
	router := gin.Default()

	// Без этого gin отвечает на неизвестный путь текстом "404 page not found",
	// а клиент разбирает тело ошибки как JSON: разбор падает, и вместо
	// сообщения пользователь видит HTTP statusText. Формат ошибки должен быть
	// один на всё API, включая ответы самого роутера.
	router.NoRoute(h.NotFound)
	// NoMethod без этого флага не вызывается: gin по умолчанию отвечает 404 и
	// на существующий путь с чужим методом.
	router.HandleMethodNotAllowed = true
	router.NoMethod(h.MethodNotAllowed)

	router.GET("/health", h.Health)

	api := router.Group("/api/v1")

	// Вход регистрируется до подключения SessionAuthMiddleware: получить
	// сессию, уже имея сессию, невозможно.
	api.POST("/demo/login", h.DemoLogin(conf.Auth.SessionSecret, conf.Auth.SecureCookie))
	api.POST("/admin/login", h.AdminLogin(conf.Admin.SecretKey))

	api.Use(middlewares.SessionAuthMiddleware(conf.Auth.SessionSecret))
	api.Use(middlewares.LoggerMiddleware(logger))

	{
		api.GET("/catalog", h.Catalog.GetList)
		api.GET("/catalog/:id", h.Catalog.GetByID)
		api.GET("/catalog/:id/similar", h.Catalog.GetSimilarItems)

		api.GET("/stats", h.Stats.GetStats)

		api.POST("/catalog/:id/queue", h.QueueEntry.Join)
		api.GET("/catalog/:id/queue/me", h.QueueEntry.GetStatus)
		api.DELETE("/catalog/:id/queue/me", h.QueueEntry.Leave)
		api.POST("/catalog/:id/purchase", h.QueueEntry.Purchase)

		// Демо-ручки под флагом: expire-now сжигает все выданные права на
		// товар, simulate заводит до сотни покупателей. На публичном стенде
		// это кнопка «сломать демонстрацию», поэтому по умолчанию их нет.
		// Флаг, а не удаление: симулятор нужен на самой защите.
		if conf.Demo.Enabled {
			demo := api.Group("/demo")
			demo.POST("/items/:id/expire-now", h.Demo.ExpireNow)
			demo.POST("/simulate", h.Demo.Simulate)
		}
	}

	admin := api.Group("/admin")
	admin.Use(middlewares.AdminAuthMiddleware(conf.Admin.SecretKey))
	{
		admin.POST("/catalog", h.Catalog.Create)
		admin.PATCH("/catalog/:id", h.Catalog.Patch)
		admin.DELETE("/catalog/:id", h.Catalog.Delete)
	}

	return router
}
