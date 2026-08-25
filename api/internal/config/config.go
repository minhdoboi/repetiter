package config

import (
	"os"
	"strings"
)

type Config struct {
	HTTPAddr         string
	DatabaseURL      string
	DevUserID        string
	OpenAIKey        string
	MistralKey       string
	GroqKey          string
	ElevenLabsKey    string
	ElevenLabsVoice  string
	MistralBaseURL   string
	DefaultTextProv  string
	DefaultTextModel string
	DefaultTTSProv   string
	DefaultTTSModel  string
	DefaultTTSVoice  string
}

func Load() Config {
	return Config{
		HTTPAddr:         env("HTTP_ADDR", ":8080"),
		DatabaseURL:      env("DATABASE_URL", "postgres://repetiter:repetiter@localhost:5432/repetiter?sslmode=disable"),
		DevUserID:        os.Getenv("DEV_USER_ID"),
		OpenAIKey:        os.Getenv("OPENAI_API_KEY"),
		MistralKey:       os.Getenv("MISTRAL_API_KEY"),
		GroqKey:          os.Getenv("GROQ_API_KEY"),
		ElevenLabsKey:    os.Getenv("ELEVENLABS_API_KEY"),
		ElevenLabsVoice:  env("ELEVENLABS_VOICE_ID", "JBFqnCBsd6RMkjVDRZzb"),
		MistralBaseURL:   strings.TrimRight(env("MISTRAL_BASE_URL", "https://api.eu.mistral.ai/v1"), "/"),
		DefaultTextProv:  env("DEFAULT_TEXT_PROVIDER", "openai"),
		DefaultTextModel: env("DEFAULT_TEXT_MODEL", "gpt-4o-mini"),
		DefaultTTSProv:   env("DEFAULT_TTS_PROVIDER", "openai"),
		DefaultTTSModel:  env("DEFAULT_TTS_MODEL", "gpt-4o-mini-tts"),
		DefaultTTSVoice:  env("DEFAULT_TTS_VOICE", "alloy"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
