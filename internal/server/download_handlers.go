package server

import (
	"net/http"
	"strconv"
	"strings"
)

// handleRecordDownload records a download in history.
func (s *Server) handleRecordDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}

	userID, ok := GetUserID(r.Context())
	if !ok {
		s.writeAPIError(w, "未认证", http.StatusUnauthorized, "")
		return
	}

	var req struct {
		SongID   int64  `json:"song_id"`
		SongName string `json:"song_name"`
		Artists  string `json:"artists"`
		Quality  string `json:"quality"`
		FileType string `json:"file_type"`
		FileSize int64  `json:"file_size"`
	}

	if err := parseJSONBody(r, &req); err != nil {
		s.writeAPIError(w, "无效的请求数据", http.StatusBadRequest, "")
		return
	}

	if req.SongID == 0 {
		s.writeAPIError(w, "歌曲ID不能为空", http.StatusBadRequest, "")
		return
	}

	req.SongName = strings.TrimSpace(req.SongName)
	req.Artists = strings.TrimSpace(req.Artists)
	req.Quality = strings.TrimSpace(req.Quality)
	req.FileType = strings.TrimSpace(req.FileType)

	if req.Quality == "" {
		req.Quality = "lossless"
	}
	if req.FileType == "" {
		req.FileType = "flac"
	}

	h, err := s.downloadSvc.Record(r.Context(), userID, req.SongID, req.SongName, req.Artists, req.Quality, req.FileType, req.FileSize)
	if err != nil {
		s.writeAPIError(w, "记录下载失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}

	s.writeAPISuccess(w, map[string]any{
		"id":         h.ID,
		"song_id":    h.SongID,
		"song_name":  h.SongName,
		"artists":    h.Artists,
		"quality":    h.Quality,
		"file_type":  h.FileType,
		"file_size":  h.FileSize,
		"created_at": h.CreatedAt,
	}, "记录成功", http.StatusCreated)
}

// handleListDownloads lists user's download history.
func (s *Server) handleListDownloads(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}

	userID, ok := GetUserID(r.Context())
	if !ok {
		s.writeAPIError(w, "未认证", http.StatusUnauthorized, "")
		return
	}

	page := parseIntParam(r, "page", 1)
	pageSize := parseIntParam(r, "page_size", 20)

	history, total, err := s.downloadSvc.List(r.Context(), userID, page, pageSize)
	if err != nil {
		s.writeAPIError(w, "获取下载历史失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}

	s.writeAPISuccess(w, map[string]any{
		"list":      history,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}, "获取下载历史成功", http.StatusOK)
}

// handleClearDownloads clears all download history for a user.
func (s *Server) handleClearDownloads(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}

	userID, ok := GetUserID(r.Context())
	if !ok {
		s.writeAPIError(w, "未认证", http.StatusUnauthorized, "")
		return
	}

	if err := s.downloadSvc.Clear(r.Context(), userID); err != nil {
		s.writeAPIError(w, "清空下载历史失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}

	s.writeAPISuccess(w, nil, "清空成功", http.StatusOK)
}

// handleDeleteDownload deletes a specific download history entry.
func (s *Server) handleDeleteDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}

	userID, ok := GetUserID(r.Context())
	if !ok {
		s.writeAPIError(w, "未认证", http.StatusUnauthorized, "")
		return
	}

	// Extract history_id from URL path: /api/downloads/:id
	path := strings.TrimPrefix(r.URL.Path, "/api/downloads/")
	historyID, err := strconv.ParseInt(path, 10, 64)
	if err != nil || historyID == 0 {
		s.writeAPIError(w, "无效的历史记录ID", http.StatusBadRequest, "")
		return
	}

	if err := s.downloadSvc.Delete(r.Context(), userID, historyID); err != nil {
		s.writeAPIError(w, "删除失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}

	s.writeAPISuccess(w, nil, "删除成功", http.StatusOK)
}
