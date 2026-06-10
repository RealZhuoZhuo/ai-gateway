package service

import (
	"context"
	"net/http"
	"strings"

	"github.com/RealZhuoZhuo/ai-gateway/internal/common"
	"github.com/RealZhuoZhuo/ai-gateway/internal/config"
	"github.com/RealZhuoZhuo/ai-gateway/internal/providers"
)

type Gateway struct {
	cfg       ConfigView
	ark       *providers.ArkClient
	dashscope *providers.DashScopeClient
	yunwu     *providers.YunwuClient
}

type ConfigView struct {
	ImageModelProviders modelRouteGroup
	VideoModelProviders modelRouteGroup
}

func NewGateway(cfg config.Config, ark *providers.ArkClient, dashscope *providers.DashScopeClient, yunwu *providers.YunwuClient) *Gateway {
	return &Gateway{
		cfg: ConfigView{
			ImageModelProviders: newModelRouteGroup(cfg.ImageModelProviders),
			VideoModelProviders: newModelRouteGroup(cfg.VideoModelProviders),
		},
		ark:       ark,
		dashscope: dashscope,
		yunwu:     yunwu,
	}
}

func (g *Gateway) GenerateImage(ctx context.Context, requestID string, in ImageGenerationRequest) (ImageGenerationResponse, error) {
	if err := requireString(in.Model, "model"); err != nil {
		return ImageGenerationResponse{}, err
	}
	prompt := imagePrompt(in)
	if err := requireString(prompt, "prompt"); err != nil {
		return ImageGenerationResponse{}, err
	}

	switch g.cfg.ImageModelProviders.providerForModel(in.Model) {
	case "dashscope":
		return g.generateDashScopeImage(ctx, requestID, in, prompt)
	case "ark":
		return g.generateArkImage(ctx, requestID, in, prompt)
	case "yunwu":
		return g.generateYunwuImage(ctx, requestID, in, prompt)
	default:
		return ImageGenerationResponse{}, invalidRequest("model is not configured")
	}
}

func (g *Gateway) generateArkImage(ctx context.Context, requestID string, in ImageGenerationRequest, prompt string) (ImageGenerationResponse, error) {
	size := arkImageSize(in)
	images, err := arkImageReferences(in)
	if err != nil {
		return ImageGenerationResponse{}, err
	}

	out, err := g.ark.GenerateImage(ctx, requestID, providers.ArkImageRequest{
		Model:                     in.Model,
		Prompt:                    prompt,
		SequentialImageGeneration: arkSequentialImageGeneration(in),
		ResponseFormat:            arkResponseFormat(in),
		Size:                      size,
		Stream:                    arkImageStream(in),
		Watermark:                 arkImageWatermark(in),
		Image:                     images,
	})
	if err != nil {
		return ImageGenerationResponse{}, providerError(err)
	}

	urls := arkImageURLs(out)
	if len(urls) == 0 {
		return ImageGenerationResponse{}, newError(http.StatusBadGateway, "provider_bad_response", "ark image response did not include url")
	}

	return ImageGenerationResponse{
		URL:      urls[0],
		URLs:     urls,
		Provider: "ark",
		Model:    in.Model,
		Size:     size,
		Status:   "succeeded",
		Metadata: arkImageMetadata(out),
	}, nil
}

func (g *Gateway) generateDashScopeImage(ctx context.Context, requestID string, in ImageGenerationRequest, prompt string) (ImageGenerationResponse, error) {
	images, err := imageReferences(in)
	if err != nil {
		return ImageGenerationResponse{}, err
	}
	req := providers.DashScopeImageRequest{
		Model: in.Model,
		Input: providers.DashScopeImageInput{
			Messages: dashScopeMessages(in.Input, prompt, images),
		},
		Parameters: dashScopeImageParameters(in, len(images) == 0),
	}

	if in.Async != nil && *in.Async {
		out, err := g.dashscope.CreateImageTask(ctx, requestID, req)
		if err != nil {
			return ImageGenerationResponse{}, providerError(err)
		}
		rawTaskID := out.Output.TaskID
		if rawTaskID == "" {
			return ImageGenerationResponse{}, newError(http.StatusBadGateway, "provider_bad_response", "dashscope image response did not include task id")
		}
		status := normalizeDashScopeStatus(common.FirstNonEmpty(out.Output.TaskStatus, out.Output.Status))
		return ImageGenerationResponse{
			TaskID:   encodeTaskID(taskPrefixDashImage, rawTaskID),
			Provider: "dashscope",
			Model:    in.Model,
			Size:     imageResolution(in),
			Status:   common.DefaultString(status, "queued"),
			Metadata: map[string]any{"request_id": out.RequestID},
		}, nil
	}

	out, err := g.dashscope.GenerateImage(ctx, requestID, req)
	if err != nil {
		return ImageGenerationResponse{}, providerError(err)
	}
	urls := dashScopeImageURLs(out)
	if len(urls) == 0 {
		return ImageGenerationResponse{}, newError(http.StatusBadGateway, "provider_bad_response", "dashscope image response did not include url")
	}
	return ImageGenerationResponse{
		URL:      urls[0],
		URLs:     urls,
		Provider: "dashscope",
		Model:    in.Model,
		Size:     imageResolution(in),
		Status:   "succeeded",
		Metadata: map[string]any{"request_id": out.RequestID, "usage": out.Usage},
	}, nil
}

func (g *Gateway) generateYunwuImage(ctx context.Context, requestID string, in ImageGenerationRequest, prompt string) (ImageGenerationResponse, error) {
	if in.Async != nil && *in.Async {
		return ImageGenerationResponse{}, invalidRequest("yunwu image generation does not support async")
	}

	if yunwuGeminiModel(in.Model) {
		parts, err := yunwuGeminiParts(in, prompt)
		if err != nil {
			return ImageGenerationResponse{}, err
		}
		out, err := g.yunwu.GenerateGeminiImage(ctx, requestID, in.Model, providers.YunwuGeminiRequest{
			Contents: []providers.YunwuGeminiContent{{
				Role:  "user",
				Parts: parts,
			}},
			GenerationConfig: providers.YunwuGeminiGenerationConfig{
				ResponseModalities: []string{"IMAGE", "TEXT"},
			},
		})
		if err != nil {
			return ImageGenerationResponse{}, providerError(err)
		}

		urls := yunwuGeminiImageURLs(out)
		if len(urls) == 0 {
			return ImageGenerationResponse{}, newError(http.StatusBadGateway, "provider_bad_response", "yunwu image response did not include image")
		}
		return ImageGenerationResponse{
			URL:      urls[0],
			URLs:     urls,
			Provider: "yunwu",
			Model:    in.Model,
			Size:     imageResolution(in),
			Status:   "succeeded",
			Metadata: yunwuGeminiMetadata(out),
		}, nil
	}

	images, err := imageReferences(in)
	if err != nil {
		return ImageGenerationResponse{}, err
	}
	out, err := g.yunwu.GenerateImage(ctx, requestID, providers.YunwuImageRequest{
		Model:          in.Model,
		Prompt:         prompt,
		N:              in.N,
		Size:           imageResolution(in),
		Image:          images,
		ResponseFormat: imageResponseFormat(in),
	})
	if err != nil {
		return ImageGenerationResponse{}, providerError(err)
	}

	urls := yunwuImageURLs(out, imageFormat(in))
	if len(urls) == 0 {
		return ImageGenerationResponse{}, newError(http.StatusBadGateway, "provider_bad_response", "yunwu image response did not include image")
	}
	return ImageGenerationResponse{
		URL:      urls[0],
		URLs:     urls,
		Provider: "yunwu",
		Model:    in.Model,
		Size:     imageResolution(in),
		Status:   "succeeded",
		Metadata: yunwuImageMetadata(out),
	}, nil
}

func (g *Gateway) GetImageTask(ctx context.Context, requestID, taskID string) (GetImageTaskResponse, error) {
	if strings.TrimSpace(taskID) == "" {
		return GetImageTaskResponse{}, invalidRequest("task_id is required")
	}
	prefix, rawTaskID, ok := decodeTaskID(taskID)
	if !ok {
		return GetImageTaskResponse{}, invalidRequest("image task_id must include a provider prefix")
	}
	switch prefix {
	case taskPrefixDashImage:
		out, err := g.dashscope.GetTask(ctx, requestID, rawTaskID)
		if err != nil {
			return GetImageTaskResponse{}, providerError(err)
		}
		urls := dashScopeTaskImageURLs(out.Output)
		metadata := map[string]any{"raw_task_id": rawTaskID, "request_id": out.RequestID}
		if out.Usage != nil {
			metadata["usage"] = out.Usage
		}
		return GetImageTaskResponse{
			TaskID:   taskID,
			Provider: "dashscope",
			Model:    g.cfg.ImageModelProviders.modelForProvider("dashscope"),
			Status:   normalizeDashScopeStatus(common.FirstNonEmpty(out.Output.TaskStatus, out.Output.Status)),
			URL:      nullableFirst(urls),
			URLs:     urls,
			Error:    dashScopeTaskError(out.Output),
			Metadata: metadata,
		}, nil
	default:
		return GetImageTaskResponse{}, invalidRequest("unknown image task_id prefix")
	}
}

func (g *Gateway) CreateVideoTask(ctx context.Context, requestID string, in CreateVideoTaskRequest) (CreateVideoTaskResponse, error) {
	if err := requireString(in.Model, "model"); err != nil {
		return CreateVideoTaskResponse{}, err
	}
	prompt := videoPrompt(in)
	if err := requireString(prompt, "prompt"); err != nil {
		return CreateVideoTaskResponse{}, err
	}

	switch g.cfg.VideoModelProviders.providerForModel(in.Model) {
	case "dashscope":
		return g.createDashScopeVideoTask(ctx, requestID, in, prompt)
	case "ark":
		return g.createArkVideoTask(ctx, requestID, in, prompt)
	default:
		return CreateVideoTaskResponse{}, invalidRequest("model is not configured")
	}
}

func (g *Gateway) createArkVideoTask(ctx context.Context, requestID string, in CreateVideoTaskRequest, prompt string) (CreateVideoTaskResponse, error) {
	content, err := arkVideoContent(in, prompt)
	if err != nil {
		return CreateVideoTaskResponse{}, err
	}

	out, err := g.ark.CreateVideoTask(ctx, requestID, providers.ArkVideoCreateRequest{
		Model:         in.Model,
		Content:       content,
		GenerateAudio: arkVideoGenerateAudio(in),
		Ratio:         arkVideoRatio(in),
		Duration:      arkVideoDuration(in),
		Watermark:     arkVideoWatermark(in),
		Seed:          arkVideoSeed(in),
		Resolution:    arkVideoResolution(in),
	})
	if err != nil {
		return CreateVideoTaskResponse{}, providerError(err)
	}

	taskID := common.FirstNonEmpty(out.TaskID, out.ID)
	if taskID == "" {
		return CreateVideoTaskResponse{}, newError(http.StatusBadGateway, "provider_bad_response", "ark video response did not include task id")
	}
	status := normalizeTaskStatus(out.Status)
	if status == "unknown" {
		status = "queued"
	}
	return CreateVideoTaskResponse{
		TaskID:   encodeTaskID(taskPrefixArk, taskID),
		Provider: "ark-seedance",
		Model:    in.Model,
		Status:   status,
	}, nil
}

func (g *Gateway) createDashScopeVideoTask(ctx context.Context, requestID string, in CreateVideoTaskRequest, prompt string) (CreateVideoTaskResponse, error) {
	input, err := dashScopeVideoInput(in, prompt)
	if err != nil {
		return CreateVideoTaskResponse{}, err
	}

	prefix := taskPrefixDashTextVideo
	switch dashScopeVideoMode(in, input.Media) {
	case videoModeImageToVideo:
		if len(input.Media) == 0 {
			return CreateVideoTaskResponse{}, invalidRequest("image is required for image-to-video model")
		}
		prefix = taskPrefixDashImageVideo
	case videoModeReferenceToVideo:
		prefix = taskPrefixDashRefVideo
	}

	out, err := g.dashscope.CreateVideoTask(ctx, requestID, providers.DashScopeVideoRequest{
		Model:      in.Model,
		Input:      input,
		Parameters: dashScopeVideoParameters(in),
	})
	if err != nil {
		return CreateVideoTaskResponse{}, providerError(err)
	}
	rawTaskID := out.Output.TaskID
	if rawTaskID == "" {
		return CreateVideoTaskResponse{}, newError(http.StatusBadGateway, "provider_bad_response", "dashscope video response did not include task id")
	}
	status := normalizeDashScopeStatus(common.FirstNonEmpty(out.Output.TaskStatus, out.Output.Status))
	return CreateVideoTaskResponse{
		TaskID:   encodeTaskID(prefix, rawTaskID),
		Provider: "dashscope",
		Model:    in.Model,
		Status:   common.DefaultString(status, "queued"),
	}, nil
}

func (g *Gateway) GetVideoTask(ctx context.Context, requestID, taskID string) (GetVideoTaskResponse, error) {
	if strings.TrimSpace(taskID) == "" {
		return GetVideoTaskResponse{}, invalidRequest("task_id is required")
	}
	prefix, rawTaskID, ok := decodeTaskID(taskID)
	if !ok {
		rawTaskID = taskID
		prefix = taskPrefixArk
	}
	switch prefix {
	case taskPrefixArk:
		return g.getArkVideoTask(ctx, requestID, rawTaskID, taskID)
	case taskPrefixDashTextVideo, taskPrefixDashImageVideo, taskPrefixDashRefVideo:
		return g.getDashScopeVideoTask(ctx, requestID, rawTaskID, taskID, prefix)
	default:
		return GetVideoTaskResponse{}, invalidRequest("unknown task_id prefix")
	}
}

func (g *Gateway) getArkVideoTask(ctx context.Context, requestID, rawTaskID, publicTaskID string) (GetVideoTaskResponse, error) {
	out, err := g.ark.GetVideoTask(ctx, requestID, rawTaskID)
	if err != nil {
		return GetVideoTaskResponse{}, providerError(err)
	}

	videoURL := common.FirstNonEmpty(out.VideoURL)
	lastFrameURL := common.FirstNonEmpty(out.LastFrameURL)
	if out.Content != nil {
		videoURL = common.FirstNonEmpty(videoURL, out.Content.VideoURL, out.Content.URL)
		lastFrameURL = common.FirstNonEmpty(lastFrameURL, out.Content.LastFrameURL)
	}

	metadata := map[string]any{}
	if out.CreatedAt != 0 {
		metadata["created_at"] = out.CreatedAt
	}
	if out.UpdatedAt != 0 {
		metadata["updated_at"] = out.UpdatedAt
	}
	if out.Seed != nil {
		metadata["seed"] = *out.Seed
	}
	if out.Resolution != "" {
		metadata["resolution"] = out.Resolution
	}
	if out.Ratio != "" {
		metadata["ratio"] = out.Ratio
	}
	if out.Duration != nil {
		metadata["duration"] = *out.Duration
	}
	if out.Usage != nil {
		metadata["usage"] = out.Usage
	}

	status := normalizeTaskStatus(out.Status)
	if status != "succeeded" {
		videoURL = ""
		lastFrameURL = ""
	}

	return GetVideoTaskResponse{
		TaskID:       common.DefaultString(publicTaskID, encodeTaskID(taskPrefixArk, common.FirstNonEmpty(out.TaskID, out.ID, rawTaskID))),
		Provider:     "ark-seedance",
		Model:        out.Model,
		Status:       status,
		VideoURL:     nullableString(videoURL),
		LastFrameURL: nullableString(lastFrameURL),
		Error:        taskErrorFromProvider(out.Error),
		Metadata:     metadata,
	}, nil
}

func (g *Gateway) getDashScopeVideoTask(ctx context.Context, requestID, rawTaskID, publicTaskID, prefix string) (GetVideoTaskResponse, error) {
	out, err := g.dashscope.GetTask(ctx, requestID, rawTaskID)
	if err != nil {
		return GetVideoTaskResponse{}, providerError(err)
	}
	mode := videoModeTextToVideo
	if prefix == taskPrefixDashImageVideo {
		mode = videoModeImageToVideo
	} else if prefix == taskPrefixDashRefVideo {
		mode = videoModeReferenceToVideo
	}
	metadata := map[string]any{"raw_task_id": rawTaskID, "request_id": out.RequestID}
	if out.Usage != nil {
		metadata["usage"] = out.Usage
	}
	return GetVideoTaskResponse{
		TaskID:   common.DefaultString(publicTaskID, encodeTaskID(prefix, rawTaskID)),
		Provider: "dashscope",
		Model:    g.cfg.VideoModelProviders.modelForProviderMode("dashscope", mode),
		Status:   normalizeDashScopeStatus(common.FirstNonEmpty(out.Output.TaskStatus, out.Output.Status)),
		VideoURL: nullableString(dashScopeTaskVideoURL(out.Output)),
		Error:    dashScopeTaskError(out.Output),
		Metadata: metadata,
	}, nil
}
