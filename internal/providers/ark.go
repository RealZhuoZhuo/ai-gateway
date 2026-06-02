package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-resty/resty/v2"
)

type ArkClient struct {
	client        *resty.Client
	imageEndpoint string
	imageAPIKey   string
	videoEndpoint string
	videoAPIKey   string
}

func NewArkClient(client *resty.Client, imageEndpoint, imageAPIKey, videoEndpoint, videoAPIKey string) *ArkClient {
	return &ArkClient{
		client:        client,
		imageEndpoint: imageEndpoint,
		imageAPIKey:   imageAPIKey,
		videoEndpoint: videoEndpoint,
		videoAPIKey:   videoAPIKey,
	}
}

type ArkImageRequest struct {
	Model                     string `json:"model"`
	Prompt                    string `json:"prompt"`
	SequentialImageGeneration string `json:"sequential_image_generation"`
	ResponseFormat            string `json:"response_format"`
	Size                      string `json:"size"`
	Stream                    bool   `json:"stream"`
	Watermark                 bool   `json:"watermark"`
	Image                     any    `json:"image,omitempty"`
}

type ArkImageResponse struct {
	Model   string         `json:"model,omitempty"`
	Created int64          `json:"created,omitempty"`
	Data    []ArkImageData `json:"data"`
	URL     string         `json:"url,omitempty"`
	Usage   map[string]any `json:"usage,omitempty"`
}

type ArkImageData struct {
	URL  string `json:"url"`
	Size string `json:"size,omitempty"`
}

func (c *ArkClient) GenerateImage(ctx context.Context, requestID string, req ArkImageRequest) (ArkImageResponse, error) {
	var out ArkImageResponse
	if c.imageAPIKey == "" {
		return out, providerNotConfigured("ark image api key is not configured")
	}
	return out, c.doJSON(ctx, http.MethodPost, c.imageEndpoint, c.imageAPIKey, requestID, req, &out)
}

type ArkVideoCreateRequest struct {
	Model         string            `json:"model"`
	Content       []ArkVideoContent `json:"content"`
	GenerateAudio *bool             `json:"generate_audio,omitempty"`
	Ratio         string            `json:"ratio,omitempty"`
	Duration      *int              `json:"duration,omitempty"`
	Watermark     *bool             `json:"watermark,omitempty"`
	Seed          *int64            `json:"seed,omitempty"`
	Resolution    string            `json:"resolution,omitempty"`
}

type ArkVideoContent struct {
	Type     string            `json:"type"`
	Text     string            `json:"text,omitempty"`
	ImageURL *ArkVideoImageURL `json:"image_url,omitempty"`
	VideoURL *ArkVideoMediaURL `json:"video_url,omitempty"`
	AudioURL *ArkVideoMediaURL `json:"audio_url,omitempty"`
	Role     string            `json:"role,omitempty"`
}

type ArkVideoImageURL struct {
	URL string `json:"url"`
}

type ArkVideoMediaURL struct {
	URL string `json:"url"`
}

type ArkVideoCreateResponse struct {
	ID     string `json:"id,omitempty"`
	TaskID string `json:"task_id,omitempty"`
	Status string `json:"status,omitempty"`
	Model  string `json:"model,omitempty"`
}

func (c *ArkClient) CreateVideoTask(ctx context.Context, requestID string, req ArkVideoCreateRequest) (ArkVideoCreateResponse, error) {
	var out ArkVideoCreateResponse
	if c.videoAPIKey == "" {
		return out, providerNotConfigured("ark video api key is not configured")
	}
	return out, c.doJSON(ctx, http.MethodPost, c.videoEndpoint, c.videoAPIKey, requestID, req, &out)
}

type ArkVideoTaskResponse struct {
	ID           string             `json:"id,omitempty"`
	TaskID       string             `json:"task_id,omitempty"`
	Model        string             `json:"model,omitempty"`
	Status       string             `json:"status,omitempty"`
	Content      *ArkTaskContent    `json:"content,omitempty"`
	VideoURL     string             `json:"video_url,omitempty"`
	LastFrameURL string             `json:"last_frame_url,omitempty"`
	Error        *ProviderTaskError `json:"error,omitempty"`
	CreatedAt    int64              `json:"created_at,omitempty"`
	UpdatedAt    int64              `json:"updated_at,omitempty"`
	Seed         *int64             `json:"seed,omitempty"`
	Resolution   string             `json:"resolution,omitempty"`
	Ratio        string             `json:"ratio,omitempty"`
	Duration     *int               `json:"duration,omitempty"`
	Usage        map[string]any     `json:"usage,omitempty"`
}

type ArkTaskContent struct {
	VideoURL     string `json:"video_url,omitempty"`
	LastFrameURL string `json:"last_frame_url,omitempty"`
	URL          string `json:"url,omitempty"`
}

type ProviderTaskError struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func (c *ArkClient) GetVideoTask(ctx context.Context, requestID, taskID string) (ArkVideoTaskResponse, error) {
	var out ArkVideoTaskResponse
	if c.videoAPIKey == "" {
		return out, providerNotConfigured("ark video api key is not configured")
	}
	endpoint := strings.TrimRight(c.videoEndpoint, "/") + "/" + taskID
	return out, c.doJSON(ctx, http.MethodGet, endpoint, c.videoAPIKey, requestID, nil, &out)
}

func (c *ArkClient) doJSON(ctx context.Context, method, url, apiKey, requestID string, body any, out any) error {
	req := c.client.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+apiKey).
		SetHeader("Content-Type", "application/json").
		SetHeader("X-Request-Id", requestID)
	if body != nil {
		req.SetBody(body)
	}

	resp, err := req.Execute(method, url)
	if err != nil {
		return err
	}
	if resp.IsError() {
		return providerHTTPError("ark provider", resp)
	}
	if len(resp.Body()) == 0 {
		return nil
	}
	if err := json.Unmarshal(resp.Body(), out); err != nil {
		return ProviderError{Status: http.StatusBadGateway, Code: "provider_bad_response", Message: "invalid ark provider response"}
	}
	return nil
}
