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
		Model:           in.Model,
		Prompt:          in.Prompt,
		NegativePrompt:  in.NegativePrompt,
		Size:            in.Size,
		Resolution:      in.Resolution,
		AspectRatio:     in.AspectRatio,
		N:               in.N,
		ReferenceImages: serviceReferenceImages(in.ReferenceImages),
		Image:           in.Image,
		ImageReference:  in.ImageReference,
		ImageFidelity:   in.ImageFidelity,
		HumanFidelity:   in.HumanFidelity,
		CallbackURL:     in.CallbackURL,
		ExternalTaskID:  in.ExternalTaskID,
		Input:           in.Input,
		Parameters:      in.Parameters,
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
		Model:           in.Model,
		Prompt:          in.Prompt,
		NegativePrompt:  in.NegativePrompt,
		FirstFrameURL:   in.FirstFrameURL,
		Image:           in.Image,
		ImageTail:       in.ImageTail,
		Resolution:      in.Resolution,
		Ratio:           in.Ratio,
		AspectRatio:     in.AspectRatio,
		Duration:        in.Duration,
		Seed:            in.Seed,
		GenerateAudio:   in.GenerateAudio,
		ReturnLastFrame: in.ReturnLastFrame,
		Mode:            in.Mode,
		Sound:           in.Sound,
		CFGScale:        in.CFGScale,
		StaticMask:      in.StaticMask,
		DynamicMasks:    in.DynamicMasks,
		CameraControl:   in.CameraControl,
		CallbackURL:     in.CallbackURL,
		ExternalTaskID:  in.ExternalTaskID,
		Input:           in.Input,
		Parameters:      in.Parameters,
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
