package httpapi

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type createFolderReq struct {
	Name string `json:"name"`
}

type patchSentenceReq struct {
	FolderID *string `json:"folder_id"`
}

func (s *Server) listFolders(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListFolders(r.Context(), userID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list folders")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"folders": list})
}

func (s *Server) createFolder(w http.ResponseWriter, r *http.Request) {
	var req createFolderReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	f, err := s.store.CreateFolder(r.Context(), userID(r), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create folder")
		return
	}
	writeJSON(w, http.StatusCreated, f)
}

func (s *Server) deleteSentence(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.store.DeleteSentence(r.Context(), userID(r), id); err != nil {
		if isNotFound(err) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not delete sentence")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) clearHistory(w http.ResponseWriter, r *http.Request) {
	n, err := s.store.ClearHistory(r.Context(), userID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not clear history")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": n})
}

func (s *Server) patchSentence(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req patchSentenceReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	var folderID *uuid.UUID
	if req.FolderID != nil {
		if *req.FolderID == "" {
			folderID = nil
		} else {
			fid, err := uuid.Parse(*req.FolderID)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid folder_id")
				return
			}
			folderID = &fid
		}
	} else {
		writeError(w, http.StatusBadRequest, "folder_id is required")
		return
	}
	sent, err := s.store.SetSentenceFolder(r.Context(), userID(r), id, folderID)
	if err != nil {
		if isNotFound(err) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not update sentence")
		return
	}
	writeJSON(w, http.StatusOK, sent)
}
