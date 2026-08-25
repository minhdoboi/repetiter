package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type TTSRequest struct {
	Text     string
	Language string
	Model    string
	Voice    string
}

type TTSProvider interface {
	Synthesize(ctx context.Context, req TTSRequest) (io.ReadCloser, string, error)
}

type OpenAITTS struct {
	APIKey         string
	BaseURL        string
	ResponseFormat string
	Client         *http.Client
}

func NewOpenAITTS(apiKey, baseURL string) *OpenAITTS {
	return &OpenAITTS{
		APIKey:         apiKey,
		BaseURL:        strings.TrimRight(baseURL, "/"),
		ResponseFormat: "mp3",
		Client:         &http.Client{Timeout: 90 * time.Second},
	}
}

func (c *OpenAITTS) Synthesize(ctx context.Context, req TTSRequest) (io.ReadCloser, string, error) {
	if c.APIKey == "" {
		return nil, "", fmt.Errorf("missing API key")
	}
	model := req.Model
	if model == "" {
		model = "gpt-4o-mini-tts"
	}
	voice := req.Voice
	if voice == "" {
		voice = "alloy"
	}
	format := c.ResponseFormat
	if format == "" {
		format = "mp3"
	}
	body, err := json.Marshal(map[string]any{
		"model":           model,
		"input":           req.Text,
		"voice":           voice,
		"response_format": format,
	})
	if err != nil {
		return nil, "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/audio/speech", bytes.NewReader(body))
	if err != nil {
		return nil, "", err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(httpReq)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, "", fmt.Errorf("tts %s: %s", resp.Status, truncate(b, 400))
	}
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		if format == "wav" {
			ct = "audio/wav"
		} else {
			ct = "audio/mpeg"
		}
	}
	return resp.Body, ct, nil
}

type ElevenLabsTTS struct {
	APIKey  string
	VoiceID string
	Client  *http.Client
}

func NewElevenLabsTTS(apiKey, voiceID string) *ElevenLabsTTS {
	if voiceID == "" {
		voiceID = "JBFqnCBsd6RMkjVDRZzb"
	}
	return &ElevenLabsTTS{
		APIKey:  apiKey,
		VoiceID: voiceID,
		Client:  &http.Client{Timeout: 90 * time.Second},
	}
}

func (c *ElevenLabsTTS) Synthesize(ctx context.Context, req TTSRequest) (io.ReadCloser, string, error) {
	if c.APIKey == "" {
		return nil, "", fmt.Errorf("missing ElevenLabs API key")
	}
	voice := req.Voice
	if voice == "" {
		voice = c.VoiceID
	}
	model := req.Model
	if model == "" {
		model = "eleven_flash_v2_5"
	}
	body, err := json.Marshal(map[string]any{
		"text":     req.Text,
		"model_id": model,
	})
	if err != nil {
		return nil, "", err
	}
	url := "https://api.elevenlabs.io/v1/text-to-speech/" + voice
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, "", err
	}
	httpReq.Header.Set("xi-api-key", c.APIKey)
	httpReq.Header.Set("Accept", "audio/mpeg")
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(httpReq)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, "", fmt.Errorf("elevenlabs %s: %s", resp.Status, truncate(b, 400))
	}
	return resp.Body, "audio/mpeg", nil
}
