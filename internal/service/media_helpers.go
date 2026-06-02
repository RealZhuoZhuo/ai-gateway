package service

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/RealZhuoZhuo/ai-gateway/internal/common"
	"github.com/RealZhuoZhuo/ai-gateway/internal/config"
	"github.com/RealZhuoZhuo/ai-gateway/internal/providers"
)

const (
	videoModeTextToVideo      = "text_to_video"
	videoModeImageToVideo     = "image_to_video"
	videoModeReferenceToVideo = "reference_to_video"
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
	return firstString(in.Prompt, videoContentText(in.Content), rawVideoContentText(in.Input), nestedString(in.Input, "prompt"), messagesText(in.Input))
}

func imageReferences(in ImageGenerationRequest) ([]string, error) {
	images := make([]string, 0, len(in.Images)+len(in.ReferenceImages)+1)
	add := func(value string) {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			images = append(images, trimmed)
		}
	}
	for _, image := range in.Images {
		add(image)
	}
	add(in.Image)
	for _, item := range in.ReferenceImages {
		if strings.TrimSpace(item.URL) == "" {
			return nil, invalidRequest("reference_images.url is required")
		}
		add(item.URL)
	}
	for _, image := range messagesImages(in.Input) {
		add(image)
	}
	return uniqueStrings(images), nil
}

func arkImageSize(in ImageGenerationRequest) string {
	return firstString(in.Size, in.Resolution, nestedString(in.Parameters, "size"), nestedString(in.Parameters, "resolution"))
}

func arkImageReferences(in ImageGenerationRequest) (any, error) {
	images, err := imageReferences(in)
	if err != nil {
		return nil, err
	}
	switch len(images) {
	case 0:
		return nil, nil
	case 1:
		return images[0], nil
	default:
		return images, nil
	}
}

func arkSequentialImageGeneration(in ImageGenerationRequest) string {
	return common.DefaultString(firstString(in.SequentialImageGeneration, nestedString(in.Parameters, "sequential_image_generation")), "disabled")
}

func arkResponseFormat(in ImageGenerationRequest) string {
	return common.DefaultString(firstString(in.ResponseFormat, nestedString(in.Parameters, "response_format")), "url")
}

func arkImageStream(in ImageGenerationRequest) bool {
	if in.Stream != nil {
		return *in.Stream
	}
	if value, ok := nestedBool(in.Parameters, "stream"); ok {
		return value
	}
	return false
}

func arkImageWatermark(in ImageGenerationRequest) bool {
	if in.Watermark != nil {
		return *in.Watermark
	}
	if value, ok := nestedBool(in.Parameters, "watermark"); ok {
		return value
	}
	return true
}

func arkImageURLs(out providers.ArkImageResponse) []string {
	urls := make([]string, 0, len(out.Data)+1)
	if out.URL != "" {
		urls = append(urls, out.URL)
	}
	for _, item := range out.Data {
		if item.URL != "" {
			urls = append(urls, item.URL)
		}
	}
	return urls
}

func arkImageMetadata(out providers.ArkImageResponse) map[string]any {
	metadata := map[string]any{}
	if out.Created != 0 {
		metadata["created"] = out.Created
	}
	if out.Usage != nil {
		metadata["usage"] = out.Usage
	}
	sizes := make([]string, 0, len(out.Data))
	for _, item := range out.Data {
		if item.Size != "" {
			sizes = append(sizes, item.Size)
		}
	}
	if len(sizes) > 0 {
		metadata["sizes"] = sizes
	}
	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

func firstFrame(in CreateVideoTaskRequest) string {
	return firstString(in.FirstFrameURL, in.Image, nestedString(in.Input, "first_frame_url"), nestedString(in.Input, "image"), nestedString(in.Input, "img_url"))
}

func dashScopeImageParameters(in ImageGenerationRequest, textOnly bool) map[string]any {
	params := cloneMap(in.Parameters)
	params = setIfMissing(params, "negative_prompt", in.NegativePrompt)
	params = setIfMissing(params, "size", imageResolution(in))
	params = setIfMissing(params, "n", imageN(in, 1))
	params = setIfMissing(params, "aspect_ratio", imageAspectRatio(in))
	params = setIfMissing(params, "watermark", imageWatermark(in, false))
	if textOnly {
		params = setIfMissing(params, "thinking_mode", dashScopeThinkingMode(in))
	}
	return params
}

func imageWatermark(in ImageGenerationRequest, fallback bool) bool {
	if in.Watermark != nil {
		return *in.Watermark
	}
	if value, ok := nestedBool(in.Parameters, "watermark"); ok {
		return value
	}
	return fallback
}

func dashScopeThinkingMode(in ImageGenerationRequest) any {
	if value, ok := nestedValue(in.Parameters, "thinking_mode"); ok {
		return value
	}
	return true
}

func imageN(in ImageGenerationRequest, fallback int) int {
	if in.N != nil {
		return *in.N
	}
	return fallback
}

func arkVideoContent(in CreateVideoTaskRequest, prompt string) ([]providers.ArkVideoContent, error) {
	if rawContent, ok := in.Input["content"].([]any); ok {
		return arkVideoContentFromRaw(rawContent)
	}
	if len(in.Content) > 0 {
		return arkVideoContentFromInput(in.Content)
	}

	content := []providers.ArkVideoContent{{Type: "text", Text: prompt}}
	if firstFrameURL := firstFrame(in); firstFrameURL != "" {
		content = append(content, providers.ArkVideoContent{
			Type:     "image_url",
			ImageURL: &providers.ArkVideoImageURL{URL: firstFrameURL},
		})
	}
	return content, nil
}

func arkVideoContentFromRaw(rawContent []any) ([]providers.ArkVideoContent, error) {
	content := make([]providers.ArkVideoContent, 0, len(rawContent))
	for _, rawItem := range rawContent {
		item, ok := rawItem.(map[string]any)
		if !ok {
			continue
		}
		itemType := nestedString(item, "type")
		if itemType == "" {
			return nil, invalidRequest("input.content.type is required")
		}
		out := providers.ArkVideoContent{
			Type: itemType,
			Text: nestedString(item, "text"),
			Role: nestedString(item, "role"),
		}
		if url := nestedMediaURL(item, "image_url"); url != "" {
			out.ImageURL = &providers.ArkVideoImageURL{URL: url}
		}
		if url := nestedMediaURL(item, "video_url"); url != "" {
			out.VideoURL = &providers.ArkVideoMediaURL{URL: url}
		}
		if url := nestedMediaURL(item, "audio_url"); url != "" {
			out.AudioURL = &providers.ArkVideoMediaURL{URL: url}
		}
		content = append(content, out)
	}
	return content, nil
}

func arkVideoContentFromInput(items []VideoContent) ([]providers.ArkVideoContent, error) {
	content := make([]providers.ArkVideoContent, 0, len(items))
	for _, item := range items {
		itemType := strings.TrimSpace(item.Type)
		if itemType == "" {
			return nil, invalidRequest("content.type is required")
		}
		out := providers.ArkVideoContent{
			Type: itemType,
			Text: strings.TrimSpace(item.Text),
			Role: strings.TrimSpace(item.Role),
		}
		if item.ImageURL != nil {
			if strings.TrimSpace(item.ImageURL.URL) == "" {
				return nil, invalidRequest("content.image_url.url is required")
			}
			out.ImageURL = &providers.ArkVideoImageURL{URL: strings.TrimSpace(item.ImageURL.URL)}
		}
		if item.VideoURL != nil {
			if strings.TrimSpace(item.VideoURL.URL) == "" {
				return nil, invalidRequest("content.video_url.url is required")
			}
			out.VideoURL = &providers.ArkVideoMediaURL{URL: strings.TrimSpace(item.VideoURL.URL)}
		}
		if item.AudioURL != nil {
			if strings.TrimSpace(item.AudioURL.URL) == "" {
				return nil, invalidRequest("content.audio_url.url is required")
			}
			out.AudioURL = &providers.ArkVideoMediaURL{URL: strings.TrimSpace(item.AudioURL.URL)}
		}
		content = append(content, out)
	}
	return content, nil
}

func videoContentText(items []VideoContent) string {
	for _, item := range items {
		if strings.TrimSpace(item.Type) == "text" && strings.TrimSpace(item.Text) != "" {
			return strings.TrimSpace(item.Text)
		}
	}
	return ""
}

func rawVideoContentText(input map[string]any) string {
	rawContent, ok := input["content"].([]any)
	if !ok {
		return ""
	}
	for _, rawItem := range rawContent {
		item, ok := rawItem.(map[string]any)
		if !ok {
			continue
		}
		if nestedString(item, "type") == "text" {
			return nestedString(item, "text")
		}
	}
	return ""
}

func arkVideoGenerateAudio(in CreateVideoTaskRequest) *bool {
	if in.GenerateAudio != nil {
		return in.GenerateAudio
	}
	if value, ok := nestedBool(in.Parameters, "generate_audio"); ok {
		return &value
	}
	return nil
}

func arkVideoWatermark(in CreateVideoTaskRequest) *bool {
	if in.Watermark != nil {
		return in.Watermark
	}
	if value, ok := nestedBool(in.Parameters, "watermark"); ok {
		return &value
	}
	return nil
}

func arkVideoDuration(in CreateVideoTaskRequest) *int {
	if in.Duration != nil {
		return in.Duration
	}
	if value, ok := nestedInt(in.Parameters, "duration"); ok {
		return &value
	}
	return nil
}

func arkVideoSeed(in CreateVideoTaskRequest) *int64 {
	if in.Seed != nil {
		return in.Seed
	}
	if value, ok := nestedInt64(in.Parameters, "seed"); ok {
		return &value
	}
	return nil
}

func arkVideoRatio(in CreateVideoTaskRequest) string {
	return firstString(in.Ratio, in.AspectRatio, nestedString(in.Parameters, "ratio"), nestedString(in.Parameters, "aspect_ratio"))
}

func arkVideoResolution(in CreateVideoTaskRequest) string {
	return firstString(in.Resolution, nestedString(in.Parameters, "resolution"))
}

func dashScopeVideoMode(in CreateVideoTaskRequest, media []providers.DashScopeVideoMedia) string {
	if mode := videoModeForModel(in.Model); mode != "" {
		return mode
	}
	if len(media) > 0 {
		return videoModeReferenceToVideo
	}
	if firstFrame(in) != "" || in.ImageTail != "" {
		return videoModeImageToVideo
	}
	return videoModeTextToVideo
}

func videoModeForModel(model string) string {
	model = strings.ToLower(model)
	if strings.Contains(model, "r2v") || strings.Contains(model, "reference-to-video") || strings.Contains(model, "reference2video") {
		return videoModeReferenceToVideo
	}
	if strings.Contains(model, "i2v") || strings.Contains(model, "image-to-video") || strings.Contains(model, "image2video") {
		return videoModeImageToVideo
	}
	if strings.Contains(model, "t2v") || strings.Contains(model, "text-to-video") || strings.Contains(model, "text2video") {
		return videoModeTextToVideo
	}
	return ""
}

func dashScopeVideoInput(in CreateVideoTaskRequest, prompt string) (providers.DashScopeVideoInput, error) {
	media, err := dashScopeVideoMedia(in)
	if err != nil {
		return providers.DashScopeVideoInput{}, err
	}
	out := providers.DashScopeVideoInput{
		Prompt: firstString(nestedString(in.Input, "prompt"), prompt),
		Media:  media,
	}
	if out.Prompt == "" {
		out.Prompt = prompt
	}
	return out, nil
}

func dashScopeVideoMedia(in CreateVideoTaskRequest) ([]providers.DashScopeVideoMedia, error) {
	if rawMedia, ok := in.Input["media"].([]any); ok {
		return dashScopeVideoMediaFromRaw(rawMedia)
	}
	if len(in.Media) > 0 {
		return dashScopeVideoMediaFromRequest(in.Media)
	}
	image := firstFrame(in)
	if image == "" {
		return nil, nil
	}
	return []providers.DashScopeVideoMedia{{
		Type: "reference_image",
		URL:  image,
	}}, nil
}

func dashScopeVideoMediaFromRaw(rawMedia []any) ([]providers.DashScopeVideoMedia, error) {
	media := make([]providers.DashScopeVideoMedia, 0, len(rawMedia))
	for _, rawItem := range rawMedia {
		item, ok := rawItem.(map[string]any)
		if !ok {
			continue
		}
		out := providers.DashScopeVideoMedia{
			Type:           nestedString(item, "type"),
			URL:            nestedString(item, "url"),
			ReferenceVoice: nestedString(item, "reference_voice"),
		}
		if out.Type == "" {
			out.Type = dashScopeMediaType(out.URL)
		}
		if out.URL == "" {
			return nil, invalidRequest("input.media.url is required")
		}
		media = append(media, out)
	}
	return media, nil
}

func dashScopeVideoMediaFromRequest(items []VideoMedia) ([]providers.DashScopeVideoMedia, error) {
	media := make([]providers.DashScopeVideoMedia, 0, len(items))
	for _, item := range items {
		out := providers.DashScopeVideoMedia{
			Type:           strings.TrimSpace(item.Type),
			URL:            strings.TrimSpace(item.URL),
			ReferenceVoice: strings.TrimSpace(item.ReferenceVoice),
		}
		if out.Type == "" {
			out.Type = dashScopeMediaType(out.URL)
		}
		if out.URL == "" {
			return nil, invalidRequest("media.url is required")
		}
		media = append(media, out)
	}
	return media, nil
}

func dashScopeMediaType(url string) string {
	lower := strings.ToLower(strings.TrimSpace(url))
	switch {
	case strings.HasSuffix(lower, ".mp4"), strings.HasSuffix(lower, ".mov"), strings.HasSuffix(lower, ".webm"), strings.HasSuffix(lower, ".m4v"):
		return "reference_video"
	default:
		return "reference_image"
	}
}

func dashScopeVideoParameters(in CreateVideoTaskRequest) map[string]any {
	params := cloneMap(in.Parameters)
	params = setIfMissing(params, "negative_prompt", in.NegativePrompt)
	params = setIfMissing(params, "duration", in.Duration)
	params = setIfMissing(params, "resolution", in.Resolution)
	params = setIfMissing(params, "ratio", videoAspectRatio(in))
	params = setIfMissing(params, "seed", in.Seed)
	params = setIfMissing(params, "prompt_extend", dashScopePromptExtend(in))
	params = setIfMissing(params, "watermark", videoWatermark(in, false))
	return params
}

func dashScopePromptExtend(in CreateVideoTaskRequest) any {
	if value, ok := nestedValue(in.Parameters, "prompt_extend"); ok {
		return value
	}
	return nil
}

func videoWatermark(in CreateVideoTaskRequest, fallback bool) bool {
	if value, ok := nestedBool(in.Parameters, "watermark"); ok {
		return value
	}
	return fallback
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

func imageQuality(in ImageGenerationRequest) string {
	return firstString(in.Quality, nestedString(in.Parameters, "quality"))
}

func imageFormat(in ImageGenerationRequest) string {
	return firstString(in.Format, nestedString(in.Parameters, "format"))
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

func nestedMediaURL(values map[string]any, key string) string {
	if len(values) == 0 {
		return ""
	}
	raw, ok := values[key]
	if !ok {
		return ""
	}
	switch typed := raw.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		return nestedString(typed, "url")
	}
	return ""
}

func nestedValue(values map[string]any, keys ...string) (any, bool) {
	if len(values) == 0 {
		return nil, false
	}
	for _, key := range keys {
		value, ok := values[key]
		if !ok {
			continue
		}
		return value, true
	}
	return nil, false
}

func nestedBool(values map[string]any, keys ...string) (bool, bool) {
	if len(values) == 0 {
		return false, false
	}
	for _, key := range keys {
		value, ok := values[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case bool:
			return typed, true
		case string:
			trimmed := strings.ToLower(strings.TrimSpace(typed))
			if trimmed == "true" {
				return true, true
			}
			if trimmed == "false" {
				return false, true
			}
		}
	}
	return false, false
}

func nestedInt(values map[string]any, keys ...string) (int, bool) {
	if value, ok := nestedInt64(values, keys...); ok {
		return int(value), true
	}
	return 0, false
}

func nestedInt64(values map[string]any, keys ...string) (int64, bool) {
	if len(values) == 0 {
		return 0, false
	}
	for _, key := range keys {
		value, ok := values[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case int:
			return int64(typed), true
		case int64:
			return typed, true
		case float64:
			if typed == float64(int64(typed)) {
				return int64(typed), true
			}
		case string:
			parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
			if err == nil {
				return parsed, true
			}
		}
	}
	return 0, false
}

func messagesText(input map[string]any) string {
	for _, item := range messageContents(input) {
		if text := nestedString(item, "text"); text != "" {
			return text
		}
	}
	return ""
}

func messagesImages(input map[string]any) []string {
	var images []string
	for _, item := range messageContents(input) {
		if image := nestedString(item, "image", "image_url", "url"); image != "" {
			images = append(images, image)
		}
	}
	return images
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

func dashScopeMessages(input map[string]any, prompt string, images []string) []providers.DashScopeMessage {
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
				if text := nestedString(item, "text"); text != "" {
					content = append(content, providers.DashScopeContent{Text: text})
				}
				if image := nestedString(item, "image", "image_url", "url"); image != "" {
					content = append(content, providers.DashScopeContent{Image: image})
				}
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
	for _, image := range images {
		if strings.TrimSpace(image) != "" {
			content = append(content, providers.DashScopeContent{Image: strings.TrimSpace(image)})
		}
	}
	return []providers.DashScopeMessage{{Role: "user", Content: content}}
}

func uniqueStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
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
	reflected := reflect.ValueOf(value)
	if canBeNil(reflected.Kind()) && reflected.IsNil() {
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

func canBeNil(kind reflect.Kind) bool {
	switch kind {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return true
	default:
		return false
	}
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

func yunwuGeminiModel(model string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(model)), "gemini-")
}

func yunwuGeminiParts(in ImageGenerationRequest, prompt string) ([]providers.YunwuGeminiPart, error) {
	images, err := imageReferences(in)
	if err != nil {
		return nil, err
	}
	parts := []providers.YunwuGeminiPart{{Text: prompt}}
	for _, image := range images {
		parts = append(parts, yunwuGeminiImagePart(image))
	}
	return parts, nil
}

func yunwuGeminiImagePart(image string) providers.YunwuGeminiPart {
	mimeType, data, ok := dataURIContent(image)
	if ok {
		return providers.YunwuGeminiPart{
			InlineData: &providers.YunwuGeminiInlineData{
				MimeType: common.DefaultString(mimeType, "image/png"),
				Data:     data,
			},
		}
	}
	return providers.YunwuGeminiPart{
		FileData: &providers.YunwuGeminiFileData{
			MimeType: imageMimeType(image),
			FileURI:  image,
		},
	}
}

func yunwuGeminiImageURLs(out providers.YunwuGeminiResponse) []string {
	var urls []string
	for _, candidate := range out.Candidates {
		for _, part := range candidate.Content.Parts {
			if inline := yunwuInlineData(part); inline != nil && strings.TrimSpace(inline.Data) != "" {
				urls = append(urls, dataURI(yunwuInlineMimeType(*inline), inline.Data))
			}
		}
	}
	return urls
}

func yunwuImageURLs(out providers.YunwuImageResponse, format string) []string {
	var urls []string
	for _, item := range out.Data {
		if item.URL != "" {
			urls = append(urls, item.URL)
			continue
		}
		if item.B64JSON != "" {
			urls = append(urls, dataURI(common.FirstNonEmpty(item.MimeType, imageMimeType(format), "image/png"), item.B64JSON))
		}
	}
	return urls
}

func yunwuGeminiMetadata(out providers.YunwuGeminiResponse) map[string]any {
	metadata := map[string]any{}
	if out.Usage != nil {
		metadata["usage"] = out.Usage
	}
	texts := make([]string, 0)
	finishReasons := make([]string, 0)
	for _, candidate := range out.Candidates {
		if candidate.FinishReason != "" {
			finishReasons = append(finishReasons, candidate.FinishReason)
		}
		for _, part := range candidate.Content.Parts {
			if text := strings.TrimSpace(part.Text); text != "" {
				texts = append(texts, text)
			}
		}
	}
	if len(texts) > 0 {
		metadata["texts"] = texts
	}
	if len(finishReasons) > 0 {
		metadata["finish_reasons"] = finishReasons
	}
	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

func yunwuImageMetadata(out providers.YunwuImageResponse) map[string]any {
	metadata := map[string]any{}
	if out.Created != 0 {
		metadata["created"] = out.Created
	}
	if out.Usage != nil {
		metadata["usage"] = out.Usage
	}
	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

func yunwuInlineData(part providers.YunwuGeminiPart) *providers.YunwuGeminiInlineData {
	if part.InlineData != nil {
		return part.InlineData
	}
	return part.InlineDataSnake
}

func yunwuInlineMimeType(inline providers.YunwuGeminiInlineData) string {
	return common.FirstNonEmpty(inline.MimeType, inline.MimeTypeSnake, "image/png")
}

func dataURI(mimeType, data string) string {
	return "data:" + common.DefaultString(strings.TrimSpace(mimeType), "image/png") + ";base64," + strings.TrimSpace(data)
}

func dataURIContent(value string) (string, string, bool) {
	trimmed := strings.TrimSpace(value)
	if !strings.HasPrefix(strings.ToLower(trimmed), "data:") {
		return "", "", false
	}
	comma := strings.Index(trimmed, ",")
	if comma < 0 {
		return "", "", false
	}
	header := trimmed[len("data:"):comma]
	data := strings.TrimSpace(trimmed[comma+1:])
	if data == "" {
		return "", "", false
	}
	mimeType := header
	if semicolon := strings.Index(mimeType, ";"); semicolon >= 0 {
		mimeType = mimeType[:semicolon]
	}
	return strings.TrimSpace(mimeType), data, true
}

func imageMimeType(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	lower = strings.TrimPrefix(lower, ".")
	switch {
	case strings.HasPrefix(lower, "data:"):
		if end := strings.Index(lower, ";"); end > len("data:") {
			return lower[len("data:"):end]
		}
	case strings.Contains(lower, "jpeg"), strings.Contains(lower, "jpg"):
		return "image/jpeg"
	case strings.Contains(lower, "png"):
		return "image/png"
	case strings.Contains(lower, "webp"):
		return "image/webp"
	case strings.Contains(lower, "gif"):
		return "image/gif"
	}
	return ""
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
