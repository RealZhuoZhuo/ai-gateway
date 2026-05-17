package httpapi

import (
	"context"

	"github.com/RealZhuoZhuo/ai-gateway/internal/service"
)

type GatewayService interface {
	GenerateImage(ctx context.Context, requestID string, in service.ImageGenerationRequest) (service.ImageGenerationResponse, error)
	GetImageTask(ctx context.Context, requestID, taskID string) (service.GetImageTaskResponse, error)
	CreateVideoTask(ctx context.Context, requestID string, in service.CreateVideoTaskRequest) (service.CreateVideoTaskResponse, error)
	GetVideoTask(ctx context.Context, requestID, taskID string) (service.GetVideoTaskResponse, error)
}
