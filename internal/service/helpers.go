package service

import (
	"errors"
	"net/http"
	"strings"

	"github.com/RealZhuoZhuo/ai-gateway/internal/common"
	"github.com/RealZhuoZhuo/ai-gateway/internal/providers"
)

func requireString(value, field string) error {
	if strings.TrimSpace(value) != "" {
		return nil
	}
	return invalidRequest(field + "为必填项")
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func nullableFirst(values []string) *string {
	if len(values) == 0 {
		return nil
	}
	return nullableString(values[0])
}

func normalizeTaskStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "queued", "pending", "created":
		return "queued"
	case "running", "processing":
		return "running"
	case "cancelled", "canceled":
		return "cancelled"
	case "succeeded", "success", "completed":
		return "succeeded"
	case "failed", "error":
		return "failed"
	case "expired":
		return "expired"
	case "":
		return "unknown"
	default:
		return "unknown"
	}
}

func providerError(err error) error {
	var providerErr providers.ProviderError
	if !errors.As(err, &providerErr) {
		return newError(http.StatusBadGateway, "provider_error", err.Error())
	}

	status := providerErr.Status
	if status == 0 {
		status = http.StatusBadGateway
	} else if status >= 500 && providerErr.Code != "provider_not_configured" {
		status = http.StatusBadGateway
	}
	return newError(status, providerErr.Code, providerErr.Message)
}

func taskErrorFromProvider(err *providers.ProviderTaskError) *TaskError {
	if err == nil {
		return nil
	}
	return &TaskError{Code: common.DefaultString(err.Code, "provider_error"), Message: err.Message}
}
