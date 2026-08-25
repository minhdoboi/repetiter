package providers

import (
	"context"
	"fmt"
	"strings"

	"repetiter/internal/config"
)

type Registry struct {
	cfg ConfigView
}

type ConfigView struct {
	OpenAIKey       string
	MistralKey      string
	GroqKey         string
	ElevenLabsKey   string
	ElevenLabsVoice string
	MistralBaseURL  string
}

func NewRegistry(cfg config.Config) *Registry {
	return &Registry{cfg: ConfigView{
		OpenAIKey:       cfg.OpenAIKey,
		MistralKey:      cfg.MistralKey,
		GroqKey:         cfg.GroqKey,
		ElevenLabsKey:   cfg.ElevenLabsKey,
		ElevenLabsVoice: cfg.ElevenLabsVoice,
		MistralBaseURL:  cfg.MistralBaseURL,
	}}
}

func AllowedText(provider, model string) bool {
	switch strings.ToLower(provider) {
	case "openai", "mistral", "groq":
		return model != ""
	case "mock":
		return true
	default:
		return false
	}
}

func AllowedTTS(provider, model string) bool {
	switch strings.ToLower(provider) {
	case "openai", "mistral", "groq", "elevenlabs", "google":
		return model != ""
	case "mock":
		return true
	default:
		return false
	}
}

func (r *Registry) Text(provider string) (TextProvider, error) {
	switch strings.ToLower(provider) {
	case "openai":
		c := NewOpenAIChat(r.cfg.OpenAIKey, "https://api.openai.com/v1")
		c.Name = "openai"
		return c, nil
	case "mistral":
		c := NewOpenAIChat(r.cfg.MistralKey, r.cfg.MistralBaseURL)
		c.Name = "mistral"
		c.ReasoningEffort = "none"
		return c, nil
	case "groq":
		c := NewOpenAIChat(r.cfg.GroqKey, "https://api.groq.com/openai/v1")
		c.Name = "groq"
		return c, nil
	case "mock":
		return MockText{}, nil
	default:
		return nil, fmt.Errorf("unknown text provider %q", provider)
	}
}

func (r *Registry) TTS(provider string) (TTSProvider, error) {
	switch strings.ToLower(provider) {
	case "openai":
		return NewOpenAITTS(r.cfg.OpenAIKey, "https://api.openai.com/v1"), nil
	case "mistral":
		return NewMistralTTS(r.cfg.MistralKey, r.cfg.MistralBaseURL), nil
	case "groq":
		c := NewOpenAITTS(r.cfg.GroqKey, "https://api.groq.com/openai/v1")
		c.ResponseFormat = "wav"
		return c, nil
	case "elevenlabs":
		return NewElevenLabsTTS(r.cfg.ElevenLabsKey, r.cfg.ElevenLabsVoice), nil
	case "google":
		return NewGoogleTTS(), nil
	case "mock":
		return MockTTS{}, nil
	default:
		return nil, fmt.Errorf("unknown tts provider %q", provider)
	}
}

func (r *Registry) ListTTSVoices(ctx context.Context, provider string) ([]TTSVoiceOption, error) {
	switch strings.ToLower(provider) {
	case "openai", "groq", "google":
		return PresetTTSVoices(provider), nil
	case "mistral":
		return ListMistralVoices(ctx, r.cfg.MistralKey, r.cfg.MistralBaseURL)
	case "elevenlabs", "mock":
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown tts provider %q", provider)
	}
}

func (r *Registry) DefaultMistralVoice(ctx context.Context) (string, error) {
	voices, err := ListMistralVoices(ctx, r.cfg.MistralKey, r.cfg.MistralBaseURL)
	if err != nil {
		return "", err
	}
	v := PickMistralDefaultVoice(voices)
	if v == "" {
		return "", fmt.Errorf("no mistral voices available")
	}
	return v, nil
}
