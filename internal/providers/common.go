package providers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-resty/resty/v2"

	"github.com/RealZhuoZhuo/ai-gateway/internal/common"
)

type ProviderError struct {
	Status  int
	Code    string
	Message string
}

func (e ProviderError) Error() string {
	return e.Message
}

func providerNotConfigured(message string) ProviderError {
	return ProviderError{Status: http.StatusServiceUnavailable, Code: "provider_not_configured", Message: message}
}

func providerHTTPError(provider string, resp *resty.Response) ProviderError {
	return ProviderError{
		Status:  resp.StatusCode(),
		Code:    "provider_error",
		Message: fmt.Sprintf("%s returned status %d: %s", provider, resp.StatusCode(), string(resp.Body())),
	}
}

func firstNonEmpty(values ...string) string {
	return common.FirstNonEmpty(values...)
}

func headersToMap(headers http.Header) map[string]string {
	out := make(map[string]string, len(headers))
	for key, values := range headers {
		if len(values) > 0 {
			out[key] = strings.Join(values, ",")
		}
	}
	return out
}

func firstNonNilStringSlice(values ...[]string) []string {
	for _, value := range values {
		if len(value) > 0 {
			return value
		}
	}
	return nil
}
