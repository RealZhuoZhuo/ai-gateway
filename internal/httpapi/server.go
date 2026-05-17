package httpapi

type Handler struct {
	gateway GatewayService
}

func NewHandler(gateway GatewayService) *Handler {
	return &Handler{gateway: gateway}
}
