package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/RealZhuoZhuo/ai-gateway/internal/httpapi/dto"
	"github.com/RealZhuoZhuo/ai-gateway/internal/service"
)

func NewRouter(handler *Handler, authenticator *service.Authenticator, logger *logrus.Logger) http.Handler {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(RequestID())
	r.Use(Logger(logger))
	r.Use(Recoverer(logger))
	r.Use(Auth(authenticator, logger))

	r.GET("/healthz", handler.Healthz)

	v1 := r.Group("/v1")
	{
		v1.POST("/images/generations", handler.GenerateImage)
		v1.GET("/images/generations/:task_id", handler.GetImageTask)
		v1.POST("/video/generations", handler.CreateVideoTask)
		v1.GET("/video/generations/:task_id", handler.GetVideoTask)
		v1.POST("/video/tasks", handler.CreateVideoTask)
		v1.GET("/video/tasks/:task_id", handler.GetVideoTask)
	}

	r.NoRoute(func(c *gin.Context) {
		WriteError(c, dto.NewAPIError(http.StatusNotFound, "not_found", "路由不存在"))
	})
	r.NoMethod(func(c *gin.Context) {
		WriteError(c, dto.NewAPIError(http.StatusMethodNotAllowed, "method_not_allowed", "请求方法不允许"))
	})

	return r
}
