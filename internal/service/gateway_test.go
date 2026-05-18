package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/go-resty/resty/v2"

	"github.com/RealZhuoZhuo/ai-gateway/internal/config"
	"github.com/RealZhuoZhuo/ai-gateway/internal/providers"
)

func TestGetArkVideoTaskUsesTaskEndpointAndContentVideoURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v3/contents/generations/tasks/task-1" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer ark-key" {
			t.Fatalf("authorization = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     "task-1",
			"model":  "doubao-seedance-1-5-pro-251215",
			"status": "succeeded",
			"content": map[string]any{
				"video_url": "https://example.com/video.mp4",
			},
		})
	}))
	defer server.Close()

	gateway := NewGateway(config.Config{}, providers.NewArkClient(
		resty.New(),
		"",
		"",
		server.URL+"/api/v3/contents/generations/tasks/",
		"ark-key",
	), nil)

	out, err := gateway.GetVideoTask(context.Background(), "request-1", "ark_task-1")
	if err != nil {
		t.Fatalf("GetVideoTask returned error: %v", err)
	}
	if out.Status != "succeeded" {
		t.Fatalf("status = %q, want succeeded", out.Status)
	}
	if out.VideoURL == nil || *out.VideoURL != "https://example.com/video.mp4" {
		t.Fatalf("video_url = %v, want content.video_url", out.VideoURL)
	}
}

func TestGetArkVideoTaskDoesNotExposeVideoURLBeforeSucceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     "task-1",
			"status": "queued",
			"content": map[string]any{
				"video_url": "https://example.com/video.mp4",
			},
		})
	}))
	defer server.Close()

	gateway := NewGateway(config.Config{}, providers.NewArkClient(
		resty.New(),
		"",
		"",
		server.URL+"/api/v3/contents/generations/tasks",
		"ark-key",
	), nil)

	out, err := gateway.GetVideoTask(context.Background(), "request-1", "ark_task-1")
	if err != nil {
		t.Fatalf("GetVideoTask returned error: %v", err)
	}
	if out.Status != "queued" {
		t.Fatalf("status = %q, want queued", out.Status)
	}
	if out.VideoURL != nil {
		t.Fatalf("video_url = %v, want nil before succeeded", *out.VideoURL)
	}
}

func TestGenerateDashScopeImageUsesWan27MultimodalMessages(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/services/aigc/multimodal-generation/generation" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer dash-key" {
			t.Fatalf("authorization = %q", got)
		}
		if got := r.Header.Get("X-DashScope-Async"); got != "" {
			t.Fatalf("X-DashScope-Async = %q, want empty", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"request_id": "dash-request",
			"output": map[string]any{
				"choices": []any{
					map[string]any{
						"message": map[string]any{
							"content": []any{
								map[string]any{"image": "https://example.com/out.png"},
							},
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	gateway := NewGateway(config.Config{
		ImageModelProviders: []config.ModelProvider{{Model: "wan2.7-image-pro", Provider: "dashscope"}},
	}, nil, providers.NewDashScopeClient(resty.New(), server.URL+"/api/v1", "dash-key"))

	out, err := gateway.GenerateImage(context.Background(), "request-1", ImageGenerationRequest{
		Model:  "wan2.7-image-pro",
		Prompt: "把图2的涂鸦喷绘在图1的汽车上",
		Images: []string{"https://example.com/car.webp", "https://example.com/paint.webp"},
		Size:   "2K",
	})
	if err != nil {
		t.Fatalf("GenerateImage returned error: %v", err)
	}
	if out.URL != "https://example.com/out.png" {
		t.Fatalf("url = %q, want provider image URL", out.URL)
	}

	if got := body["model"]; got != "wan2.7-image-pro" {
		t.Fatalf("model = %v", got)
	}
	input := body["input"].(map[string]any)
	messages := input["messages"].([]any)
	content := messages[0].(map[string]any)["content"].([]any)
	gotContent := []map[string]string{}
	for _, raw := range content {
		item := raw.(map[string]any)
		gotItem := map[string]string{}
		if text := stringValue(item["text"]); text != "" {
			gotItem["text"] = text
		}
		if image := stringValue(item["image"]); image != "" {
			gotItem["image"] = image
		}
		gotContent = append(gotContent, gotItem)
	}
	wantContent := []map[string]string{
		{"text": "把图2的涂鸦喷绘在图1的汽车上"},
		{"image": "https://example.com/car.webp"},
		{"image": "https://example.com/paint.webp"},
	}
	if !reflect.DeepEqual(gotContent, wantContent) {
		t.Fatalf("content = %#v, want %#v", gotContent, wantContent)
	}
	params := body["parameters"].(map[string]any)
	if params["size"] != "2K" {
		t.Fatalf("size = %v, want 2K", params["size"])
	}
	if params["n"] != float64(1) {
		t.Fatalf("n = %v, want 1", params["n"])
	}
	if params["watermark"] != false {
		t.Fatalf("watermark = %v, want false", params["watermark"])
	}
}

func TestGenerateDashScopeImageAsyncCreatesTask(t *testing.T) {
	var path string
	var asyncHeader string
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		asyncHeader = r.Header.Get("X-DashScope-Async")
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"request_id": "dash-request",
			"output": map[string]any{
				"task_id":     "task-1",
				"task_status": "PENDING",
			},
		})
	}))
	defer server.Close()

	async := true
	gateway := NewGateway(config.Config{
		ImageModelProviders: []config.ModelProvider{{Model: "wan2.7-image-pro", Provider: "dashscope"}},
	}, nil, providers.NewDashScopeClient(resty.New(), server.URL+"/api/v1", "dash-key"))

	out, err := gateway.GenerateImage(context.Background(), "request-1", ImageGenerationRequest{
		Model:  "wan2.7-image-pro",
		Prompt: "一间有着精致窗户的花店",
		Async:  &async,
	})
	if err != nil {
		t.Fatalf("GenerateImage returned error: %v", err)
	}
	if path != "/api/v1/services/aigc/image-generation/generation" {
		t.Fatalf("path = %s", path)
	}
	if asyncHeader != "enable" {
		t.Fatalf("X-DashScope-Async = %q, want enable", asyncHeader)
	}
	if out.TaskID != "dashscope-img_task-1" {
		t.Fatalf("task_id = %q, want dashscope-img_task-1", out.TaskID)
	}
	if out.Status != "queued" {
		t.Fatalf("status = %q, want queued", out.Status)
	}
	params := body["parameters"].(map[string]any)
	if params["thinking_mode"] != true {
		t.Fatalf("thinking_mode = %v, want true", params["thinking_mode"])
	}
}

func TestCreateDashScopeR2VTaskUsesVideoSynthesisMedia(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/services/aigc/video-generation/video-synthesis" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("X-DashScope-Async"); got != "enable" {
			t.Fatalf("X-DashScope-Async = %q, want enable", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"request_id": "dash-request",
			"output": map[string]any{
				"task_id":     "video-task-1",
				"task_status": "RUNNING",
			},
		})
	}))
	defer server.Close()

	duration := 10
	gateway := NewGateway(config.Config{
		VideoModelProviders: []config.ModelProvider{{Model: "wan2.7-r2v", Provider: "dashscope"}},
	}, nil, providers.NewDashScopeClient(resty.New(), server.URL+"/api/v1", "dash-key"))

	out, err := gateway.CreateVideoTask(context.Background(), "request-1", CreateVideoTaskRequest{
		Model:      "wan2.7-r2v",
		Prompt:     "视频1抱着图3，在图4的椅子上弹奏",
		Resolution: "720P",
		Ratio:      "16:9",
		Duration:   &duration,
		Media: []VideoMedia{
			{Type: "reference_image", URL: "https://example.com/girl.jpg", ReferenceVoice: "https://example.com/girl.mp3"},
			{Type: "reference_video", URL: "https://example.com/role.mp4", ReferenceVoice: "https://example.com/boy.mp3"},
		},
	})
	if err != nil {
		t.Fatalf("CreateVideoTask returned error: %v", err)
	}
	if out.TaskID != "dashscope-r2v_video-task-1" {
		t.Fatalf("task_id = %q, want dashscope-r2v_video-task-1", out.TaskID)
	}

	input := body["input"].(map[string]any)
	if input["prompt"] != "视频1抱着图3，在图4的椅子上弹奏" {
		t.Fatalf("prompt = %v", input["prompt"])
	}
	media := input["media"].([]any)
	first := media[0].(map[string]any)
	if first["type"] != "reference_image" || first["url"] != "https://example.com/girl.jpg" || first["reference_voice"] != "https://example.com/girl.mp3" {
		t.Fatalf("first media = %#v", first)
	}
	second := media[1].(map[string]any)
	if second["type"] != "reference_video" || second["url"] != "https://example.com/role.mp4" || second["reference_voice"] != "https://example.com/boy.mp3" {
		t.Fatalf("second media = %#v", second)
	}
	params := body["parameters"].(map[string]any)
	if params["resolution"] != "720P" {
		t.Fatalf("resolution = %v, want 720P", params["resolution"])
	}
	if params["ratio"] != "16:9" {
		t.Fatalf("ratio = %v, want 16:9", params["ratio"])
	}
	if params["duration"] != float64(10) {
		t.Fatalf("duration = %v, want 10", params["duration"])
	}
	if params["watermark"] != false {
		t.Fatalf("watermark = %v, want false", params["watermark"])
	}
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	return value.(string)
}
