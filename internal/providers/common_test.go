package providers

import (
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
)

func TestCurlCommandRedactsAuthorization(t *testing.T) {
	req := resty.New().R().
		SetHeader("Authorization", "Bearer sk-secret").
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]any{
			"model":  "gpt-image-2",
			"prompt": "city",
		})
	req.Method = "POST"
	req.URL = "https://yunwu.ai/v1/images/generations"

	command := CurlCommand(req)
	for _, want := range []string{
		"curl 'https://yunwu.ai/v1/images/generations'",
		"-X 'POST'",
		"-H 'Authorization: Bearer ***'",
		"-H 'Content-Type: application/json'",
		"-d '{",
		"\"model\":\"gpt-image-2\"",
		"\"prompt\":\"city\"",
	} {
		if !strings.Contains(command, want) {
			t.Fatalf("curl command missing %q: %s", want, command)
		}
	}
	if strings.Contains(command, "sk-secret") {
		t.Fatalf("curl command leaked secret: %s", command)
	}
}

func TestLogCurlOnBeforeRequest(t *testing.T) {
	var logged string
	err := LogCurlOnBeforeRequest(func(command string) {
		logged = command
	})(nil, resty.New().R())
	if err != nil {
		t.Fatalf("LogCurlOnBeforeRequest returned error: %v", err)
	}
	if logged == "" {
		t.Fatal("expected logged curl command")
	}
}
