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
	kling     *providers.KlingClient
	dashscope *providers.DashScopeClient
}

type ConfigView struct {
	ImageModelProviders modelRouteGroup
	VideoModelProviders modelRouteGroup
}

func NewGateway(cfg config.Config, ark *providers.ArkClient, kling *providers.KlingClient, dashscope *providers.DashScopeClient) *Gateway {
	return &Gateway{
		cfg: ConfigView{
			ImageModelProviders: newModelRouteGroup(cfg.ImageModelProviders),
			VideoModelProviders: newModelRouteGroup(cfg.VideoModelProviders),
		},
		ark:       ark,
		kling:     kling,
		dashscope: dashscope,
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
	case "kling":
		return g.generateKlingImage(ctx, requestID, in, prompt)
	case "ark":
		return g.generateArkImage(ctx, requestID, in, prompt)
	default:
		return ImageGenerationResponse{}, invalidRequest("model is not configured")
	}
}

func (g *Gateway) generateArkImage(ctx context.Context, requestID string, in ImageGenerationRequest, prompt string) (ImageGenerationResponse, error) {
	size := in.Size
	images := make([]string, 0, len(in.ReferenceImages))
	for _, item := range in.ReferenceImages {
		if strings.TrimSpace(item.URL) == "" {
			return ImageGenerationResponse{}, invalidRequest("reference_images.url is required")
		}
		images = append(images, item.URL)
	}

	out, err := g.ark.GenerateImage(ctx, requestID, providers.ArkImageRequest{
		Model:                     in.Model,
		Prompt:                    prompt,
		SequentialImageGeneration: "disabled",
		ResponseFormat:            "url",
		Size:                      size,
		Stream:                    false,
		Watermark:                 false,
		Image:                     images,
	})
	if err != nil {
		return ImageGenerationResponse{}, providerError(err)
	}

	imageURL := out.URL
	if imageURL == "" && len(out.Data) > 0 {
		imageURL = out.Data[0].URL
	}
	if imageURL == "" {
		return ImageGenerationResponse{}, newError(http.StatusBadGateway, "provider_bad_response", "ark image response did not include url")
	}

	return ImageGenerationResponse{
		URL:      imageURL,
		URLs:     []string{imageURL},
		Provider: "ark",
		Model:    in.Model,
		Size:     size,
		Status:   "succeeded",
	}, nil
}

func (g *Gateway) generateDashScopeImage(ctx context.Context, requestID string, in ImageGenerationRequest, prompt string) (ImageGenerationResponse, error) {
	params := cloneMap(in.Parameters)
	params = setIfMissing(params, "negative_prompt", in.NegativePrompt)
	params = setIfMissing(params, "size", imageResolution(in))
	params = setIfMissing(params, "n", in.N)
	params = setIfMissing(params, "aspect_ratio", imageAspectRatio(in))

	out, err := g.dashscope.GenerateImage(ctx, requestID, providers.DashScopeImageRequest{
		Model: in.Model,
		Input: providers.DashScopeImageInput{
			Messages: dashScopeMessages(in.Input, prompt, firstImage(in)),
		},
		Parameters: params,
	})
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

func (g *Gateway) generateKlingImage(ctx context.Context, requestID string, in ImageGenerationRequest, prompt string) (ImageGenerationResponse, error) {
	out, err := g.kling.CreateImageTask(ctx, requestID, providers.KlingImageCreateRequest{
		ModelName:      in.Model,
		Prompt:         prompt,
		NegativePrompt: in.NegativePrompt,
		Image:          firstImage(in),
		ImageReference: in.ImageReference,
		ImageFidelity:  in.ImageFidelity,
		HumanFidelity:  in.HumanFidelity,
		Resolution:     klingImageResolution(in),
		N:              in.N,
		AspectRatio:    imageAspectRatio(in),
		CallbackURL:    in.CallbackURL,
		ExternalTaskID: in.ExternalTaskID,
	})
	if err != nil {
		return ImageGenerationResponse{}, providerError(err)
	}
	rawTaskID := out.Data.TaskID
	if rawTaskID == "" {
		return ImageGenerationResponse{}, newError(http.StatusBadGateway, "provider_bad_response", "kling image response did not include task id")
	}
	taskID := encodeTaskID(taskPrefixKlingImage, rawTaskID)
	polled, err := g.waitKlingImage(ctx, requestID, rawTaskID)
	if err != nil {
		if ctx.Err() != nil {
			return ImageGenerationResponse{}, newError(http.StatusGatewayTimeout, "provider_timeout", "kling image task did not finish before request timeout; query task_id "+taskID)
		}
		return ImageGenerationResponse{}, err
	}
	urls := klingImageURLs(polled)
	if len(urls) == 0 {
		return ImageGenerationResponse{}, newError(http.StatusBadGateway, "provider_bad_response", "kling image response did not include url")
	}
	return ImageGenerationResponse{
		URL:      urls[0],
		URLs:     urls,
		TaskID:   taskID,
		Provider: "kling",
		Model:    in.Model,
		Size:     klingImageResolution(in),
		Status:   normalizeKlingStatus(polled.Data.TaskStatus),
		Metadata: map[string]any{"raw_task_id": rawTaskID, "request_id": polled.RequestID},
	}, nil
}

func (g *Gateway) waitKlingImage(ctx context.Context, requestID, rawTaskID string) (providers.KlingTaskResponse, error) {
	var last providers.KlingTaskResponse
	for attempt := 0; ; attempt++ {
		out, err := g.kling.GetImageTask(ctx, requestID, rawTaskID)
		if err != nil {
			return out, providerError(err)
		}
		last = out
		status := normalizeKlingStatus(out.Data.TaskStatus)
		if status == "succeeded" {
			return out, nil
		}
		if status == "failed" || status == "cancelled" || status == "expired" {
			return out, newError(http.StatusBadGateway, "provider_error", "kling image task ended with status "+out.Data.TaskStatus)
		}
		timer := timeAfter(waitDuration(attempt))
		select {
		case <-ctx.Done():
			return last, ctx.Err()
		case <-timer:
		}
	}
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
	case taskPrefixKlingImage:
		out, err := g.kling.GetImageTask(ctx, requestID, rawTaskID)
		if err != nil {
			return GetImageTaskResponse{}, providerError(err)
		}
		urls := klingImageURLs(out)
		metadata := map[string]any{"raw_task_id": rawTaskID, "request_id": out.RequestID}
		return GetImageTaskResponse{
			TaskID:   taskID,
			Provider: "kling",
			Status:   normalizeKlingStatus(out.Data.TaskStatus),
			URL:      nullableFirst(urls),
			URLs:     urls,
			Error:    taskErrorFromProvider(out.Data.Error),
			Metadata: metadata,
		}, nil
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
	case "kling":
		return g.createKlingVideoTask(ctx, requestID, in, prompt)
	case "ark":
		return g.createArkVideoTask(ctx, requestID, in, prompt)
	default:
		return CreateVideoTaskResponse{}, invalidRequest("model is not configured")
	}
}

func (g *Gateway) createArkVideoTask(ctx context.Context, requestID string, in CreateVideoTaskRequest, prompt string) (CreateVideoTaskResponse, error) {
	firstFrameURL := firstFrame(in)
	if err := requireString(firstFrameURL, "first_frame_url"); err != nil {
		return CreateVideoTaskResponse{}, err
	}
	if in.Resolution != "" && !common.OneOf(in.Resolution, "480p", "720p", "1080p") {
		return CreateVideoTaskResponse{}, invalidRequest("resolution must be one of 480p, 720p, 1080p")
	}
	ratio := videoAspectRatio(in)
	if ratio != "" && !common.OneOf(ratio, "16:9", "4:3", "1:1", "3:4", "9:16", "21:9", "adaptive") {
		return CreateVideoTaskResponse{}, invalidRequest("ratio is invalid")
	}
	if in.Duration != nil && (*in.Duration < 4 || *in.Duration > 15) {
		return CreateVideoTaskResponse{}, invalidRequest("duration must be between 4 and 15 seconds")
	}
	generateAudio := true
	if in.GenerateAudio != nil {
		generateAudio = *in.GenerateAudio
	}
	returnLastFrame := false
	if in.ReturnLastFrame != nil {
		returnLastFrame = *in.ReturnLastFrame
	}

	out, err := g.ark.CreateVideoTask(ctx, requestID, providers.ArkVideoCreateRequest{
		Model: in.Model,
		Content: []providers.ArkVideoContent{
			{Type: "text", Text: prompt},
			{Type: "image_url", ImageURL: &providers.ArkVideoImageURL{URL: firstFrameURL}, Role: "first_frame"},
		},
		ServiceTier:     "default",
		GenerateAudio:   generateAudio,
		ReturnLastFrame: returnLastFrame,
		Watermark:       false,
		Resolution:      in.Resolution,
		Ratio:           ratio,
		Duration:        in.Duration,
		Seed:            in.Seed,
	})
	if err != nil {
		return CreateVideoTaskResponse{}, providerError(err)
	}

	taskID := common.FirstNonEmpty(out.TaskID, out.ID)
	if taskID == "" {
		return CreateVideoTaskResponse{}, newError(http.StatusBadGateway, "provider_bad_response", "ark video response did not include task id")
	}
	status := normalizeTaskStatus(out.Status)
	if status == "" {
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
	input := cloneMap(in.Input)
	input = setIfMissing(input, "prompt", prompt)
	image := firstFrame(in)
	prefix := taskPrefixDashTextVideo
	if dashScopeVideoMode(in, image) == videoModeImageToVideo {
		if image == "" {
			return CreateVideoTaskResponse{}, invalidRequest("image is required for image-to-video model")
		}
		input = setIfMissing(input, "img_url", image)
		input = setIfMissing(input, "image", image)
		if in.ImageTail != "" {
			input = setIfMissing(input, "last_frame_url", in.ImageTail)
		}
		prefix = taskPrefixDashImageVideo
	}

	params := cloneMap(in.Parameters)
	params = setIfMissing(params, "negative_prompt", in.NegativePrompt)
	params = setIfMissing(params, "duration", in.Duration)
	params = setIfMissing(params, "size", in.Resolution)
	params = setIfMissing(params, "resolution", in.Resolution)
	params = setIfMissing(params, "ratio", videoAspectRatio(in))
	params = setIfMissing(params, "aspect_ratio", videoAspectRatio(in))
	params = setIfMissing(params, "seed", in.Seed)
	params = setIfMissing(params, "audio", in.GenerateAudio)

	out, err := g.dashscope.CreateVideoTask(ctx, requestID, providers.DashScopeVideoRequest{
		Model:      in.Model,
		Input:      input,
		Parameters: params,
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

func (g *Gateway) createKlingVideoTask(ctx context.Context, requestID string, in CreateVideoTaskRequest, prompt string) (CreateVideoTaskResponse, error) {
	firstFrameURL := firstFrame(in)
	req := providers.KlingVideoCreateRequest{
		ModelName:      in.Model,
		Prompt:         prompt,
		NegativePrompt: in.NegativePrompt,
		Image:          firstFrameURL,
		ImageTail:      in.ImageTail,
		Mode:           in.Mode,
		Duration:       videoDurationString(in),
		AspectRatio:    videoAspectRatio(in),
		CFGScale:       in.CFGScale,
		Sound:          in.Sound,
		StaticMask:     in.StaticMask,
		DynamicMasks:   in.DynamicMasks,
		CameraControl:  in.CameraControl,
		CallbackURL:    in.CallbackURL,
		ExternalTaskID: in.ExternalTaskID,
	}
	var (
		out    providers.KlingTaskResponse
		err    error
		prefix = taskPrefixKlingTextVideo
	)
	if firstFrameURL != "" || in.ImageTail != "" {
		if firstFrameURL == "" {
			return CreateVideoTaskResponse{}, invalidRequest("image is required when image_tail is provided")
		}
		out, err = g.kling.CreateImageToVideoTask(ctx, requestID, req)
		prefix = taskPrefixKlingImageVideo
	} else {
		out, err = g.kling.CreateTextToVideoTask(ctx, requestID, req)
	}
	if err != nil {
		return CreateVideoTaskResponse{}, providerError(err)
	}
	rawTaskID := out.Data.TaskID
	if rawTaskID == "" {
		return CreateVideoTaskResponse{}, newError(http.StatusBadGateway, "provider_bad_response", "kling video response did not include task id")
	}
	status := normalizeKlingStatus(out.Data.TaskStatus)
	return CreateVideoTaskResponse{
		TaskID:   encodeTaskID(prefix, rawTaskID),
		Provider: "kling",
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
	case taskPrefixKlingTextVideo:
		return g.getKlingTextVideoTask(ctx, requestID, rawTaskID, taskID)
	case taskPrefixKlingImageVideo:
		return g.getKlingImageVideoTask(ctx, requestID, rawTaskID, taskID)
	case taskPrefixDashTextVideo, taskPrefixDashImageVideo:
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

	return GetVideoTaskResponse{
		TaskID:       common.DefaultString(publicTaskID, encodeTaskID(taskPrefixArk, common.FirstNonEmpty(out.TaskID, out.ID, rawTaskID))),
		Provider:     "ark-seedance",
		Model:        out.Model,
		Status:       normalizeTaskStatus(out.Status),
		VideoURL:     nullableString(videoURL),
		LastFrameURL: nullableString(lastFrameURL),
		Error:        taskErrorFromProvider(out.Error),
		Metadata:     metadata,
	}, nil
}

func (g *Gateway) getKlingTextVideoTask(ctx context.Context, requestID, rawTaskID, publicTaskID string) (GetVideoTaskResponse, error) {
	out, err := g.kling.GetTextToVideoTask(ctx, requestID, rawTaskID)
	if err != nil {
		return GetVideoTaskResponse{}, providerError(err)
	}
	return klingVideoTaskResponse(out, publicTaskID, "kling", taskPrefixKlingTextVideo, rawTaskID)
}

func (g *Gateway) getKlingImageVideoTask(ctx context.Context, requestID, rawTaskID, publicTaskID string) (GetVideoTaskResponse, error) {
	out, err := g.kling.GetImageToVideoTask(ctx, requestID, rawTaskID)
	if err != nil {
		return GetVideoTaskResponse{}, providerError(err)
	}
	return klingVideoTaskResponse(out, publicTaskID, "kling", taskPrefixKlingImageVideo, rawTaskID)
}

func klingVideoTaskResponse(out providers.KlingTaskResponse, publicTaskID, providerName, prefix, rawTaskID string) (GetVideoTaskResponse, error) {
	metadata := map[string]any{"raw_task_id": rawTaskID, "request_id": out.RequestID}
	if out.Data.CreatedAt != 0 {
		metadata["created_at"] = out.Data.CreatedAt
	}
	if out.Data.UpdatedAt != 0 {
		metadata["updated_at"] = out.Data.UpdatedAt
	}
	return GetVideoTaskResponse{
		TaskID:   common.DefaultString(publicTaskID, encodeTaskID(prefix, rawTaskID)),
		Provider: providerName,
		Model:    "",
		Status:   normalizeKlingStatus(out.Data.TaskStatus),
		VideoURL: nullableString(klingVideoURL(out)),
		Error:    taskErrorFromProvider(out.Data.Error),
		Metadata: metadata,
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
