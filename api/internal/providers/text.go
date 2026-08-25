package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type Suggestion struct {
	Translation    string   `json:"translation"`
	Reformulations []string `json:"reformulations"`
	Related        []string `json:"related"`
}

type TextRequest struct {
	SourceText string
	SourceLang string
	TargetLang string
	Model      string
}

type TextProvider interface {
	Complete(ctx context.Context, req TextRequest) (Suggestion, error)
}

func SystemPrompt(sourceLang, targetLang string) string {
	src := langName(sourceLang)
	dst := langName(targetLang)
	return fmt.Sprintf(`You are a language tutor. Translate the user's sentence from %s to %s.
Return a JSON object only, no markdown, with this exact shape:
{"translation":"...","reformulations":["...","...","..."],"related":["...","...","..."]}

Rules:
- translation: a natural %s sentence a native speaker would say. It MUST be written in %s, never in %s. Do not copy the source. Do not append language codes such as (vi) or (fr).
- reformulations: 3 other ways to say the same idea in %s (register, word choice, or structure).
- related: 3 nearby useful sentences in %s for the same situation (follow-ups, answers, or vocabulary around the scene).
- Keep each line short enough to speak aloud while walking.`, src, dst, dst, dst, src, dst, dst)
}

// CanonicalTextModel maps marketing names and aliases to API model ids.
func CanonicalTextModel(provider, model string) string {
	model = strings.TrimSpace(model)
	if !strings.EqualFold(provider, "mistral") {
		return model
	}
	switch strings.ToLower(model) {
	case "", "mistral-small-4", "mistral-small", "small-4":
		return "mistral-small-latest"
	default:
		return model
	}
}

func langName(code string) string {
	switch strings.ToLower(code) {
	case "fr":
		return "French"
	case "vi":
		return "Vietnamese"
	default:
		return code
	}
}

func ParseSuggestion(raw string) (Suggestion, error) {
	raw = strings.TrimSpace(raw)
	if i := strings.Index(raw, "{"); i >= 0 {
		if j := strings.LastIndex(raw, "}"); j > i {
			raw = raw[i : j+1]
		}
	}
	var s Suggestion
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return Suggestion{}, fmt.Errorf("parse suggestion json: %w", err)
	}
	if strings.TrimSpace(s.Translation) == "" {
		return Suggestion{}, fmt.Errorf("empty translation")
	}
	s.Reformulations = trimList(s.Reformulations, 5)
	s.Related = trimList(s.Related, 5)
	return s, nil
}

func trimList(in []string, n int) []string {
	out := make([]string, 0, n)
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		out = append(out, s)
		if len(out) >= n {
			break
		}
	}
	return out
}
