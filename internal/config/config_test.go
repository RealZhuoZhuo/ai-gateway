package config

import "testing"

func TestNormalizeModelProvidersAcceptsYunwu(t *testing.T) {
	out, err := normalizeModelProviders("image_model_providers", []ModelProvider{{
		Model:    "gpt-image-2",
		Provider: "YUNWU",
	}})
	if err != nil {
		t.Fatalf("normalizeModelProviders returned error: %v", err)
	}
	if len(out) != 1 || out[0].Provider != "yunwu" {
		t.Fatalf("providers = %#v", out)
	}
}

func TestNormalizeModelProvidersRejectsUnknownProvider(t *testing.T) {
	_, err := normalizeModelProviders("image_model_providers", []ModelProvider{{
		Model:    "gpt-image-2",
		Provider: "unknown",
	}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNormalizeModelProvidersRejectsYunwuVideoProvider(t *testing.T) {
	_, err := normalizeModelProviders("video_model_providers", []ModelProvider{{
		Model:    "video-model",
		Provider: "yunwu",
	}})
	if err == nil {
		t.Fatal("expected error")
	}
}
