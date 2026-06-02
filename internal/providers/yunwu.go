package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-resty/resty/v2"
)

type YunwuClient struct {
	client  *resty.Client
	baseURL string
	apiKey  string
}

func NewYunwuClient(client *resty.Client, baseURL, apiKey string) *YunwuClient {
	return &YunwuClient{
		client:  client,
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
	}
}

type YunwuGeminiRequest struct {
	Contents         []YunwuGeminiContent        `json:"contents"`
	GenerationConfig YunwuGeminiGenerationConfig `json:"generationConfig"`
}

type YunwuGeminiGenerationConfig struct {
	ResponseModalities []string `json:"response_modalities"`
}

type YunwuGeminiContent struct {
	Role  string            `json:"role"`
	Parts []YunwuGeminiPart `json:"parts"`
}

type YunwuGeminiPart struct {
	Text            string                 `json:"text,omitempty"`
	InlineData      *YunwuGeminiInlineData `json:"inlineData,omitempty"`
	InlineDataSnake *YunwuGeminiInlineData `json:"inline_data,omitempty"`
	FileData        *YunwuGeminiFileData   `json:"fileData,omitempty"`
}

type YunwuGeminiInlineData struct {
	MimeType      string `json:"mimeType,omitempty"`
	MimeTypeSnake string `json:"mime_type,omitempty"`
	Data          string `json:"data,omitempty"`
}

type YunwuGeminiFileData struct {
	MimeType string `json:"mimeType,omitempty"`
	FileURI  string `json:"fileUri,omitempty"`
}

type YunwuGeminiResponse struct {
	Candidates []YunwuGeminiCandidate `json:"candidates,omitempty"`
	Usage      map[string]any         `json:"usageMetadata,omitempty"`
}

type YunwuGeminiCandidate struct {
	Content      YunwuGeminiContent `json:"content"`
	FinishReason string             `json:"finishReason,omitempty"`
}

type YunwuImageRequest struct {
	Model   string `json:"model"`
	Prompt  string `json:"prompt"`
	N       *int   `json:"n,omitempty"`
	Size    string `json:"size,omitempty"`
	Quality string `json:"quality,omitempty"`
	Format  string `json:"format,omitempty"`
}

type YunwuImageResponse struct {
	Created int64            `json:"created,omitempty"`
	Data    []YunwuImageData `json:"data,omitempty"`
	Usage   map[string]any   `json:"usage,omitempty"`
}

type YunwuImageData struct {
	URL      string `json:"url,omitempty"`
	B64JSON  string `json:"b64_json,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
}

func (c *YunwuClient) GenerateGeminiImage(ctx context.Context, requestID, model string, req YunwuGeminiRequest) (YunwuGeminiResponse, error) {
	var out YunwuGeminiResponse
	if err := c.ensureConfigured(); err != nil {
		return out, err
	}
	path := "/v1beta/models/" + model + ":generateContent"
	return out, c.doJSON(ctx, http.MethodPost, path, requestID, req, &out)
}

func (c *YunwuClient) GenerateImage(ctx context.Context, requestID string, req YunwuImageRequest) (YunwuImageResponse, error) {
	var out YunwuImageResponse
	if err := c.ensureConfigured(); err != nil {
		return out, err
	}
	return out, c.doJSON(ctx, http.MethodPost, "/v1/images/generations", requestID, req, &out)
}

func (c *YunwuClient) ensureConfigured() error {
	if c.baseURL == "" || c.apiKey == "" {
		return providerNotConfigured("yunwu api key is not configured")
	}
	return nil
}

func (c *YunwuClient) doJSON(ctx context.Context, method, path, requestID string, body any, out any) error {
	req := c.client.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+c.apiKey).
		SetHeader("Content-Type", "application/json").
		SetHeader("X-Request-Id", requestID)
	if body != nil {
		req.SetBody(body)
	}

	resp, err := req.Execute(method, c.baseURL+path)
	if err != nil {
		return err
	}
	if resp.IsError() {
		return providerHTTPError("yunwu provider", resp)
	}
	if len(resp.Body()) == 0 {
		return nil
	}
	if err := json.Unmarshal(resp.Body(), out); err != nil {
		return ProviderError{Status: http.StatusBadGateway, Code: "provider_bad_response", Message: "invalid yunwu provider response"}
	}
	return nil
}
