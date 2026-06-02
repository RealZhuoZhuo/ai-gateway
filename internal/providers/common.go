package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
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

func LogCurlOnBeforeRequest(log func(string)) resty.RequestMiddleware {
	return func(_ *resty.Client, req *resty.Request) error {
		if log != nil {
			log(CurlCommand(req))
		}
		return nil
	}
}

func CurlCommand(req *resty.Request) string {
	parts := []string{"curl", shellQuote(req.URL)}
	if req.Method != "" {
		parts = append(parts, "-X", shellQuote(req.Method))
	}
	for _, header := range curlHeaders(req.Header) {
		parts = append(parts, "-H", shellQuote(header))
	}
	if body := curlBody(req.Body); body != "" {
		parts = append(parts, "-d", shellQuote(body))
	}
	return strings.Join(parts, " ")
}

func curlHeaders(headers http.Header) []string {
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]string, 0, len(headers))
	for _, key := range keys {
		for _, value := range headers.Values(key) {
			out = append(out, key+": "+redactHeaderValue(key, value))
		}
	}
	return out
}

func redactHeaderValue(key, value string) string {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "authorization":
		if value == "" {
			return value
		}
		if fields := strings.Fields(value); len(fields) > 0 {
			return fields[0] + " ***"
		}
		return "***"
	default:
		return value
	}
}

func curlBody(body any) string {
	if body == nil {
		return ""
	}
	switch typed := body.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprint(typed)
		}
		return string(data)
	}
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
