package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/h0sh1-no/MeloVault/internal/auth"
	"github.com/h0sh1-no/MeloVault/internal/favorite"
)

// handleAddFavorite adds a song to favorites.
func (s *Server) handleAddFavorite(w http.ResponseWriter, r *http.Request) {
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
		Album    string `json:"album"`
		PicURL   string `json:"pic_url"`
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
	req.Album = strings.TrimSpace(req.Album)
	req.PicURL = strings.TrimSpace(req.PicURL)

	fav, err := s.favoriteSvc.Add(r.Context(), userID, req.SongID, req.SongName, req.Artists, req.Album, req.PicURL)
	if err != nil {
		s.writeAPIError(w, "收藏失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}

	s.writeAPISuccess(w, map[string]any{
		"id":         fav.ID,
		"song_id":    fav.SongID,
		"song_name":  fav.SongName,
		"artists":    fav.Artists,
		"album":      fav.Album,
		"pic_url":    fav.PicURL,
		"created_at": fav.CreatedAt,
	}, "收藏成功", http.StatusCreated)
}

// handleRemoveFavorite removes a song from favorites.
func (s *Server) handleRemoveFavorite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}

	userID, ok := GetUserID(r.Context())
	if !ok {
		s.writeAPIError(w, "未认证", http.StatusUnauthorized, "")
		return
	}

	// Extract song_id from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/favorites/")
	songID, err := strconv.ParseInt(path, 10, 64)
	if err != nil || songID == 0 {
		s.writeAPIError(w, "无效的歌曲ID", http.StatusBadRequest, "")
		return
	}

	if err := s.favoriteSvc.Remove(r.Context(), userID, songID); err != nil {
		if err == favorite.ErrNotFavorited {
			s.writeAPIError(w, "歌曲不在收藏列表中", http.StatusNotFound, "")
			return
		}
		s.writeAPIError(w, "取消收藏失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}

	s.writeAPISuccess(w, nil, "取消收藏成功", http.StatusOK)
}

// handleListFavorites lists user's favorites.
func (s *Server) handleListFavorites(w http.ResponseWriter, r *http.Request) {
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

	favorites, total, err := s.favoriteSvc.List(r.Context(), userID, page, pageSize)
	if err != nil {
		s.writeAPIError(w, "获取收藏列表失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}

	s.writeAPISuccess(w, map[string]any{
		"list":      favorites,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}, "获取收藏列表成功", http.StatusOK)
}

// handleCheckFavorite checks if a song is favorited.
func (s *Server) handleCheckFavorite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}

	userID, ok := GetUserID(r.Context())
	if !ok {
		s.writeAPIError(w, "未认证", http.StatusUnauthorized, "")
		return
	}

	// Extract song_id from URL path: /api/favorites/check/:song_id
	path := strings.TrimPrefix(r.URL.Path, "/api/favorites/check/")
	songID, err := strconv.ParseInt(path, 10, 64)
	if err != nil || songID == 0 {
		s.writeAPIError(w, "无效的歌曲ID", http.StatusBadRequest, "")
		return
	}

	isFavorited, err := s.favoriteSvc.IsFavorited(r.Context(), userID, songID)
	if err != nil {
		s.writeAPIError(w, "检查收藏状态失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}

	s.writeAPISuccess(w, map[string]any{
		"is_favorited": isFavorited,
		"song_id":      songID,
	}, "检查成功", http.StatusOK)
}

// handleBatchCheckFavorites checks multiple songs if they are favorited.
func (s *Server) handleBatchCheckFavorites(w http.ResponseWriter, r *http.Request) {
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
		SongIDs []int64 `json:"song_ids"`
	}

	if err := parseJSONBody(r, &req); err != nil {
		s.writeAPIError(w, "无效的请求数据", http.StatusBadRequest, "")
		return
	}

	if len(req.SongIDs) == 0 {
		s.writeAPISuccess(w, map[string]any{"favorites": map[int64]bool{}}, "获取成功", http.StatusOK)
		return
	}

	if len(req.SongIDs) > 100 {
		s.writeAPIError(w, "最多检查100首歌曲", http.StatusBadRequest, "")
		return
	}

	favorites, err := s.favoriteSvc.GetBySongIDs(r.Context(), userID, req.SongIDs)
	if err != nil {
		s.writeAPIError(w, "检查收藏状态失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}

	s.writeAPISuccess(w, map[string]any{
		"favorites": favorites,
	}, "检查成功", http.StatusOK)
}

// Ensure pgx ErrNoRows is accessible
var _ = pgx.ErrNoRows
var _ = auth.ErrUserNotFound
