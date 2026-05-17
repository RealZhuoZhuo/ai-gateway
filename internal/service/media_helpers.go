package service

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/RealZhuoZhuo/ai-gateway/internal/common"
	"github.com/RealZhuoZhuo/ai-gateway/internal/config"
	"github.com/RealZhuoZhuo/ai-gateway/internal/providers"
)

const (
	videoModeTextToVideo  = "text_to_video"
	videoModeImageToVideo = "image_to_video"
)

type modelRouteGroup struct {
	routes []modelRoute
}

type modelRoute struct {
	model    string
	provider string
}

func newModelRouteGroup(providers []config.ModelProvider) modelRouteGroup {
	routes := make([]modelRoute, 0, len(providers))
	for _, route := range providers {
		routes = append(routes, modelRoute{
			model:    strings.TrimSpace(route.Model),
			provider: normalizeProvider(route.Provider),
		})
	}
	return modelRouteGroup{routes: routes}
}

func (g modelRouteGroup) providerForModel(model string) string {
	return g.routeForModel(model).provider
}

func (g modelRouteGroup) modelForProvider(provider string) string {
	provider = normalizeProvider(provider)
	for _, route := range g.routes {
		if route.model == "" || route.provider != provider {
			continue
		}
		return route.model
	}
	return ""
}

func (g modelRouteGroup) modelForProviderMode(provider, mode string) string {
	provider = normalizeProvider(provider)
	for _, route := range g.routes {
		if route.model == "" || route.provider != provider || videoModeForModel(route.model) != mode {
			continue
		}
		return route.model
	}
	return ""
}

func (g modelRouteGroup) routeForModel(model string) modelRoute {
	model = strings.TrimSpace(model)
	for _, route := range g.routes {
		if route.model == model {
			return route
		}
	}
	return modelRoute{}
}

func normalizeProvider(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}

func imagePrompt(in ImageGenerationRequest) string {
	return firstString(in.Prompt, nestedString(in.Input, "prompt"), messagesText(in.Input))
}

func videoPrompt(in CreateVideoTaskRequest) string {
	return firstString(in.Prompt, nestedString(in.Input, "prompt"), messagesText(in.Input))
}

func firstImage(in ImageGenerationRequest) string {
	if in.Image != "" {
		return in.Image
	}
	if len(in.ReferenceImages) > 0 {
		return in.ReferenceImages[0].URL
	}
	return firstString(nestedString(in.Input, "image"), messagesImage(in.Input))
}

func firstFrame(in CreateVideoTaskRequest) string {
	return firstString(in.FirstFrameURL, in.Image, nestedString(in.Input, "first_frame_url"), nestedString(in.Input, "image"), nestedString(in.Input, "img_url"))
}

func dashScopeVideoMode(in CreateVideoTaskRequest, image string) string {
	if mode := videoModeForModel(in.Model); mode != "" {
		return mode
	}
	if image != "" || in.ImageTail != "" {
		return videoModeImageToVideo
	}
	return videoModeTextToVideo
}

func videoModeForModel(model string) string {
	model = strings.ToLower(model)
	if strings.Contains(model, "i2v") || strings.Contains(model, "image-to-video") || strings.Contains(model, "image2video") {
		return videoModeImageToVideo
	}
	if strings.Contains(model, "t2v") || strings.Contains(model, "text-to-video") || strings.Contains(model, "text2video") {
		return videoModeTextToVideo
	}
	return ""
}

func videoAspectRatio(in CreateVideoTaskRequest) string {
	return firstString(in.AspectRatio, in.Ratio, nestedString(in.Parameters, "aspect_ratio"), nestedString(in.Parameters, "ratio"))
}

func imageAspectRatio(in ImageGenerationRequest) string {
	return firstString(in.AspectRatio, nestedString(in.Parameters, "aspect_ratio"))
}

func imageResolution(in ImageGenerationRequest) string {
	return firstString(in.Resolution, in.Size, nestedString(in.Parameters, "resolution"), nestedString(in.Parameters, "size"))
}

func klingImageResolution(in ImageGenerationRequest) string {
	return firstString(in.Resolution, in.Size, nestedString(in.Parameters, "resolution"), nestedString(in.Parameters, "size"))
}

func videoDurationString(in CreateVideoTaskRequest) string {
	if in.Duration != nil {
		return strconv.Itoa(*in.Duration)
	}
	return firstString(nestedString(in.Parameters, "duration"), nestedString(in.Input, "duration"))
}

func firstString(values ...string) string {
	return common.FirstNonEmpty(values...)
}

func nestedString(values map[string]any, keys ...string) string {
	if len(values) == 0 {
		return ""
	}
	for _, key := range keys {
		value, ok := values[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case string:
			return strings.TrimSpace(typed)
		case fmt.Stringer:
			return strings.TrimSpace(typed.String())
		case float64:
			if typed == float64(int64(typed)) {
				return strconv.FormatInt(int64(typed), 10)
			}
			return strconv.FormatFloat(typed, 'f', -1, 64)
		case int:
			return strconv.Itoa(typed)
		case int64:
			return strconv.FormatInt(typed, 10)
		}
	}
	return ""
}

func messagesText(input map[string]any) string {
	for _, item := range messageContents(input) {
		if text := nestedString(item, "text"); text != "" {
			return text
		}
	}
	return ""
}

func messagesImage(input map[string]any) string {
	for _, item := range messageContents(input) {
		if image := nestedString(item, "image", "image_url", "url"); image != "" {
			return image
		}
	}
	return ""
}

func messageContents(input map[string]any) []map[string]any {
	rawMessages, ok := input["messages"].([]any)
	if !ok {
		return nil
	}
	var out []map[string]any
	for _, rawMessage := range rawMessages {
		message, ok := rawMessage.(map[string]any)
		if !ok {
			continue
		}
		rawContent, ok := message["content"].([]any)
		if !ok {
			continue
		}
		for _, rawItem := range rawContent {
			item, ok := rawItem.(map[string]any)
			if ok {
				out = append(out, item)
			}
		}
	}
	return out
}

func dashScopeMessages(input map[string]any, prompt, image string) []providers.DashScopeMessage {
	rawMessages, ok := input["messages"].([]any)
	if ok {
		messages := make([]providers.DashScopeMessage, 0, len(rawMessages))
		for _, rawMessage := range rawMessages {
			message, ok := rawMessage.(map[string]any)
			if !ok {
				continue
			}
			role := common.DefaultString(nestedString(message, "role"), "user")
			rawContent, ok := message["content"].([]any)
			if !ok {
				continue
			}
			content := make([]providers.DashScopeContent, 0, len(rawContent))
			for _, rawItem := range rawContent {
				item, ok := rawItem.(map[string]any)
				if !ok {
					continue
				}
				content = append(content, providers.DashScopeContent{
					Text:  nestedString(item, "text"),
					Image: nestedString(item, "image", "image_url", "url"),
				})
			}
			if len(content) > 0 {
				messages = append(messages, providers.DashScopeMessage{Role: role, Content: content})
			}
		}
		if len(messages) > 0 {
			return messages
		}
	}

	content := []providers.DashScopeContent{{Text: prompt}}
	if image != "" {
		content = append(content, providers.DashScopeContent{Image: image})
	}
	return []providers.DashScopeMessage{{Role: "user", Content: content}}
}

func cloneMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func setIfMissing(values map[string]any, key string, value any) map[string]any {
	if value == nil {
		return values
	}
	if s, ok := value.(string); ok && strings.TrimSpace(s) == "" {
		return values
	}
	if values == nil {
		values = map[string]any{}
	}
	if _, exists := values[key]; !exists {
		values[key] = value
	}
	return values
}

func normalizeDashScopeStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "PENDING":
		return "queued"
	case "RUNNING":
		return "running"
	case "SUCCEEDED":
		return "succeeded"
	case "FAILED":
		return "failed"
	case "CANCELED", "CANCELLED":
		return "cancelled"
	default:
		return normalizeTaskStatus(status)
	}
}

func normalizeKlingStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "submitted":
		return "queued"
	case "processing":
		return "running"
	case "succeed":
		return "succeeded"
	default:
		return normalizeTaskStatus(status)
	}
}

func dashScopeTaskError(out providers.DashScopeTaskOutput) *TaskError {
	if out.Code == "" && out.Message == "" {
		return nil
	}
	return &TaskError{Code: common.DefaultString(out.Code, "provider_error"), Message: out.Message}
}

func dashScopeImageURLs(out providers.DashScopeImageResponse) []string {
	var urls []string
	for _, choice := range out.Output.Choices {
		for _, item := range choice.Message.Content {
			if item.Image != "" {
				urls = append(urls, item.Image)
			}
		}
	}
	return urls
}

func dashScopeTaskImageURLs(out providers.DashScopeTaskOutput) []string {
	var urls []string
	for _, result := range out.Results {
		if result.URL != "" {
			urls = append(urls, result.URL)
		}
		if result.VideoURL != "" {
			urls = append(urls, result.VideoURL)
		}
	}
	for _, choice := range out.Choices {
		for _, item := range choice.Message.Content {
			if item.Image != "" {
				urls = append(urls, item.Image)
			}
		}
	}
	if out.URL != "" {
		urls = append(urls, out.URL)
	}
	return urls
}

func dashScopeTaskVideoURL(out providers.DashScopeTaskOutput) string {
	if out.VideoURL != "" {
		return out.VideoURL
	}
	if out.URL != "" {
		return out.URL
	}
	for _, result := range out.Results {
		if result.VideoURL != "" {
			return result.VideoURL
		}
		if result.URL != "" {
			return result.URL
		}
	}
	return ""
}

func klingImageURLs(out providers.KlingTaskResponse) []string {
	if out.Data.TaskResult == nil {
		return nil
	}
	urls := make([]string, 0, len(out.Data.TaskResult.Images))
	for _, image := range out.Data.TaskResult.Images {
		if image.URL != "" {
			urls = append(urls, image.URL)
		}
	}
	return urls
}

func klingVideoURL(out providers.KlingTaskResponse) string {
	if out.Data.TaskResult == nil {
		return ""
	}
	for _, video := range out.Data.TaskResult.Videos {
		if video.URL != "" {
			return video.URL
		}
	}
	return ""
}

func waitDuration(attempt int) time.Duration {
	if attempt < 2 {
		return 500 * time.Millisecond
	}
	return 1500 * time.Millisecond
}

func timeAfter(d time.Duration) <-chan time.Time {
	return time.After(d)
}
