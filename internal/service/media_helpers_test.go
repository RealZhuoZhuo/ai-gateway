package service

import (
	"testing"

	"github.com/RealZhuoZhuo/ai-gateway/internal/config"
)

func TestMediaProviderForModel(t *testing.T) {
	imageRoutes := newModelRouteGroup([]config.ModelProvider{
		{Model: "ep-abc", Provider: "ark"},
		{Model: "wan2.6-image", Provider: "dashscope"},
		{Model: "kling-v2-6", Provider: "kling"},
	})
	videoRoutes := newModelRouteGroup([]config.ModelProvider{
		{Model: "seedance", Provider: "ark"},
		{Model: "wan2.7-t2v-2026-04-25", Provider: "dashscope"},
		{Model: "wan2.7-i2v-2026-04-25", Provider: "dashscope"},
		{Model: "kling-v2-6", Provider: "kling"},
	})
	tests := []struct {
		name  string
		model string
		want  string
	}{
		{name: "dashscope image", model: "wan2.6-image", want: "dashscope"},
		{name: "kling", model: "kling-v2-6", want: "kling"},
		{name: "ark", model: "ep-abc", want: "ark"},
		{name: "unconfigured", model: "missing-model", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := imageRoutes.providerForModel(tt.model); got != tt.want {
				t.Fatalf("provider = %q, want %q", got, tt.want)
			}
		})
	}

	if got := imageRoutes.providerForModel("wan2.7-t2v-2026-04-25"); got != "" {
		t.Fatalf("video model in image routes = %q, want empty", got)
	}
	if got := videoRoutes.providerForModel("wan2.7-t2v-2026-04-25"); got != "dashscope" {
		t.Fatalf("video provider = %q, want dashscope", got)
	}
}

func TestDashScopeVideoMode(t *testing.T) {
	if got := dashScopeVideoMode(CreateVideoTaskRequest{Model: "wan2.7-t2v-2026-04-25"}, ""); got != videoModeTextToVideo {
		t.Fatalf("text mode = %q, want %q", got, videoModeTextToVideo)
	}
	if got := dashScopeVideoMode(CreateVideoTaskRequest{Model: "wan2.7-i2v-2026-04-25"}, ""); got != videoModeImageToVideo {
		t.Fatalf("image mode = %q, want %q", got, videoModeImageToVideo)
	}
	if got := dashScopeVideoMode(CreateVideoTaskRequest{Model: "custom-wan"}, "https://example.com/image.png"); got != videoModeImageToVideo {
		t.Fatalf("image input mode = %q, want %q", got, videoModeImageToVideo)
	}
}

func TestModelRouteGroupExactMatch(t *testing.T) {
	routes := newModelRouteGroup([]config.ModelProvider{
		{Model: "kling-v2-6", Provider: "kling"},
	})

	if got := routes.providerForModel("kling-v2-6"); got != "kling" {
		t.Fatalf("provider = %q, want kling", got)
	}
	if got := routes.providerForModel("kling-v2-1"); got != "" {
		t.Fatalf("unconfigured provider = %q, want empty", got)
	}
}

func TestTaskIDRoundTrip(t *testing.T) {
	taskID := encodeTaskID(taskPrefixDashImageVideo, "abc_123")
	prefix, raw, ok := decodeTaskID(taskID)
	if !ok {
		t.Fatal("decodeTaskID returned ok=false")
	}
	if prefix != taskPrefixDashImageVideo {
		t.Fatalf("prefix = %q, want %q", prefix, taskPrefixDashImageVideo)
	}
	if raw != "abc_123" {
		t.Fatalf("raw = %q, want abc_123", raw)
	}
}
