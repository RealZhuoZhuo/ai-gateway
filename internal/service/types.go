package service

type ImageGenerationRequest struct {
	Model                     string
	Prompt                    string
	NegativePrompt            string
	Size                      string
	Resolution                string
	AspectRatio               string
	N                         *int
	ReferenceImages           []ReferenceImage
	Image                     string
	Images                    []string
	Quality                   string
	Format                    string
	SequentialImageGeneration string
	ResponseFormat            string
	Stream                    *bool
	Watermark                 *bool
	Async                     *bool
	Input                     map[string]any
	Parameters                map[string]any
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
	Model          string
	Prompt         string
	Content        []VideoContent
	Media          []VideoMedia
	NegativePrompt string
	FirstFrameURL  string
	Image          string
	ImageTail      string
	Resolution     string
	Ratio          string
	AspectRatio    string
	Duration       *int
	Seed           *int64
	GenerateAudio  *bool
	Watermark      *bool
	Input          map[string]any
	Parameters     map[string]any
}

type VideoContent struct {
	Type     string
	Text     string
	ImageURL *MediaURL
	VideoURL *MediaURL
	AudioURL *MediaURL
	Role     string
}

type MediaURL struct {
	URL string
}

type VideoMedia struct {
	Type           string
	URL            string
	ReferenceVoice string
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
