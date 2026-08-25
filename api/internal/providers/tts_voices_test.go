package providers

import (
	"testing"

	"repetiter/internal/config"
)

func TestValidTTSVoice(t *testing.T) {
	tests := []struct {
		provider string
		voice    string
		want     bool
	}{
		{"mistral", "fr_marie_neutral", true},
		{"mistral", "GATds6kYPBX2tRfQExbR", false},
		{"elevenlabs", "GATds6kYPBX2tRfQExbR", true},
		{"elevenlabs", "alloy", false},
		{"openai", "alloy", true},
		{"openai", "fr_marie_neutral", false},
		{"groq", "austin", true},
	}
	for _, tc := range tests {
		if got := ValidTTSVoice(tc.provider, tc.voice); got != tc.want {
			t.Fatalf("ValidTTSVoice(%q, %q) = %v, want %v", tc.provider, tc.voice, got, tc.want)
		}
	}
}

func TestResolveTTSVoice(t *testing.T) {
	cfg := config.Config{ElevenLabsVoice: "voice-from-env"}
	got := ResolveTTSVoice("mistral", "GATds6kYPBX2tRfQExbR", cfg)
	if got != "" {
		t.Fatalf("ResolveTTSVoice = %q, want empty", got)
	}
}

func TestPickMistralDefaultVoice(t *testing.T) {
	voices := []TTSVoiceOption{
		{ID: "en_james_neutral", Label: "James"},
		{ID: "fr_marie_neutral", Label: "Marie"},
	}
	got := PickMistralDefaultVoice(voices)
	if got != "fr_marie_neutral" {
		t.Fatalf("PickMistralDefaultVoice = %q, want fr_marie_neutral", got)
	}
}

func TestMistralVoiceToOptionPrefersSlug(t *testing.T) {
	slug := "fr_marie_neutral"
	got := mistralVoiceToOption(mistralVoiceItem{
		ID:   "uuid-123",
		Name: "Marie - Neutral",
		Slug: &slug,
	})
	if got.ID != "fr_marie_neutral" || got.Label != "Marie - Neutral" {
		t.Fatalf("unexpected option: %+v", got)
	}
}

func TestMistralVoicesBaseURL(t *testing.T) {
	got := mistralVoicesBaseURL("https://api.eu.mistral.ai/v1")
	if got != "https://api.mistral.ai/v1" {
		t.Fatalf("mistralVoicesBaseURL = %q", got)
	}
	if mistralVoicesBaseURL("https://api.mistral.ai/v1") != "https://api.mistral.ai/v1" {
		t.Fatal("expected global URL unchanged")
	}
}
