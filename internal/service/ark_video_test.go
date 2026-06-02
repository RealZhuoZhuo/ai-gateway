package service

import "testing"

func TestArkVideoContentSupportsReferenceMedia(t *testing.T) {
	content, err := arkVideoContent(CreateVideoTaskRequest{
		Content: []VideoContent{{
			Type: "text",
			Text: "prompt",
		}, {
			Type:     "image_url",
			ImageURL: &MediaURL{URL: "https://example.com/ref.jpg"},
			Role:     "reference_image",
		}, {
			Type:     "video_url",
			VideoURL: &MediaURL{URL: "https://example.com/ref.mp4"},
			Role:     "reference_video",
		}, {
			Type:     "audio_url",
			AudioURL: &MediaURL{URL: "https://example.com/ref.mp3"},
			Role:     "reference_audio",
		}},
	}, "fallback")
	if err != nil {
		t.Fatalf("arkVideoContent returned error: %v", err)
	}
	if len(content) != 4 {
		t.Fatalf("content = %#v", content)
	}
	if content[1].ImageURL == nil || content[1].ImageURL.URL != "https://example.com/ref.jpg" || content[1].Role != "reference_image" {
		t.Fatalf("image content = %#v", content[1])
	}
	if content[2].VideoURL == nil || content[2].VideoURL.URL != "https://example.com/ref.mp4" || content[2].Role != "reference_video" {
		t.Fatalf("video content = %#v", content[2])
	}
	if content[3].AudioURL == nil || content[3].AudioURL.URL != "https://example.com/ref.mp3" || content[3].Role != "reference_audio" {
		t.Fatalf("audio content = %#v", content[3])
	}
}

func TestArkVideoContentFromInputMap(t *testing.T) {
	in := CreateVideoTaskRequest{
		Input: map[string]any{
			"content": []any{
				map[string]any{
					"type": "text",
					"text": "prompt",
				},
				map[string]any{
					"type":      "video_url",
					"video_url": map[string]any{"url": "https://example.com/ref.mp4"},
					"role":      "reference_video",
				},
			},
		},
	}
	prompt := videoPrompt(in)
	if prompt != "prompt" {
		t.Fatalf("prompt = %q", prompt)
	}
	content, err := arkVideoContent(in, prompt)
	if err != nil {
		t.Fatalf("arkVideoContent returned error: %v", err)
	}
	if len(content) != 2 || content[1].VideoURL == nil || content[1].VideoURL.URL != "https://example.com/ref.mp4" {
		t.Fatalf("content = %#v", content)
	}
}

func TestArkVideoParameters(t *testing.T) {
	generateAudio := true
	watermark := false
	duration := 11
	seed := int64(42)
	in := CreateVideoTaskRequest{
		Ratio:         "16:9",
		Resolution:    "1080p",
		GenerateAudio: &generateAudio,
		Watermark:     &watermark,
		Duration:      &duration,
		Seed:          &seed,
	}
	if arkVideoGenerateAudio(in) == nil || *arkVideoGenerateAudio(in) != true {
		t.Fatalf("generate_audio = %#v", arkVideoGenerateAudio(in))
	}
	if arkVideoWatermark(in) == nil || *arkVideoWatermark(in) != false {
		t.Fatalf("watermark = %#v", arkVideoWatermark(in))
	}
	if arkVideoDuration(in) == nil || *arkVideoDuration(in) != 11 {
		t.Fatalf("duration = %#v", arkVideoDuration(in))
	}
	if arkVideoSeed(in) == nil || *arkVideoSeed(in) != 42 {
		t.Fatalf("seed = %#v", arkVideoSeed(in))
	}
	if arkVideoRatio(in) != "16:9" {
		t.Fatalf("ratio = %q", arkVideoRatio(in))
	}
	if arkVideoResolution(in) != "1080p" {
		t.Fatalf("resolution = %q", arkVideoResolution(in))
	}
}
