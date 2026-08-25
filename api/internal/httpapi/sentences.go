package httpapi

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"repetiter/internal/providers"
	"repetiter/internal/store"
)

const (
	dailyTranslationCap = 50
	dailyTTSSecondCap   = 300
)

var allowedLangs = map[string]bool{"fr": true, "vi": true}

type createSentenceReq struct {
	SourceText string `json:"source_text"`
	SourceLang string `json:"source_lang"`
	TargetLang string `json:"target_lang"`
}

func (s *Server) createSentence(w http.ResponseWriter, r *http.Request) {
	var req createSentenceReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.SourceText = strings.TrimSpace(req.SourceText)
	req.SourceLang = strings.ToLower(strings.TrimSpace(req.SourceLang))
	req.TargetLang = strings.ToLower(strings.TrimSpace(req.TargetLang))
	if req.SourceText == "" {
		writeError(w, http.StatusBadRequest, "source_text is required")
		return
	}
	if utf8.RuneCountInString(req.SourceText) > 2000 {
		writeError(w, http.StatusBadRequest, "sentence is too long")
		return
	}
	if !allowedLangs[req.SourceLang] || !allowedLangs[req.TargetLang] {
		writeError(w, http.StatusBadRequest, "v1 only supports fr and vi")
		return
	}
	if req.SourceLang == req.TargetLang {
		writeError(w, http.StatusBadRequest, "source and target must differ")
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
	if usage.Translations >= dailyTranslationCap {
		writeError(w, http.StatusTooManyRequests, "daily translation cap reached")
		return
	}

	text, err := s.prov.Text(u.TextProvider)
	if err != nil || !providers.AllowedText(u.TextProvider, u.TextModel) {
		writeError(w, http.StatusBadRequest, "text provider is not available")
		return
	}
	sug, err := text.Complete(r.Context(), providers.TextRequest{
		SourceText: req.SourceText,
		SourceLang: req.SourceLang,
		TargetLang: req.TargetLang,
		Model:      providers.CanonicalTextModel(u.TextProvider, u.TextModel),
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, "translation failed: "+err.Error())
		return
	}

	sent, err := s.store.InsertSentence(r.Context(), store.Sentence{
		UserID:      uid,
		SourceText:  req.SourceText,
		Translation: sug.Translation,
		SourceLang:  req.SourceLang,
		TargetLang:  req.TargetLang,
	}, sug.Reformulations, sug.Related)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not save sentence")
		return
	}
	_ = s.store.IncrTranslations(r.Context(), uid)
	writeJSON(w, http.StatusCreated, sent)
}

func (s *Server) listSentences(w http.ResponseWriter, r *http.Request) {
	scope := strings.TrimSpace(r.URL.Query().Get("scope"))
	if scope == "" {
		scope = "history"
	}
	var folderID *uuid.UUID
	if fid := strings.TrimSpace(r.URL.Query().Get("folder_id")); fid != "" {
		id, err := uuid.Parse(fid)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid folder_id")
			return
		}
		folderID = &id
	}
	list, err := s.store.ListSentences(r.Context(), userID(r), 50, scope, folderID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list sentences")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sentences": list})
}

func (s *Server) getSentence(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	sent, err := s.store.GetSentence(r.Context(), userID(r), id)
	if err != nil {
		if isNotFound(err) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load sentence")
		return
	}
	writeJSON(w, http.StatusOK, sent)
}

func (s *Server) getMe(w http.ResponseWriter, r *http.Request) {
	u, err := s.store.GetUser(r.Context(), userID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load account")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

type patchMeReq struct {
	SourceLang   string `json:"source_lang"`
	TargetLang   string `json:"target_lang"`
	TextProvider string `json:"text_provider"`
	TextModel    string `json:"text_model"`
	TTSProvider  string `json:"tts_provider"`
	TTSModel     string `json:"tts_model"`
	TTSVoice     string `json:"tts_voice"`
}

func (s *Server) patchMe(w http.ResponseWriter, r *http.Request) {
	var req patchMeReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.TextProvider != "" && req.TextModel != "" && !providers.AllowedText(req.TextProvider, req.TextModel) {
		writeError(w, http.StatusBadRequest, "unknown text provider or model")
		return
	}
	if req.TTSProvider != "" && req.TTSModel != "" && !providers.AllowedTTS(req.TTSProvider, req.TTSModel) {
		writeError(w, http.StatusBadRequest, "unknown tts provider or model")
		return
	}
	if req.TTSVoice != "" {
		prov := req.TTSProvider
		if prov == "" {
			cur, err := s.store.GetUser(r.Context(), userID(r))
			if err != nil {
				writeError(w, http.StatusInternalServerError, "could not load account")
				return
			}
			prov = cur.TTSProvider
		}
		if !providers.ValidTTSVoice(prov, req.TTSVoice) {
			writeError(w, http.StatusBadRequest, "unknown voice for tts provider")
			return
		}
	}
	if req.SourceLang != "" && !allowedLangs[strings.ToLower(req.SourceLang)] {
		writeError(w, http.StatusBadRequest, "v1 only supports fr and vi")
		return
	}
	if req.TargetLang != "" && !allowedLangs[strings.ToLower(req.TargetLang)] {
		writeError(w, http.StatusBadRequest, "v1 only supports fr and vi")
		return
	}

	ttsVoice := req.TTSVoice
	if req.TTSProvider != "" && (ttsVoice == "" || !providers.ValidTTSVoice(req.TTSProvider, ttsVoice)) {
		if strings.EqualFold(req.TTSProvider, "mistral") {
			v, err := s.prov.DefaultMistralVoice(r.Context())
			if err != nil {
				writeError(w, http.StatusBadGateway, "could not load mistral voices: "+err.Error())
				return
			}
			ttsVoice = v
		} else {
			ttsVoice = providers.DefaultTTSVoice(req.TTSProvider, s.cfg)
		}
	}

	u, err := s.store.UpdateUser(r.Context(), userID(r), store.User{
		SourceLang:   strings.ToLower(req.SourceLang),
		TargetLang:   strings.ToLower(req.TargetLang),
		TextProvider: strings.ToLower(req.TextProvider),
		TextModel:    req.TextModel,
		TTSProvider:  strings.ToLower(req.TTSProvider),
		TTSModel:     req.TTSModel,
		TTSVoice:     ttsVoice,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update account")
		return
	}
	writeJSON(w, http.StatusOK, u)
}
