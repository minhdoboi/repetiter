package httpapi

import (
	"log"
	"net/http"
	"strings"
)

func (s *Server) listTTSVoices(w http.ResponseWriter, r *http.Request) {
	provider := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("provider")))
	if provider == "" {
		writeError(w, http.StatusBadRequest, "provider is required")
		return
	}
	if !strings.EqualFold(provider, "mistral") && !strings.EqualFold(provider, "openai") && !strings.EqualFold(provider, "groq") && !strings.EqualFold(provider, "google") {
		writeError(w, http.StatusBadRequest, "voice listing is not available for this provider")
		return
	}

	voices, err := s.prov.ListTTSVoices(r.Context(), provider)
	if err != nil {
		log.Printf("list tts voices provider=%s: %v", provider, err)
		writeError(w, http.StatusBadGateway, "could not list voices: "+err.Error())
		return
	}
	log.Printf("list tts voices provider=%s -> %d voice(s)", provider, len(voices))
	writeJSON(w, http.StatusOK, map[string]any{"voices": voices})
}
