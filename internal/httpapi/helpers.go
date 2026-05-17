package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/RealZhuoZhuo/ai-gateway/internal/httpapi/dto"
	"github.com/RealZhuoZhuo/ai-gateway/internal/service"
)

func writeServiceError(c *gin.Context, err error) {
	var serviceErr service.Error
	if errors.As(err, &serviceErr) {
		WriteError(c, dto.NewAPIError(serviceErr.Status, serviceErr.Code, serviceErr.Message))
		return
	}
	WriteError(c, dto.NewAPIError(http.StatusInternalServerError, "internal_error", "internal server error"))
}
