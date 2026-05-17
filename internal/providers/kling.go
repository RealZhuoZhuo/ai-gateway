package providers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

type KlingClient struct {
	client    *resty.Client
	baseURL   string
	accessKey string
	secretKey string
}

func NewKlingClient(client *resty.Client, baseURL, accessKey, secretKey string) *KlingClient {
	return &KlingClient{
		client:    client,
		baseURL:   strings.TrimRight(baseURL, "/"),
		accessKey: accessKey,
		secretKey: secretKey,
	}
}

type KlingImageCreateRequest struct {
	ModelName      string   `json:"model_name,omitempty"`
	Prompt         string   `json:"prompt"`
	NegativePrompt string   `json:"negative_prompt,omitempty"`
	Image          string   `json:"image,omitempty"`
	ImageReference string   `json:"image_reference,omitempty"`
	ImageFidelity  *float64 `json:"image_fidelity,omitempty"`
	HumanFidelity  *float64 `json:"human_fidelity,omitempty"`
	Resolution     string   `json:"resolution,omitempty"`
	N              *int     `json:"n,omitempty"`
	AspectRatio    string   `json:"aspect_ratio,omitempty"`
	CallbackURL    string   `json:"callback_url,omitempty"`
	ExternalTaskID string   `json:"external_task_id,omitempty"`
}

type KlingVideoCreateRequest struct {
	ModelName      string         `json:"model_name,omitempty"`
	Prompt         string         `json:"prompt,omitempty"`
	NegativePrompt string         `json:"negative_prompt,omitempty"`
	Image          string         `json:"image,omitempty"`
	ImageTail      string         `json:"image_tail,omitempty"`
	Mode           string         `json:"mode,omitempty"`
	Duration       string         `json:"duration,omitempty"`
	AspectRatio    string         `json:"aspect_ratio,omitempty"`
	CFGScale       *float64       `json:"cfg_scale,omitempty"`
	Sound          string         `json:"sound,omitempty"`
	StaticMask     any            `json:"static_mask,omitempty"`
	DynamicMasks   any            `json:"dynamic_masks,omitempty"`
	CameraControl  map[string]any `json:"camera_control,omitempty"`
	CallbackURL    string         `json:"callback_url,omitempty"`
	ExternalTaskID string         `json:"external_task_id,omitempty"`
}

type KlingTaskResponse struct {
	Code      int           `json:"code,omitempty"`
	Message   string        `json:"message,omitempty"`
	Data      KlingTaskData `json:"data"`
	RequestID string        `json:"request_id,omitempty"`
}

type KlingTaskData struct {
	TaskID     string             `json:"task_id,omitempty"`
	TaskStatus string             `json:"task_status,omitempty"`
	TaskInfo   map[string]any     `json:"task_info,omitempty"`
	TaskResult *KlingTaskResult   `json:"task_result,omitempty"`
	Error      *ProviderTaskError `json:"error,omitempty"`
	CreatedAt  int64              `json:"created_at,omitempty"`
	UpdatedAt  int64              `json:"updated_at,omitempty"`
}

type KlingTaskResult struct {
	Images []KlingImageResult `json:"images,omitempty"`
	Videos []KlingVideoResult `json:"videos,omitempty"`
}

type KlingImageResult struct {
	Index int    `json:"index,omitempty"`
	URL   string `json:"url,omitempty"`
}

type KlingVideoResult struct {
	ID       string `json:"id,omitempty"`
	URL      string `json:"url,omitempty"`
	Duration string `json:"duration,omitempty"`
}

func (c *KlingClient) CreateImageTask(ctx context.Context, requestID string, req KlingImageCreateRequest) (KlingTaskResponse, error) {
	var out KlingTaskResponse
	if err := c.ensureConfigured(); err != nil {
		return out, err
	}
	return out, c.doJSON(ctx, http.MethodPost, "/v1/images/generations", requestID, req, &out)
}

func (c *KlingClient) GetImageTask(ctx context.Context, requestID, taskID string) (KlingTaskResponse, error) {
	var out KlingTaskResponse
	if err := c.ensureConfigured(); err != nil {
		return out, err
	}
	return out, c.doJSON(ctx, http.MethodGet, "/v1/images/generations/"+taskID, requestID, nil, &out)
}

func (c *KlingClient) CreateTextToVideoTask(ctx context.Context, requestID string, req KlingVideoCreateRequest) (KlingTaskResponse, error) {
	var out KlingTaskResponse
	if err := c.ensureConfigured(); err != nil {
		return out, err
	}
	return out, c.doJSON(ctx, http.MethodPost, "/v1/videos/text2video", requestID, req, &out)
}

func (c *KlingClient) GetTextToVideoTask(ctx context.Context, requestID, taskID string) (KlingTaskResponse, error) {
	var out KlingTaskResponse
	if err := c.ensureConfigured(); err != nil {
		return out, err
	}
	return out, c.doJSON(ctx, http.MethodGet, "/v1/videos/text2video/"+taskID, requestID, nil, &out)
}

func (c *KlingClient) CreateImageToVideoTask(ctx context.Context, requestID string, req KlingVideoCreateRequest) (KlingTaskResponse, error) {
	var out KlingTaskResponse
	if err := c.ensureConfigured(); err != nil {
		return out, err
	}
	return out, c.doJSON(ctx, http.MethodPost, "/v1/videos/image2video", requestID, req, &out)
}

func (c *KlingClient) GetImageToVideoTask(ctx context.Context, requestID, taskID string) (KlingTaskResponse, error) {
	var out KlingTaskResponse
	if err := c.ensureConfigured(); err != nil {
		return out, err
	}
	return out, c.doJSON(ctx, http.MethodGet, "/v1/videos/image2video/"+taskID, requestID, nil, &out)
}

func (c *KlingClient) ensureConfigured() error {
	if c.baseURL == "" || c.accessKey == "" || c.secretKey == "" {
		return providerNotConfigured("kling credentials are not configured")
	}
	return nil
}

func (c *KlingClient) doJSON(ctx context.Context, method, path, requestID string, body any, out any) error {
	token, err := makeKlingToken(c.accessKey, c.secretKey, time.Now())
	if err != nil {
		return err
	}
	req := c.client.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+token).
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
		return providerHTTPError("kling provider", resp)
	}
	if len(resp.Body()) == 0 {
		return nil
	}
	if err := json.Unmarshal(resp.Body(), out); err != nil {
		return ProviderError{Status: http.StatusBadGateway, Code: "provider_bad_response", Message: "invalid kling provider response"}
	}
	return nil
}

func makeKlingToken(accessKey, secretKey string, now time.Time) (string, error) {
	header, err := base64URLJSON(map[string]string{"alg": "HS256", "typ": "JWT"})
	if err != nil {
		return "", err
	}
	claims, err := base64URLJSON(map[string]any{
		"iss": accessKey,
		"exp": now.Unix() + 1800,
		"nbf": now.Unix() - 5,
	})
	if err != nil {
		return "", err
	}
	unsigned := header + "." + claims
	mac := hmac.New(sha256.New, []byte(secretKey))
	_, _ = mac.Write([]byte(unsigned))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return unsigned + "." + signature, nil
}

func base64URLJSON(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
