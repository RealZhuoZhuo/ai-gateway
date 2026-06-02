package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/RealZhuoZhuo/ai-gateway/internal/httpapi/dto"
	"github.com/RealZhuoZhuo/ai-gateway/internal/service"
)

func (h *Handler) Healthz(c *gin.Context) {
	WriteJSON(c, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) GenerateImage(c *gin.Context) {
	var in dto.ImageGenerationRequest
	if !DecodeJSON(c, &in) {
		return
	}

	out, err := h.gateway.GenerateImage(c.Request.Context(), RequestIDFromContext(c), service.ImageGenerationRequest{
		Model:                     in.Model,
		Prompt:                    in.Prompt,
		NegativePrompt:            in.NegativePrompt,
		Size:                      in.Size,
		Resolution:                in.Resolution,
		AspectRatio:               in.AspectRatio,
		N:                         in.N,
		ReferenceImages:           serviceReferenceImages(in.ReferenceImages),
		Image:                     in.Image.First(),
		Images:                    in.Image.Values(),
		Quality:                   in.Quality,
		Format:                    in.Format,
		SequentialImageGeneration: in.SequentialImageGeneration,
		ResponseFormat:            in.ResponseFormat,
		Stream:                    in.Stream,
		Watermark:                 in.Watermark,
		Async:                     in.Async,
		Input:                     in.Input,
		Parameters:                in.Parameters,
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, dto.ImageGenerationResponse{
		URL:      out.URL,
		URLs:     out.URLs,
		TaskID:   out.TaskID,
		Provider: out.Provider,
		Model:    out.Model,
		Size:     out.Size,
		Status:   out.Status,
		Metadata: out.Metadata,
	})
}

func (h *Handler) GetImageTask(c *gin.Context) {
	out, err := h.gateway.GetImageTask(c.Request.Context(), RequestIDFromContext(c), c.Param("task_id"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, dto.GetImageTaskResponse{
		TaskID:   out.TaskID,
		Provider: out.Provider,
		Model:    out.Model,
		Status:   out.Status,
		URL:      out.URL,
		URLs:     out.URLs,
		Error:    dtoTaskError(out.Error),
		Metadata: out.Metadata,
	})
}

func (h *Handler) CreateVideoTask(c *gin.Context) {
	var in dto.CreateVideoTaskRequest
	if !DecodeJSON(c, &in) {
		return
	}

	out, err := h.gateway.CreateVideoTask(c.Request.Context(), RequestIDFromContext(c), service.CreateVideoTaskRequest{
		Model:          in.Model,
		Prompt:         in.Prompt,
		Content:        serviceVideoContent(in.Content),
		Media:          serviceVideoMedia(in.Media),
		NegativePrompt: in.NegativePrompt,
		FirstFrameURL:  in.FirstFrameURL,
		Image:          in.Image,
		ImageTail:      in.ImageTail,
		Resolution:     in.Resolution,
		Ratio:          in.Ratio,
		AspectRatio:    in.AspectRatio,
		Duration:       in.Duration,
		Seed:           in.Seed,
		Input:          in.Input,
		Parameters:     in.Parameters,
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, dto.CreateVideoTaskResponse{
		TaskID:   out.TaskID,
		Provider: out.Provider,
		Model:    out.Model,
		Status:   out.Status,
	})
}

func (h *Handler) GetVideoTask(c *gin.Context) {
	out, err := h.gateway.GetVideoTask(c.Request.Context(), RequestIDFromContext(c), c.Param("task_id"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, dto.GetVideoTaskResponse{
		TaskID:       out.TaskID,
		Provider:     out.Provider,
		Model:        out.Model,
		Status:       out.Status,
		VideoURL:     out.VideoURL,
		LastFrameURL: out.LastFrameURL,
		Error:        dtoTaskError(out.Error),
		Metadata:     out.Metadata,
	})
}

func dtoTaskError(err *service.TaskError) *dto.TaskError {
	if err == nil {
		return nil
	}
	return &dto.TaskError{Code: err.Code, Message: err.Message}
}

func serviceReferenceImages(images []dto.ReferenceImage) []service.ReferenceImage {
	out := make([]service.ReferenceImage, 0, len(images))
	for _, image := range images {
		out = append(out, service.ReferenceImage{URL: image.URL})
	}
	return out
}

func serviceVideoContent(items []dto.VideoContent) []service.VideoContent {
	out := make([]service.VideoContent, 0, len(items))
	for _, item := range items {
		var imageURL *service.MediaURL
		if item.ImageURL != nil {
			imageURL = &service.MediaURL{URL: item.ImageURL.URL}
		}
		out = append(out, service.VideoContent{
			Type:     item.Type,
			Text:     item.Text,
			ImageURL: imageURL,
			Role:     item.Role,
		})
	}
	return out
}

func serviceVideoMedia(items []dto.VideoMedia) []service.VideoMedia {
	out := make([]service.VideoMedia, 0, len(items))
	for _, item := range items {
		out = append(out, service.VideoMedia{
			Type:           item.Type,
			URL:            item.URL,
			ReferenceVoice: item.ReferenceVoice,
		})
	}
	return out
}
