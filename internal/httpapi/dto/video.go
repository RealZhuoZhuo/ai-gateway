package dto

type CreateVideoTaskRequest struct {
	Model           string         `json:"model"`
	Prompt          string         `json:"prompt"`
	NegativePrompt  string         `json:"negative_prompt"`
	FirstFrameURL   string         `json:"first_frame_url"`
	Image           string         `json:"image"`
	ImageTail       string         `json:"image_tail"`
	Resolution      string         `json:"resolution"`
	Ratio           string         `json:"ratio"`
	AspectRatio     string         `json:"aspect_ratio"`
	Duration        *int           `json:"duration"`
	Seed            *int64         `json:"seed"`
	GenerateAudio   *bool          `json:"generate_audio"`
	ReturnLastFrame *bool          `json:"return_last_frame"`
	Mode            string         `json:"mode"`
	Sound           string         `json:"sound"`
	CFGScale        *float64       `json:"cfg_scale"`
	StaticMask      any            `json:"static_mask"`
	DynamicMasks    any            `json:"dynamic_masks"`
	CameraControl   map[string]any `json:"camera_control"`
	CallbackURL     string         `json:"callback_url"`
	ExternalTaskID  string         `json:"external_task_id"`
	Input           map[string]any `json:"input"`
	Parameters      map[string]any `json:"parameters"`
}

type CreateVideoTaskResponse struct {
	TaskID   string `json:"task_id"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Status   string `json:"status"`
}

type GetVideoTaskResponse struct {
	TaskID       string         `json:"task_id"`
	Provider     string         `json:"provider"`
	Model        string         `json:"model"`
	Status       string         `json:"status"`
	VideoURL     *string        `json:"video_url"`
	LastFrameURL *string        `json:"last_frame_url"`
	Error        *TaskError     `json:"error"`
	Metadata     map[string]any `json:"metadata"`
}

type TaskError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
