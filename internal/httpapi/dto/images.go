package dto

type ImageGenerationRequest struct {
	Model           string           `json:"model"`
	Prompt          string           `json:"prompt"`
	NegativePrompt  string           `json:"negative_prompt"`
	Size            string           `json:"size"`
	Resolution      string           `json:"resolution"`
	AspectRatio     string           `json:"aspect_ratio"`
	N               *int             `json:"n"`
	ReferenceImages []ReferenceImage `json:"reference_images"`
	Image           string           `json:"image"`
	ImageReference  string           `json:"image_reference"`
	ImageFidelity   *float64         `json:"image_fidelity"`
	HumanFidelity   *float64         `json:"human_fidelity"`
	CallbackURL     string           `json:"callback_url"`
	ExternalTaskID  string           `json:"external_task_id"`
	Input           map[string]any   `json:"input"`
	Parameters      map[string]any   `json:"parameters"`
}

type ReferenceImage struct {
	URL   string `json:"url"`
	Label string `json:"label,omitempty"`
}

type ImageGenerationResponse struct {
	URL      string         `json:"url"`
	URLs     []string       `json:"urls,omitempty"`
	TaskID   string         `json:"task_id,omitempty"`
	Provider string         `json:"provider"`
	Model    string         `json:"model"`
	Size     string         `json:"size,omitempty"`
	Status   string         `json:"status,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type GetImageTaskResponse struct {
	TaskID   string         `json:"task_id"`
	Provider string         `json:"provider"`
	Model    string         `json:"model"`
	Status   string         `json:"status"`
	URL      *string        `json:"url"`
	URLs     []string       `json:"urls,omitempty"`
	Error    *TaskError     `json:"error"`
	Metadata map[string]any `json:"metadata"`
}
