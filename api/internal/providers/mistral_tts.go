package providers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type MistralTTS struct {
	APIKey  string
	BaseURL string
	Client  *http.Client
}

func NewMistralTTS(apiKey, baseURL string) *MistralTTS {
	return &MistralTTS{
		APIKey:  apiKey,
		BaseURL: strings.TrimRight(baseURL, "/"),
		Client:  &http.Client{Timeout: 90 * time.Second},
	}
}

type mistralSpeechResp struct {
	AudioData string `json:"audio_data"`
}

func (c *MistralTTS) Synthesize(ctx context.Context, req TTSRequest) (io.ReadCloser, string, error) {
	if c.APIKey == "" {
		return nil, "", fmt.Errorf("missing Mistral API key")
	}
	model := req.Model
	if model == "" {
		model = "voxtral-mini-tts-2603"
	}
	voice := req.Voice
	if voice == "" {
		return nil, "", fmt.Errorf("missing mistral voice_id")
	}
	body, err := json.Marshal(map[string]any{
		"model":           model,
		"input":           req.Text,
		"voice_id":        voice,
		"response_format": "mp3",
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
	httpReq.Header.Set("Accept", "application/json, audio/mpeg, audio/wav, audio/*")

	resp, err := c.Client.Do(httpReq)
	if err != nil {
		return nil, "", err
	}
	raw, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("mistral tts %s: %s", resp.Status, truncate(raw, 400))
	}

	audio, ct, err := decodeMistralSpeechBody(raw, resp.Header.Get("Content-Type"))
	if err != nil {
		return nil, "", err
	}
	return io.NopCloser(bytes.NewReader(audio)), ct, nil
}

func decodeMistralSpeechBody(raw []byte, contentType string) ([]byte, string, error) {
	if len(raw) == 0 {
		return nil, "", fmt.Errorf("mistral tts: empty response body")
	}
	if strings.Contains(contentType, "json") || raw[0] == '{' {
		var parsed mistralSpeechResp
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return nil, "", fmt.Errorf("mistral tts: invalid json: %s", truncate(raw, 200))
		}
		if parsed.AudioData == "" {
			return nil, "", fmt.Errorf("mistral tts: missing audio_data")
		}
		decoded, err := base64.StdEncoding.DecodeString(parsed.AudioData)
		if err != nil {
			return nil, "", fmt.Errorf("mistral tts: decode audio_data: %w", err)
		}
		return decoded, sniffAudioContentType(decoded), nil
	}
	return raw, sniffAudioContentType(raw), nil
}

func sniffAudioContentType(data []byte) string {
	if len(data) >= 4 && string(data[:4]) == "RIFF" {
		return "audio/wav"
	}
	if len(data) >= 3 && string(data[:3]) == "ID3" {
		return "audio/mpeg"
	}
	if len(data) >= 2 && data[0] == 0xff && (data[1]&0xe0) == 0xe0 {
		return "audio/mpeg"
	}
	return "audio/mpeg"
}
