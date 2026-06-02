package service

import (
	"testing"

	"github.com/RealZhuoZhuo/ai-gateway/internal/providers"
)

func TestYunwuGeminiImageURLs(t *testing.T) {
	out := providers.YunwuGeminiResponse{
		Candidates: []providers.YunwuGeminiCandidate{{
			Content: providers.YunwuGeminiContent{
				Parts: []providers.YunwuGeminiPart{{
					Text: "done",
				}, {
					InlineData: &providers.YunwuGeminiInlineData{
						MimeType: "image/jpeg",
						Data:     "abc123",
					},
				}},
			},
		}},
	}

	urls := yunwuGeminiImageURLs(out)
	if len(urls) != 1 || urls[0] != "data:image/jpeg;base64,abc123" {
		t.Fatalf("urls = %#v", urls)
	}
}

func TestYunwuGeminiPartsIncludesReferenceImages(t *testing.T) {
	parts, err := yunwuGeminiParts(ImageGenerationRequest{
		Images: []string{
			"https://cdn.example/ref.jpg",
			"data:image/png;base64,abc123",
		},
	}, "draw")
	if err != nil {
		t.Fatalf("yunwuGeminiParts returned error: %v", err)
	}
	if len(parts) != 3 {
		t.Fatalf("parts = %#v", parts)
	}
	if parts[0].Text != "draw" {
		t.Fatalf("text part = %#v", parts[0])
	}
	if parts[1].FileData == nil || parts[1].FileData.FileURI != "https://cdn.example/ref.jpg" || parts[1].FileData.MimeType != "image/jpeg" {
		t.Fatalf("file part = %#v", parts[1])
	}
	if parts[2].InlineData == nil || parts[2].InlineData.Data != "abc123" || parts[2].InlineData.MimeType != "image/png" {
		t.Fatalf("inline part = %#v", parts[2])
	}
}

func TestYunwuImageURLs(t *testing.T) {
	out := providers.YunwuImageResponse{
		Data: []providers.YunwuImageData{{
			URL: "https://cdn.example/image.jpg",
		}, {
			B64JSON:  "xyz",
			MimeType: "image/jpeg",
		}, {
			B64JSON: "pngdata",
		}},
	}

	urls := yunwuImageURLs(out, "jpeg")
	want := []string{
		"https://cdn.example/image.jpg",
		"data:image/jpeg;base64,xyz",
		"data:image/jpeg;base64,pngdata",
	}
	if len(urls) != len(want) {
		t.Fatalf("urls = %#v", urls)
	}
	for i := range want {
		if urls[i] != want[i] {
			t.Fatalf("urls[%d] = %q, want %q", i, urls[i], want[i])
		}
	}
}
