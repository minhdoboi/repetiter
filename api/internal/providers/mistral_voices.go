package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type TTSVoiceOption struct {
	ID        string   `json:"id"`
	Label     string   `json:"label"`
	Languages []string `json:"languages,omitempty"`
}

type mistralVoicesResp struct {
	Items []mistralVoiceItem `json:"items"`
	Total int                `json:"total"`
}

type mistralVoiceItem struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Slug      *string  `json:"slug"`
	Languages []string `json:"languages"`
}

func ListMistralVoices(ctx context.Context, apiKey, baseURL string) ([]TTSVoiceOption, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("missing Mistral API key")
	}
	base := mistralVoicesBaseURL(baseURL)
	if base != strings.TrimRight(baseURL, "/") {
		log.Printf("mistral voices: using global endpoint %s (audio/voices not on %s)", base, strings.TrimRight(baseURL, "/"))
	}
	client := &http.Client{Timeout: 30 * time.Second}

	var all []TTSVoiceOption
	offset := 0
	for {
		u, err := url.Parse(base + "/audio/voices")
		if err != nil {
			return nil, err
		}
		q := u.Query()
		q.Set("limit", "100")
		q.Set("offset", fmt.Sprintf("%d", offset))
		q.Set("type", "all")
		u.RawQuery = q.Encode()

		log.Printf("mistral voices: GET %s", u.String())

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("mistral voices: GET %s request failed: %v", u.String(), err)
			return nil, err
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= 300 {
			log.Printf("mistral voices: GET %s -> %s: %s", u.String(), resp.Status, truncate(body, 400))
			return nil, fmt.Errorf("mistral voices %s: %s", resp.Status, truncate(body, 400))
		}

		var parsed mistralVoicesResp
		if err := json.Unmarshal(body, &parsed); err != nil {
			log.Printf("mistral voices: GET %s decode failed: %v body=%s", u.String(), err, truncate(body, 200))
			return nil, err
		}
		log.Printf("mistral voices: GET %s -> %d item(s), total=%d offset=%d", u.String(), len(parsed.Items), parsed.Total, offset)
		if len(parsed.Items) == 0 {
			break
		}
		for _, item := range parsed.Items {
			all = append(all, mistralVoiceToOption(item))
		}
		offset += len(parsed.Items)
		if parsed.Total > 0 && offset >= parsed.Total {
			break
		}
		if len(parsed.Items) < 100 {
			break
		}
	}
	log.Printf("mistral voices: loaded %d voice(s)", len(all))
	return all, nil
}

// mistralVoicesBaseURL returns the base URL for GET /v1/audio/voices.
// The voices API is currently exposed on the global endpoint, not api.eu.mistral.ai.
func mistralVoicesBaseURL(baseURL string) string {
	base := strings.TrimRight(baseURL, "/")
	if strings.Contains(base, "api.eu.mistral.ai") {
		return "https://api.mistral.ai/v1"
	}
	return base
}

func mistralVoiceToOption(item mistralVoiceItem) TTSVoiceOption {
	id := item.ID
	if item.Slug != nil && strings.TrimSpace(*item.Slug) != "" {
		id = strings.TrimSpace(*item.Slug)
	}
	label := strings.TrimSpace(item.Name)
	if label == "" {
		label = id
	}
	return TTSVoiceOption{
		ID:        id,
		Label:     label,
		Languages: item.Languages,
	}
}

func PickMistralDefaultVoice(voices []TTSVoiceOption) string {
	for _, pref := range []string{"fr_", "vi_", "en_"} {
		for _, v := range voices {
			if strings.HasPrefix(v.ID, pref) {
				return v.ID
			}
		}
	}
	for _, v := range voices {
		for _, lang := range v.Languages {
			if strings.HasPrefix(strings.ToLower(lang), "fr") {
				return v.ID
			}
		}
	}
	if len(voices) > 0 {
		return voices[0].ID
	}
	return ""
}
