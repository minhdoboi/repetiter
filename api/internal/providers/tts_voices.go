package providers

import (
	"strings"

	"repetiter/internal/config"
)

var (
	openaiTTSVoices = map[string]bool{
		"alloy": true, "ash": true, "ballad": true, "coral": true, "echo": true,
		"fable": true, "nova": true, "onyx": true, "sage": true, "shimmer": true, "verse": true,
	}
	groqTTSVoices = map[string]bool{
		"austin": true, "troy": true, "hannah": true, "autumn": true, "daniel": true,
	}
)

func PresetTTSVoices(provider string) []TTSVoiceOption {
	switch strings.ToLower(provider) {
	case "openai":
		return []TTSVoiceOption{
			{ID: "alloy", Label: "Alloy"},
			{ID: "echo", Label: "Echo"},
			{ID: "fable", Label: "Fable"},
			{ID: "nova", Label: "Nova"},
			{ID: "onyx", Label: "Onyx"},
			{ID: "shimmer", Label: "Shimmer"},
		}
	case "groq":
		return []TTSVoiceOption{
			{ID: "austin", Label: "Austin"},
			{ID: "troy", Label: "Troy"},
			{ID: "hannah", Label: "Hannah"},
		}
	case "google":
		return []TTSVoiceOption{
			{ID: "normal", Label: "Normal"},
			{ID: "slow", Label: "Slow (clearer)"},
		}
	default:
		return nil
	}
}

func DefaultTTSVoice(provider string, cfg config.Config) string {
	switch strings.ToLower(provider) {
	case "groq":
		return "austin"
	case "elevenlabs":
		return cfg.ElevenLabsVoice
	case "openai":
		return "alloy"
	case "google":
		return "normal"
	case "mistral":
		return ""
	default:
		if cfg.DefaultTTSVoice != "" {
			return cfg.DefaultTTSVoice
		}
		return "alloy"
	}
}

func ValidTTSVoice(provider, voice string) bool {
	voice = strings.TrimSpace(voice)
	if voice == "" {
		return false
	}
	switch strings.ToLower(provider) {
	case "openai":
		return openaiTTSVoices[voice]
	case "groq":
		return groqTTSVoices[voice]
	case "google":
		return googleTTSVoices[voice]
	case "mistral":
		return validMistralVoice(voice)
	case "elevenlabs":
		return !openaiTTSVoices[voice] && !groqTTSVoices[voice] && !validMistralVoice(voice)
	default:
		return true
	}
}

func validMistralVoice(voice string) bool {
	if openaiTTSVoices[voice] || groqTTSVoices[voice] {
		return false
	}
	if isLikelyElevenLabsVoiceID(voice) {
		return false
	}
	return len(voice) >= 3
}

func isLikelyElevenLabsVoiceID(voice string) bool {
	if len(voice) < 15 {
		return false
	}
	for _, c := range voice {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			continue
		}
		return false
	}
	return true
}

func ResolveTTSVoice(provider, voice string, cfg config.Config) string {
	if ValidTTSVoice(provider, voice) {
		return voice
	}
	return DefaultTTSVoice(provider, cfg)
}
