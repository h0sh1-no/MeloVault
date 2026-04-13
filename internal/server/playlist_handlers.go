package server

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/h0sh1-no/MeloVault/internal/ipgeo"
	"github.com/h0sh1-no/MeloVault/internal/playlist"
)

func (s *Server) handlePlaylistsRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListMyPlaylists(w, r)
	case http.MethodPost:
		s.handleCreatePlaylist(w, r)
	default:
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
	}
}

func (s *Server) handlePlaylistDetailRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/playlists/")

	if strings.Contains(path, "/songs") {
		parts := strings.SplitN(path, "/songs", 2)
		playlistID, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil || playlistID == 0 {
			s.writeAPIError(w, "无效的歌单ID", http.StatusBadRequest, "")
			return
		}

		songPart := ""
		if len(parts) > 1 {
			songPart = strings.TrimPrefix(parts[1], "/")
		}

		switch r.Method {
		case http.MethodPost:
			s.handleAddSongToPlaylist(w, r, playlistID)
		case http.MethodDelete:
			if songPart == "" {
				s.writeAPIError(w, "缺少歌曲ID", http.StatusBadRequest, "")
				return
			}
			songID, err := strconv.ParseInt(songPart, 10, 64)
			if err != nil || songID == 0 {
				s.writeAPIError(w, "无效的歌曲ID", http.StatusBadRequest, "")
				return
			}
			s.handleRemoveSongFromPlaylist(w, r, playlistID, songID)
		default:
			s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		}
		return
	}

	playlistID, err := strconv.ParseInt(path, 10, 64)
	if err != nil || playlistID == 0 {
		s.writeAPIError(w, "无效的歌单ID", http.StatusBadRequest, "")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetPlaylistDetail(w, r, playlistID)
	case http.MethodPut:
		s.handleUpdatePlaylist(w, r, playlistID)
	case http.MethodDelete:
		s.handleDeletePlaylist(w, r, playlistID)
	default:
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
	}
}

func (s *Server) handleCreatePlaylist(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		s.writeAPIError(w, "未认证", http.StatusUnauthorized, "")
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		CoverURL    string `json:"cover_url"`
	}
	if err := parseJSONBody(r, &req); err != nil {
		s.writeAPIError(w, "无效的请求数据", http.StatusBadRequest, "")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		s.writeAPIError(w, "歌单名称不能为空", http.StatusBadRequest, "")
		return
	}
	if len(req.Name) > 200 {
		s.writeAPIError(w, "歌单名称过长", http.StatusBadRequest, "")
		return
	}

	p, err := s.playlistSvc.Create(r.Context(), userID, req.Name, req.Description, req.CoverURL)
	if err != nil {
		if err == playlist.ErrLimitExceed {
			s.writeAPIError(w, "歌单数量已达上限", http.StatusBadRequest, "PLAYLIST_LIMIT")
			return
		}
		s.writeAPIError(w, "创建歌单失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}

	s.writeAPISuccess(w, p, "创建成功", http.StatusCreated)
}

func (s *Server) handleListMyPlaylists(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		s.writeAPIError(w, "未认证", http.StatusUnauthorized, "")
		return
	}

	page := parseIntParam(r, "page", 1)
	pageSize := parseIntParam(r, "page_size", 50)

	playlists, total, err := s.playlistSvc.ListByUser(r.Context(), userID, page, pageSize)
	if err != nil {
		s.writeAPIError(w, "获取歌单列表失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}

	s.writeAPISuccess(w, map[string]any{
		"list":      playlists,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}, "获取成功", http.StatusOK)
}

func (s *Server) handleGetPlaylistDetail(w http.ResponseWriter, r *http.Request, playlistID int64) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		s.writeAPIError(w, "未认证", http.StatusUnauthorized, "")
		return
	}

	p, err := s.playlistSvc.GetByID(r.Context(), playlistID)
	if err != nil {
		s.writeAPIError(w, "歌单不存在", http.StatusNotFound, "")
		return
	}
	if p.UserID != userID {
		s.writeAPIError(w, "无权访问该歌单", http.StatusForbidden, "")
		return
	}

	songs, err := s.playlistSvc.ListSongs(r.Context(), playlistID, 0)
	if err != nil {
		s.writeAPIError(w, "获取歌曲列表失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}

	s.writeAPISuccess(w, map[string]any{
		"playlist": p,
		"songs":    songs,
	}, "获取成功", http.StatusOK)
}

func (s *Server) handleUpdatePlaylist(w http.ResponseWriter, r *http.Request, playlistID int64) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		s.writeAPIError(w, "未认证", http.StatusUnauthorized, "")
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		CoverURL    string `json:"cover_url"`
		IsPublic    *bool  `json:"is_public"`
	}
	if err := parseJSONBody(r, &req); err != nil {
		s.writeAPIError(w, "无效的请求数据", http.StatusBadRequest, "")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if len(req.Name) > 200 {
		s.writeAPIError(w, "歌单名称过长", http.StatusBadRequest, "")
		return
	}

	p, err := s.playlistSvc.Update(r.Context(), userID, playlistID, req.Name, req.Description, req.CoverURL, req.IsPublic)
	if err != nil {
		if err == playlist.ErrNotFound {
			s.writeAPIError(w, "歌单不存在", http.StatusNotFound, "")
			return
		}
		if err == playlist.ErrNotOwner {
			s.writeAPIError(w, "无权修改该歌单", http.StatusForbidden, "")
			return
		}
		s.writeAPIError(w, "更新歌单失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}

	s.writeAPISuccess(w, p, "更新成功", http.StatusOK)
}

func (s *Server) handleDeletePlaylist(w http.ResponseWriter, r *http.Request, playlistID int64) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		s.writeAPIError(w, "未认证", http.StatusUnauthorized, "")
		return
	}

	err := s.playlistSvc.Delete(r.Context(), userID, playlistID)
	if err != nil {
		if err == playlist.ErrNotFound {
			s.writeAPIError(w, "歌单不存在", http.StatusNotFound, "")
			return
		}
		if err == playlist.ErrNotOwner {
			s.writeAPIError(w, "无权删除该歌单", http.StatusForbidden, "")
			return
		}
		s.writeAPIError(w, "删除歌单失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}

	s.writeAPISuccess(w, nil, "删除成功", http.StatusOK)
}

func (s *Server) handleAddSongToPlaylist(w http.ResponseWriter, r *http.Request, playlistID int64) {
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

	song, err := s.playlistSvc.AddSong(r.Context(), userID, playlistID,
		req.SongID, strings.TrimSpace(req.SongName), strings.TrimSpace(req.Artists),
		strings.TrimSpace(req.Album), strings.TrimSpace(req.PicURL))
	if err != nil {
		if err == playlist.ErrNotFound {
			s.writeAPIError(w, "歌单不存在", http.StatusNotFound, "")
			return
		}
		if err == playlist.ErrNotOwner {
			s.writeAPIError(w, "无权操作该歌单", http.StatusForbidden, "")
			return
		}
		s.writeAPIError(w, "添加歌曲失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}

	s.writeAPISuccess(w, song, "添加成功", http.StatusCreated)
}

func (s *Server) handleRemoveSongFromPlaylist(w http.ResponseWriter, r *http.Request, playlistID, songID int64) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		s.writeAPIError(w, "未认证", http.StatusUnauthorized, "")
		return
	}

	err := s.playlistSvc.RemoveSong(r.Context(), userID, playlistID, songID)
	if err != nil {
		if err == playlist.ErrNotFound {
			s.writeAPIError(w, "歌单不存在", http.StatusNotFound, "")
			return
		}
		if err == playlist.ErrNotOwner {
			s.writeAPIError(w, "无权操作该歌单", http.StatusForbidden, "")
			return
		}
		if err == playlist.ErrSongMissing {
			s.writeAPIError(w, "歌曲不在歌单中", http.StatusNotFound, "")
			return
		}
		s.writeAPIError(w, "移除歌曲失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}

	s.writeAPISuccess(w, nil, "移除成功", http.StatusOK)
}

// handleSharedPlaylist serves a public shared playlist with optional auth.
// Authenticated users see the full song list; unauthenticated users see only the first 3 songs.
func (s *Server) handleSharedPlaylist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/shared/playlist/")
	playlistID, err := strconv.ParseInt(path, 10, 64)
	if err != nil || playlistID == 0 {
		s.writeAPIError(w, "无效的歌单ID", http.StatusBadRequest, "")
		return
	}

	p, err := s.playlistSvc.GetPublicByID(r.Context(), playlistID)
	if err != nil {
		s.writeAPIError(w, "歌单不存在或未公开", http.StatusNotFound, "")
		return
	}

	_, isLoggedIn := GetUserID(r.Context())

	var songs []playlist.Song
	if isLoggedIn {
		songs, err = s.playlistSvc.ListSongs(r.Context(), playlistID, 0)
	} else {
		songs, err = s.playlistSvc.ListSongs(r.Context(), playlistID, 3)
	}
	if err != nil {
		s.writeAPIError(w, "获取歌曲失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}

	// Log the share view activity
	sharer := r.URL.Query().Get("sharer")
	if s.activitySvc != nil {
		sharerID, _ := strconv.ParseInt(sharer, 10, 64)
		clientIP := ipgeo.ExtractIP(r)
		ua := r.UserAgent()
		var uid *int64
		if id, ok := GetUserID(r.Context()); ok {
			uid = &id
		}
		meta := map[string]any{"playlist_id": playlistID}
		if sharerID > 0 {
			meta["sharer_id"] = sharerID
		}
		go s.activitySvc.LogActivity(context.Background(), uid, "playlist_share_view", clientIP, ua, meta)
	}

	s.writeAPISuccess(w, map[string]any{
		"playlist":  p,
		"songs":     songs,
		"truncated": !isLoggedIn && p.SongCount > 3,
	}, "获取成功", http.StatusOK)
}
