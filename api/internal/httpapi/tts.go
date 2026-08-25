package httpapi

import (
	"io"
	"log"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"repetiter/internal/providers"
)

type ttsReq struct {
	Text       string `json:"text"`
	Language   string `json:"language"`
	SentenceID string `json:"sentence_id"`
	VariantID  string `json:"variant_id"`
}

func (s *Server) tts(w http.ResponseWriter, r *http.Request) {
	var req ttsReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Text = strings.TrimSpace(req.Text)
	req.Language = strings.ToLower(strings.TrimSpace(req.Language))
	if req.Text == "" {
		writeError(w, http.StatusBadRequest, "text is required")
		return
	}
	if req.Language != "" && !allowedLangs[req.Language] {
		writeError(w, http.StatusBadRequest, "v1 only supports fr and vi")
		return
	}

	uid := userID(r)
	u, err := s.store.GetUser(r.Context(), uid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load account")
		return
	}
	usage, err := s.store.GetUsage(r.Context(), uid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load usage")
		return
	}
	secs := estimateSeconds(req.Text)
	if usage.TTSSeconds+secs > dailyTTSSecondCap {
		writeError(w, http.StatusTooManyRequests, "daily speech cap reached")
		return
	}

	tts, err := s.prov.TTS(u.TTSProvider)
	if err != nil || !providers.AllowedTTS(u.TTSProvider, u.TTSModel) {
		writeError(w, http.StatusBadRequest, "tts provider is not available")
		return
	}

	voice := providers.ResolveTTSVoice(u.TTSProvider, u.TTSVoice, s.cfg)
	if voice == "" && strings.EqualFold(u.TTSProvider, "mistral") {
		v, err := s.prov.DefaultMistralVoice(r.Context())
		if err != nil {
			writeError(w, http.StatusBadGateway, "could not load mistral voice: "+err.Error())
			return
		}
		voice = v
	}

	body, contentType, err := tts.Synthesize(r.Context(), providers.TTSRequest{
		Text:     req.Text,
		Language: req.Language,
		Model:    u.TTSModel,
		Voice:    voice,
	})
	if err != nil {
		log.Printf("tts synthesize failed provider=%s model=%s voice=%s: %v", u.TTSProvider, u.TTSModel, voice, err)
		writeError(w, http.StatusBadGateway, "speech failed: "+err.Error())
		return
	}
	defer body.Close()

	var sentenceID, variantID *uuid.UUID
	if req.SentenceID != "" {
		if id, err := uuid.Parse(req.SentenceID); err == nil {
			sentenceID = &id
		}
	}
	if req.VariantID != "" {
		if id, err := uuid.Parse(req.VariantID); err == nil {
			variantID = &id
		}
	}
	_ = s.store.InsertAudioJob(r.Context(), uid, sentenceID, variantID, u.TTSProvider, u.TTSModel, req.Language, req.Text)
	_ = s.store.IncrTTSSeconds(r.Context(), uid, secs)

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, body)
}

func estimateSeconds(text string) int {
	n := utf8.RuneCountInString(text)
	sec := n / 12
	if sec < 1 {
		return 1
	}
	return sec
}
