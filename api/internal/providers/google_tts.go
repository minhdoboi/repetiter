package providers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

const googleTTSMaxChars = 200

var googleTTSVoices = map[string]bool{
	"normal": true,
	"slow":   true,
}

type GoogleTTS struct {
	Client *http.Client
}

func NewGoogleTTS() *GoogleTTS {
	return &GoogleTTS{
		Client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *GoogleTTS) Synthesize(ctx context.Context, req TTSRequest) (io.ReadCloser, string, error) {
	lang := strings.ToLower(strings.TrimSpace(req.Language))
	if lang == "" {
		lang = "fr"
	}
	slow := strings.EqualFold(req.Voice, "slow")

	var out bytes.Buffer
	for _, chunk := range chunkGoogleTTS(req.Text, googleTTSMaxChars) {
		audio, err := c.fetchChunk(ctx, chunk, lang, slow)
		if err != nil {
			return nil, "", err
		}
		out.Write(audio)
	}
	if out.Len() == 0 {
		return nil, "", fmt.Errorf("google tts: empty audio")
	}
	return io.NopCloser(bytes.NewReader(out.Bytes())), "audio/mpeg", nil
}

func (c *GoogleTTS) fetchChunk(ctx context.Context, text, lang string, slow bool) ([]byte, error) {
	q := url.Values{}
	q.Set("ie", "UTF-8")
	q.Set("client", "tw-ob")
	q.Set("tl", lang)
	q.Set("q", text)
	if slow {
		q.Set("ttsspeed", "0.24")
	} else {
		q.Set("ttsspeed", "1")
	}

	reqURL := "https://translate.google.com/translate_tts?" + q.Encode()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("User-Agent", "Mozilla/5.0 (compatible; repetiter/1.0)")

	resp, err := c.Client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("google tts %s: %s", resp.Status, truncate(raw, 400))
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("google tts: empty chunk response")
	}
	return raw, nil
}

func chunkGoogleTTS(text string, maxRunes int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if utf8.RuneCountInString(text) <= maxRunes {
		return []string{text}
	}

	var chunks []string
	for text != "" {
		if utf8.RuneCountInString(text) <= maxRunes {
			chunks = append(chunks, text)
			break
		}
		cut := runeIndexAt(text, maxRunes)
		splitAt := cut
		for i := cut; i > cut/2; i-- {
			if i >= len(text) {
				continue
			}
			switch text[i] {
			case ' ', '.', ',', ';', ':', '!', '?', '\n':
				splitAt = i + 1
				i = cut / 2
			}
		}
		chunks = append(chunks, strings.TrimSpace(text[:splitAt]))
		text = strings.TrimSpace(text[splitAt:])
	}
	return chunks
}

func runeIndexAt(s string, n int) int {
	i := 0
	for pos := range s {
		if i == n {
			return pos
		}
		i++
	}
	return len(s)
}
