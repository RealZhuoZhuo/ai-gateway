package dto

import (
	"encoding/json"
	"strings"
)

type ImageGenerationRequest struct {
	Model                     string           `json:"model"`
	Prompt                    string           `json:"prompt"`
	NegativePrompt            string           `json:"negative_prompt"`
	Size                      string           `json:"size"`
	Resolution                string           `json:"resolution"`
	AspectRatio               string           `json:"aspect_ratio"`
	N                         *int             `json:"n"`
	ReferenceImages           []ReferenceImage `json:"reference_images"`
	Image                     FlexibleStrings  `json:"image"`
	SequentialImageGeneration string           `json:"sequential_image_generation"`
	ResponseFormat            string           `json:"response_format"`
	Stream                    *bool            `json:"stream"`
	Watermark                 *bool            `json:"watermark"`
	Async                     *bool            `json:"async"`
	Input                     map[string]any   `json:"input"`
	Parameters                map[string]any   `json:"parameters"`
}

type ReferenceImage struct {
	URL   string `json:"url"`
	Label string `json:"label,omitempty"`
}

type FlexibleStrings []string

func (v *FlexibleStrings) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*v = nil
		return nil
	}
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*v = compactStrings([]string{single})
		return nil
	}
	var many []string
	if err := json.Unmarshal(data, &many); err != nil {
		return err
	}
	*v = compactStrings(many)
	return nil
}

func (v FlexibleStrings) First() string {
	values := v.Values()
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (v FlexibleStrings) Values() []string {
	return compactStrings(v)
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
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
