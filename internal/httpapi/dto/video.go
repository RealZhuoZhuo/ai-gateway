package dto

type CreateVideoTaskRequest struct {
	Model          string         `json:"model"`
	Prompt         string         `json:"prompt"`
	Content        []VideoContent `json:"content"`
	Media          []VideoMedia   `json:"media"`
	NegativePrompt string         `json:"negative_prompt"`
	FirstFrameURL  string         `json:"first_frame_url"`
	Image          string         `json:"image"`
	ImageTail      string         `json:"image_tail"`
	Resolution     string         `json:"resolution"`
	Ratio          string         `json:"ratio"`
	AspectRatio    string         `json:"aspect_ratio"`
	Duration       *int           `json:"duration"`
	Seed           *int64         `json:"seed"`
	GenerateAudio  *bool          `json:"generate_audio"`
	Watermark      *bool          `json:"watermark"`
	Input          map[string]any `json:"input"`
	Parameters     map[string]any `json:"parameters"`
}

type VideoContent struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *MediaURL `json:"image_url,omitempty"`
	VideoURL *MediaURL `json:"video_url,omitempty"`
	AudioURL *MediaURL `json:"audio_url,omitempty"`
	Role     string    `json:"role,omitempty"`
}

type MediaURL struct {
	URL string `json:"url"`
}

type VideoMedia struct {
	Type           string `json:"type,omitempty"`
	URL            string `json:"url,omitempty"`
	ReferenceVoice string `json:"reference_voice,omitempty"`
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
