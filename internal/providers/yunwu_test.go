package providers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
)

func TestYunwuGenerateGeminiImage(t *testing.T) {
	var gotPath string
	var gotAuth string
	var gotRequestID string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotRequestID = r.Header.Get("X-Request-Id")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"candidates": [{
				"content": {
					"parts": [
						{"text": "done"},
						{"inlineData": {"mimeType": "image/jpeg", "data": "abc123"}}
					]
				},
				"finishReason": "STOP"
			}],
			"usageMetadata": {"totalTokenCount": 3}
		}`))
	}))
	defer server.Close()

	client := NewYunwuClient(resty.New(), server.URL, "sk-test")
	out, err := client.GenerateGeminiImage(context.Background(), "rid-1", "gemini-3-pro-image-preview", YunwuGeminiRequest{
		Contents: []YunwuGeminiContent{{
			Role:  "user",
			Parts: []YunwuGeminiPart{{Text: "draw"}},
		}},
		GenerationConfig: YunwuGeminiGenerationConfig{ResponseModalities: []string{"IMAGE", "TEXT"}},
	})
	if err != nil {
		t.Fatalf("GenerateGeminiImage returned error: %v", err)
	}
	if gotPath != "/v1beta/models/gemini-3-pro-image-preview:generateContent" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotAuth != "Bearer sk-test" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotRequestID != "rid-1" {
		t.Fatalf("X-Request-Id = %q", gotRequestID)
	}
	config, _ := gotBody["generationConfig"].(map[string]any)
	modalities, _ := config["response_modalities"].([]any)
	if len(modalities) != 2 || modalities[0] != "IMAGE" || modalities[1] != "TEXT" {
		t.Fatalf("response_modalities = %#v", modalities)
	}
	if len(out.Candidates) != 1 || len(out.Candidates[0].Content.Parts) != 2 {
		t.Fatalf("unexpected candidates: %#v", out.Candidates)
	}
	inline := out.Candidates[0].Content.Parts[1].InlineData
	if inline == nil || inline.MimeType != "image/jpeg" || inline.Data != "abc123" {
		t.Fatalf("inline data = %#v", inline)
	}
}

func TestYunwuGenerateImage(t *testing.T) {
	var gotPath string
	var gotBody YunwuImageRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"created": 123,
			"data": [
				{"url": "https://cdn.example/image.jpg"},
				{"b64_json": "xyz", "mime_type": "image/jpeg"}
			],
			"usage": {"images": 2}
		}`))
	}))
	defer server.Close()

	n := 2
	client := NewYunwuClient(resty.New(), server.URL, "sk-test")
	out, err := client.GenerateImage(context.Background(), "rid-2", YunwuImageRequest{
		Model:   "gpt-image-2",
		Prompt:  "city",
		N:       &n,
		Size:    "1024x1024",
		Quality: "low",
		Format:  "jpeg",
	})
	if err != nil {
		t.Fatalf("GenerateImage returned error: %v", err)
	}
	if gotPath != "/v1/images/generations" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotBody.Model != "gpt-image-2" || gotBody.Prompt != "city" || gotBody.N == nil || *gotBody.N != 2 || gotBody.Size != "1024x1024" || gotBody.Quality != "low" || gotBody.Format != "jpeg" {
		t.Fatalf("request body = %#v", gotBody)
	}
	if out.Created != 123 || len(out.Data) != 2 || out.Data[0].URL != "https://cdn.example/image.jpg" || out.Data[1].B64JSON != "xyz" {
		t.Fatalf("response = %#v", out)
	}
}

func TestYunwuNotConfigured(t *testing.T) {
	client := NewYunwuClient(resty.New(), "https://yunwu.ai", "")
	_, err := client.GenerateImage(context.Background(), "rid-3", YunwuImageRequest{Model: "gpt-image-2", Prompt: "city"})
	var providerErr ProviderError
	if !errors.As(err, &providerErr) {
		t.Fatalf("error = %T %v", err, err)
	}
	if providerErr.Code != "provider_not_configured" {
		t.Fatalf("code = %q", providerErr.Code)
	}
}
