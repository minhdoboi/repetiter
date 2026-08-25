package providers

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCanonicalTextModelMistralAliases(t *testing.T) {
	if got := CanonicalTextModel("mistral", "mistral-small-4"); got != "mistral-small-latest" {
		t.Fatalf("got %q", got)
	}
	if got := CanonicalTextModel("mistral", "mistral-small-latest"); got != "mistral-small-latest" {
		t.Fatalf("got %q", got)
	}
	if got := CanonicalTextModel("openai", "mistral-small-4"); got != "mistral-small-4" {
		t.Fatalf("left openai model alone: %q", got)
	}
}

func TestExtractChatContentStringAndChunks(t *testing.T) {
	got, err := extractChatContent(json.RawMessage(`"Xin chào"`))
	if err != nil || got != "Xin chào" {
		t.Fatalf("string: %q %v", got, err)
	}
	got, err = extractChatContent(json.RawMessage(`[{"type":"thinking","thinking":[{"type":"text","text":"plan"}]},{"type":"text","text":"{\"translation\":\"Xin chào\"}"}]`))
	if err != nil || !strings.Contains(got, "Xin chào") {
		t.Fatalf("chunks: %q %v", got, err)
	}
}

func TestParseSuggestionMarkdownFence(t *testing.T) {
	raw := "```json\n{\"translation\":\"Xin chào\",\"reformulations\":[\"Chào bạn\"],\"related\":[\"Bạn khỏe không?\"]}\n```"
	s, err := ParseSuggestion(raw)
	if err != nil {
		t.Fatal(err)
	}
	if s.Translation != "Xin chào" {
		t.Fatalf("translation: %q", s.Translation)
	}
	if len(s.Reformulations) != 1 || s.Reformulations[0] != "Chào bạn" {
		t.Fatalf("reformulations: %#v", s.Reformulations)
	}
}

func TestParseSuggestionEmpty(t *testing.T) {
	if _, err := ParseSuggestion(`{"translation":"","reformulations":[],"related":[]}`); err == nil {
		t.Fatal("expected error")
	}
}
