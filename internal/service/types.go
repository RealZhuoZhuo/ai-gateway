package service

type ImageGenerationRequest struct {
	Model           string
	Prompt          string
	NegativePrompt  string
	Size            string
	Resolution      string
	AspectRatio     string
	N               *int
	ReferenceImages []ReferenceImage
	Image           string
	ImageReference  string
	ImageFidelity   *float64
	HumanFidelity   *float64
	CallbackURL     string
	ExternalTaskID  string
	Input           map[string]any
	Parameters      map[string]any
}

type ReferenceImage struct {
	URL string
}

type ImageGenerationResponse struct {
	URL      string
	URLs     []string
	TaskID   string
	Provider string
	Model    string
	Size     string
	Status   string
	Metadata map[string]any
}

type GetImageTaskResponse struct {
	TaskID   string
	Provider string
	Model    string
	Status   string
	URL      *string
	URLs     []string
	Error    *TaskError
	Metadata map[string]any
}

type CreateVideoTaskRequest struct {
	Model           string
	Prompt          string
	NegativePrompt  string
	FirstFrameURL   string
	Image           string
	ImageTail       string
	Resolution      string
	Ratio           string
	AspectRatio     string
	Duration        *int
	Seed            *int64
	GenerateAudio   *bool
	ReturnLastFrame *bool
	Mode            string
	Sound           string
	CFGScale        *float64
	StaticMask      any
	DynamicMasks    any
	CameraControl   map[string]any
	CallbackURL     string
	ExternalTaskID  string
	Input           map[string]any
	Parameters      map[string]any
}

type CreateVideoTaskResponse struct {
	TaskID   string
	Provider string
	Model    string
	Status   string
}

type GetVideoTaskResponse struct {
	TaskID       string
	Provider     string
	Model        string
	Status       string
	VideoURL     *string
	LastFrameURL *string
	Error        *TaskError
	Metadata     map[string]any
}

type TaskError struct {
	Code    string
	Message string
}
