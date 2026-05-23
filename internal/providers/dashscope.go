package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-resty/resty/v2"
)

type DashScopeClient struct {
	client  *resty.Client
	baseURL string
	apiKey  string
}

const dashScopeDataInspectionHeaderValue = `{"input":"disable", "output":"disable"}`

func NewDashScopeClient(client *resty.Client, baseURL, apiKey string) *DashScopeClient {
	return &DashScopeClient{
		client:  client,
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
	}
}

type DashScopeImageRequest struct {
	Model      string              `json:"model"`
	Input      DashScopeImageInput `json:"input"`
	Parameters map[string]any      `json:"parameters,omitempty"`
}

type DashScopeImageInput struct {
	Messages []DashScopeMessage `json:"messages"`
}

type DashScopeMessage struct {
	Role    string             `json:"role"`
	Content []DashScopeContent `json:"content"`
}

type DashScopeContent struct {
	Text  string `json:"text,omitempty"`
	Image string `json:"image,omitempty"`
}

type DashScopeImageResponse struct {
	RequestID string               `json:"request_id,omitempty"`
	Output    DashScopeImageOutput `json:"output"`
	Usage     map[string]any       `json:"usage,omitempty"`
}

type DashScopeImageOutput struct {
	Choices    []DashScopeImageChoice `json:"choices,omitempty"`
	TaskID     string                 `json:"task_id,omitempty"`
	TaskStatus string                 `json:"task_status,omitempty"`
}

type DashScopeImageChoice struct {
	Message DashScopeMessage `json:"message"`
}

type DashScopeVideoRequest struct {
	Model      string              `json:"model"`
	Input      DashScopeVideoInput `json:"input"`
	Parameters map[string]any      `json:"parameters,omitempty"`
}

type DashScopeVideoInput struct {
	Prompt string                `json:"prompt,omitempty"`
	Media  []DashScopeVideoMedia `json:"media,omitempty"`
}

type DashScopeVideoMedia struct {
	Type           string `json:"type,omitempty"`
	URL            string `json:"url,omitempty"`
	ReferenceVoice string `json:"reference_voice,omitempty"`
}

type DashScopeTaskCreateResponse struct {
	RequestID string              `json:"request_id,omitempty"`
	Output    DashScopeTaskOutput `json:"output"`
	Usage     map[string]any      `json:"usage,omitempty"`
}

type DashScopeTaskResponse struct {
	RequestID string              `json:"request_id,omitempty"`
	Output    DashScopeTaskOutput `json:"output"`
	Usage     map[string]any      `json:"usage,omitempty"`
}

type DashScopeTaskOutput struct {
	TaskID     string                 `json:"task_id,omitempty"`
	TaskStatus string                 `json:"task_status,omitempty"`
	Status     string                 `json:"status,omitempty"`
	Code       string                 `json:"code,omitempty"`
	Message    string                 `json:"message,omitempty"`
	Results    []DashScopeResult      `json:"results,omitempty"`
	Choices    []DashScopeImageChoice `json:"choices,omitempty"`
	VideoURL   string                 `json:"video_url,omitempty"`
	URL        string                 `json:"url,omitempty"`
}

type DashScopeResult struct {
	URL          string `json:"url,omitempty"`
	VideoURL     string `json:"video_url,omitempty"`
	OrigPrompt   string `json:"orig_prompt,omitempty"`
	ActualPrompt string `json:"actual_prompt,omitempty"`
}

func (c *DashScopeClient) GenerateImage(ctx context.Context, requestID string, req DashScopeImageRequest) (DashScopeImageResponse, error) {
	var out DashScopeImageResponse
	if err := c.ensureConfigured(); err != nil {
		return out, err
	}
	return out, c.doJSON(ctx, http.MethodPost, "/services/aigc/multimodal-generation/generation", requestID, false, true, req, &out)
}

func (c *DashScopeClient) CreateImageTask(ctx context.Context, requestID string, req DashScopeImageRequest) (DashScopeTaskCreateResponse, error) {
	var out DashScopeTaskCreateResponse
	if err := c.ensureConfigured(); err != nil {
		return out, err
	}
	return out, c.doJSON(ctx, http.MethodPost, "/services/aigc/image-generation/generation", requestID, true, true, req, &out)
}

func (c *DashScopeClient) CreateVideoTask(ctx context.Context, requestID string, req DashScopeVideoRequest) (DashScopeTaskCreateResponse, error) {
	var out DashScopeTaskCreateResponse
	if err := c.ensureConfigured(); err != nil {
		return out, err
	}
	return out, c.doJSON(ctx, http.MethodPost, "/services/aigc/video-generation/video-synthesis", requestID, true, true, req, &out)
}

func (c *DashScopeClient) GetTask(ctx context.Context, requestID, taskID string) (DashScopeTaskResponse, error) {
	var out DashScopeTaskResponse
	if err := c.ensureConfigured(); err != nil {
		return out, err
	}
	return out, c.doJSON(ctx, http.MethodGet, "/tasks/"+taskID, requestID, false, false, nil, &out)
}

func (c *DashScopeClient) ensureConfigured() error {
	if c.baseURL == "" || c.apiKey == "" {
		return providerNotConfigured("dashscope api key is not configured")
	}
	return nil
}

func (c *DashScopeClient) doJSON(ctx context.Context, method, path, requestID string, async bool, dataInspection bool, body any, out any) error {
	req := c.client.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+c.apiKey).
		SetHeader("Content-Type", "application/json").
		SetHeader("X-Request-Id", requestID)
	if async {
		req.SetHeader("X-DashScope-Async", "enable")
	}
	if dataInspection {
		req.SetHeader("X-DashScope-DataInspection", dashScopeDataInspectionHeaderValue)
	}
	if body != nil {
		req.SetBody(body)
	}

	resp, err := req.Execute(method, c.baseURL+path)
	if err != nil {
		return err
	}
	if resp.IsError() {
		return providerHTTPError("dashscope provider", resp)
	}
	if len(resp.Body()) == 0 {
		return nil
	}
	if err := json.Unmarshal(resp.Body(), out); err != nil {
		return ProviderError{Status: http.StatusBadGateway, Code: "provider_bad_response", Message: "invalid dashscope provider response"}
	}
	return nil
}
