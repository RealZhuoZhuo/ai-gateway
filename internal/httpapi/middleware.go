package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/RealZhuoZhuo/ai-gateway/internal/httpapi/dto"
	"github.com/RealZhuoZhuo/ai-gateway/internal/service"
)

const requestIDKey = "request_id"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader("X-Request-Id"))
		if requestID == "" {
			requestID = uuid.NewString()
		}
		c.Set(requestIDKey, requestID)
		c.Header("X-Request-Id", requestID)
		c.Next()
	}
}

func Logger(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.WithFields(logrus.Fields{
			"request_id": RequestIDFromContext(c),
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"status":     c.Writer.Status(),
			"latency_ms": time.Since(start).Milliseconds(),
			"client_ip":  c.ClientIP(),
		}).Info("request completed")
	}
}

func Recoverer(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.WithFields(logrus.Fields{
					"panic":      recovered,
					"request_id": RequestIDFromContext(c),
				}).Error("panic recovered")
				WriteError(c, dto.NewAPIError(http.StatusInternalServerError, "internal_error", "internal server error"))
				c.Abort()
			}
		}()
		c.Next()
	}
}

func Auth(authenticator *service.Authenticator, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		token, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || strings.TrimSpace(token) == "" {
			WriteError(c, dto.NewAPIError(http.StatusUnauthorized, "unauthorized", "missing bearer token"))
			c.Abort()
			return
		}

		allowed, err := authenticator.ValidAPIKey(c.Request.Context(), strings.TrimSpace(token))
		if err != nil {
			logger.WithError(err).WithField("request_id", RequestIDFromContext(c)).Error("api key validation failed")
			WriteError(c, dto.NewAPIError(http.StatusInternalServerError, "auth_error", "api key validation failed"))
			c.Abort()
			return
		}
		if !allowed {
			WriteError(c, dto.NewAPIError(http.StatusUnauthorized, "unauthorized", "invalid api key"))
			c.Abort()
			return
		}
		c.Next()
	}
}

func RequestIDFromContext(c *gin.Context) string {
	if value, ok := c.Get(requestIDKey); ok {
		if requestID, ok := value.(string); ok {
			return requestID
		}
	}
	return ""
}

func DecodeJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		WriteError(c, dto.NewAPIError(http.StatusBadRequest, "invalid_request", "invalid json body"))
		return false
	}
	return true
}

func WriteJSON(c *gin.Context, status int, body any) {
	c.Header("X-Request-Id", RequestIDFromContext(c))
	c.JSON(status, body)
}

func WriteError(c *gin.Context, err error) {
	var apiErr dto.APIError
	if !errors.As(err, &apiErr) {
		apiErr = dto.NewAPIError(http.StatusInternalServerError, "internal_error", "internal server error")
	}
	if apiErr.Status == 0 {
		apiErr.Status = http.StatusInternalServerError
	}
	WriteJSON(c, apiErr.Status, dto.ErrorBody{
		Error: dto.ErrorDetail{
			Code:      apiErr.Code,
			Message:   apiErr.Message,
			RequestID: RequestIDFromContext(c),
		},
	})
}
