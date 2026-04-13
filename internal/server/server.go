// Package server implements the HTTP server, routing, and request handlers.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/h0sh1-no/MeloVault/internal/accountpool"
	"github.com/h0sh1-no/MeloVault/internal/activity"
	"github.com/h0sh1-no/MeloVault/internal/admin"
	"github.com/h0sh1-no/MeloVault/internal/auth"
	"github.com/h0sh1-no/MeloVault/internal/config"
	"github.com/h0sh1-no/MeloVault/internal/cookie"
	"github.com/h0sh1-no/MeloVault/internal/database"
	"github.com/h0sh1-no/MeloVault/internal/download"
	"github.com/h0sh1-no/MeloVault/internal/favorite"
	"github.com/h0sh1-no/MeloVault/internal/ipgeo"
	"github.com/h0sh1-no/MeloVault/internal/legal"
	"github.com/h0sh1-no/MeloVault/internal/netease"
	"github.com/h0sh1-no/MeloVault/internal/playlist"
	"github.com/h0sh1-no/MeloVault/internal/settings"
	"github.com/h0sh1-no/MeloVault/internal/sitesettings"
)

var (
	validQualities = map[string]struct{}{
		"standard": {},
		"exhigh":   {},
		"lossless": {},
		"hires":    {},
		"sky":      {},
		"jyeffect": {},
		"jymaster": {},
	}

	validSongTypes = map[string]struct{}{
		"url":   {},
		"name":  {},
		"lyric": {},
		"json":  {},
	}

	idRegex = regexp.MustCompile(`\d{5,}`)
)

// Server contains all HTTP handlers and shared dependencies.
type Server struct {
	cfg             config.Config
	logger          *log.Logger
	startedAt       time.Time
	api             *netease.Client
	cookieManager   *cookie.Manager
	fileLocks       sync.Map
	runtimeCookieMu sync.RWMutex
	runtimeCookies  map[string]string
	db              *database.Pool
	auth            *auth.Service
	favoriteSvc     *favorite.Service
	downloadSvc     *download.Service
	adminSvc        *admin.Service
	settingsSvc     *settings.Service
	activitySvc     *activity.Service
	legalSvc        *legal.Service
	playlistSvc     *playlist.Service
	siteSettingsSvc *sitesettings.Service
	accountPool     *accountpool.Service
}

// New constructs a backend server.
func New(cfg config.Config, logger *log.Logger) *Server {
	return &Server{
		cfg:            cfg,
		logger:         logger,
		startedAt:      time.Now(),
		api:            netease.NewClient(),
		cookieManager:  cookie.NewManager(cfg.CookieFile),
		runtimeCookies: map[string]string{},
	}
}

// WithDB sets the database pool and initializes dependent services.
func (s *Server) WithDB(pool *database.Pool) *Server {
	s.db = pool
	if pool != nil {
		s.auth = auth.NewService(pool, auth.Config{
			JWTSecret:          s.cfg.JWTSecret,
			JWTAccessDuration:  s.cfg.JWTAccessDuration,
			JWTRefreshDuration: s.cfg.JWTRefreshDuration,
			FrontendURL:        s.cfg.FrontendURL,
			SMTPHost:           s.cfg.SMTPHost,
			SMTPPort:           s.cfg.SMTPPort,
			SMTPUser:           s.cfg.SMTPUser,
			SMTPPassword:       s.cfg.SMTPPassword,
		})
		s.favoriteSvc = favorite.NewService(pool)
		s.downloadSvc = download.NewService(pool)
		s.adminSvc = admin.NewService(pool)
		s.settingsSvc = settings.NewService(pool)
		s.activitySvc = activity.NewService(pool)
		s.legalSvc = legal.NewService(pool)
		s.playlistSvc = playlist.NewService(pool)
		s.siteSettingsSvc = sitesettings.NewService(pool)
		s.accountPool = accountpool.NewService(pool)

		// Load Netease Real IP from DB so it survives restarts.
		if feat, err := s.siteSettingsSvc.Get(context.Background()); err == nil {
			if feat.NeteaseRealIP != "" {
				s.api.SetRealIP(feat.NeteaseRealIP)
			}
			// Load SMTP config from DB so it survives restarts.
			if feat.SmtpConfigured() {
				s.auth.SetSMTPConfig(feat.SMTPHost, feat.SMTPPort, feat.SMTPUser, feat.SMTPPassword, feat.SMTPFrom)
			}
		}
	}
	return s
}

// Handler returns the root HTTP handler with middleware.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Public endpoints
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/song", s.handleSong)
	mux.HandleFunc("/Song_V1", s.handleSong)
	mux.HandleFunc("/api/stream", s.handleStream)
	// Public search for third-party sites (e.g. blogs); same handler as /api/search but no JWT.
	mux.HandleFunc("/api/public/search", s.handleSearch)
	mux.HandleFunc("/api/search", s.authMiddlewareFunc(s.handleSearch))
	mux.HandleFunc("/search", s.handleSPAOrSearch)
	mux.HandleFunc("/Search", s.handleSPAOrSearch)
	mux.Handle("/playlist", s.OptionalAuthMiddleware(http.HandlerFunc(s.handlePlaylist)))
	mux.Handle("/Playlist", s.OptionalAuthMiddleware(http.HandlerFunc(s.handlePlaylist)))
	mux.Handle("/album", s.OptionalAuthMiddleware(http.HandlerFunc(s.handleAlbum)))
	mux.Handle("/Album", s.OptionalAuthMiddleware(http.HandlerFunc(s.handleAlbum)))
	mux.Handle("/download", s.AuthMiddleware(http.HandlerFunc(s.handleDownload)))
	mux.Handle("/Download", s.AuthMiddleware(http.HandlerFunc(s.handleDownload)))
	mux.HandleFunc("/api/info", s.handleAPIInfo)

	// Setup endpoints (public, one-time)
	mux.HandleFunc("/api/setup/status", s.handleSetupStatus)
	mux.HandleFunc("/api/setup/init", s.handleSetupInit)

	// Legal documents (public)
	mux.HandleFunc("/api/legal/", s.handleGetLegalDocument)

	// Auth endpoints (public)
	mux.HandleFunc("/api/auth/register", s.handleRegister)
	mux.HandleFunc("/api/auth/login", s.handleLogin)
	mux.HandleFunc("/api/auth/linuxdo", s.handleLinuxdoLogin)
	mux.HandleFunc("/api/auth/linuxdo/callback", s.handleLinuxdoCallback)
	mux.HandleFunc("/api/auth/refresh", s.handleRefreshToken)
	mux.HandleFunc("/api/auth/send-code", s.handleSendCode)

	// Protected endpoints (require authentication)
	mux.Handle("/api/auth/me", s.AuthMiddleware(http.HandlerFunc(s.handleGetCurrentUser)))
	mux.Handle("/api/user/profile", s.AuthMiddleware(http.HandlerFunc(s.handleUpdateProfile)))
	mux.Handle("/api/user/password", s.AuthMiddleware(http.HandlerFunc(s.handleChangePassword)))

	// Favorites endpoints
	mux.HandleFunc("/api/favorites", s.authMiddlewareFunc(s.handleFavoritesRouter))
	mux.HandleFunc("/api/favorites/", s.authMiddlewareFunc(s.handleFavoritesDetailRouter))
	mux.HandleFunc("/api/favorites/check/", s.authMiddlewareFunc(s.handleCheckFavorite))
	mux.Handle("/api/favorites/batch-check", s.AuthMiddleware(http.HandlerFunc(s.handleBatchCheckFavorites)))

	// User playlists endpoints
	mux.HandleFunc("/api/playlists", s.authMiddlewareFunc(s.handlePlaylistsRouter))
	mux.HandleFunc("/api/playlists/", s.authMiddlewareFunc(s.handlePlaylistDetailRouter))

	// Shared playlist (public with optional auth)
	mux.Handle("/api/shared/playlist/", s.OptionalAuthMiddleware(http.HandlerFunc(s.handleSharedPlaylist)))

	// User settings endpoints
	mux.HandleFunc("/api/user/settings", s.authMiddlewareFunc(s.handleSettingsRouter))

	// Download history endpoints
	mux.HandleFunc("/api/downloads", s.authMiddlewareFunc(s.handleDownloadsRouter))
	mux.Handle("/api/downloads/", s.AuthMiddleware(http.HandlerFunc(s.handleDeleteDownload)))

	// Admin endpoints (require admin/superadmin role)
	mux.Handle("/api/admin/stats", s.AdminMiddleware(http.HandlerFunc(s.handleAdminStats)))
	mux.Handle("/api/admin/users", s.AdminMiddleware(http.HandlerFunc(s.handleAdminUsersRouter)))
	mux.Handle("/api/admin/users/", s.AdminMiddleware(http.HandlerFunc(s.handleAdminUserDetail)))
	mux.Handle("/api/admin/downloads", s.AdminMiddleware(http.HandlerFunc(s.handleAdminDownloads)))
	mux.Handle("/api/admin/netease/qr/key", s.AdminMiddleware(http.HandlerFunc(s.handleAdminNeteaseQRKey)))
	mux.Handle("/api/admin/netease/qr/check", s.AdminMiddleware(http.HandlerFunc(s.handleAdminNeteaseQRCheck)))
	mux.Handle("/api/admin/netease/cookie", s.AdminMiddleware(http.HandlerFunc(s.handleAdminNeteaseCookie)))
	mux.Handle("/api/admin/netease/accounts", s.AdminMiddleware(http.HandlerFunc(s.handleAdminNeteaseAccountsRouter)))
	mux.Handle("/api/admin/netease/accounts/", s.AdminMiddleware(http.HandlerFunc(s.handleAdminNeteaseAccountDetail)))

	// Admin legal documents
	mux.Handle("/api/admin/legal", s.AdminMiddleware(http.HandlerFunc(s.handleAdminLegalRouter)))

	// Site settings (public read, admin write)
	mux.HandleFunc("/api/site-settings", s.handleGetSiteSettings)
	mux.Handle("/api/admin/site-settings", s.AdminMiddleware(http.HandlerFunc(s.handleAdminSiteSettings)))
	mux.Handle("/api/admin/site-settings/test-email", s.AdminMiddleware(http.HandlerFunc(s.handleAdminTestEmail)))

	// Admin analytics endpoints
	mux.Handle("/api/admin/analytics/overview", s.AdminMiddleware(http.HandlerFunc(s.handleAnalyticsOverview)))
	mux.Handle("/api/admin/analytics/activity", s.AdminMiddleware(http.HandlerFunc(s.handleAnalyticsActivity)))
	mux.Handle("/api/admin/analytics/online", s.AdminMiddleware(http.HandlerFunc(s.handleAnalyticsOnline)))
	mux.Handle("/api/admin/analytics/provinces", s.AdminMiddleware(http.HandlerFunc(s.handleAnalyticsProvinces)))
	mux.Handle("/api/admin/analytics/trends", s.AdminMiddleware(http.HandlerFunc(s.handleAnalyticsTrends)))

	return s.withMiddleware(mux)
}

// authMiddlewareFunc wraps AuthMiddleware for use with http.HandlerFunc.
func (s *Server) authMiddlewareFunc(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.AuthMiddleware(http.HandlerFunc(handler)).ServeHTTP(w, r)
	}
}

// handleFavoritesRouter routes /api/favorites requests.
func (s *Server) handleFavoritesRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListFavorites(w, r)
	case http.MethodPost:
		s.handleAddFavorite(w, r)
	default:
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
	}
}

// handleFavoritesDetailRouter routes /api/favorites/:id requests.
func (s *Server) handleFavoritesDetailRouter(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		s.handleRemoveFavorite(w, r)
		return
	}
	s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
}

// handleDownloadsRouter routes /api/downloads requests.
func (s *Server) handleDownloadsRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListDownloads(w, r)
	case http.MethodPost:
		s.handleRecordDownload(w, r)
	case http.MethodDelete:
		s.handleClearDownloads(w, r)
	default:
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
	}
}

func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		s.setCORS(sw, r)
		if r.Method == http.MethodOptions {
			sw.WriteHeader(http.StatusNoContent)
			return
		}

		defer func() {
			if rec := recover(); rec != nil {
				s.logger.Printf("panic: %v", rec)
				s.writeAPIError(sw, "服务器内部错误", http.StatusInternalServerError, "")
			}
			s.logger.Printf("%s %s %d %s", r.Method, r.URL.Path, sw.statusCode, time.Since(start))
		}()

		next.ServeHTTP(sw, r)

		if s.activitySvc != nil && r.Method != http.MethodOptions {
			action := classifyAction(r.URL.Path)
			if action != "" {
				clientIP := ipgeo.ExtractIP(r)
				ua := r.UserAgent()
				var uid *int64
				if id, ok := GetUserID(r.Context()); ok {
					uid = &id
				}
				go s.activitySvc.LogActivity(context.Background(), uid, action, clientIP, ua, nil)
			}
		}
	})
}

func (s *Server) setCORS(w http.ResponseWriter, r *http.Request) {
	origin := s.cfg.CORSOrigins
	if s.siteSettingsSvc != nil {
		if f, err := s.siteSettingsSvc.Get(r.Context()); err == nil && f.SiteURL != "" {
			origin = f.SiteURL
		}
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	if origin != "*" {
		w.Header().Set("Vary", "Origin")
	}
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
	w.Header().Set("Access-Control-Max-Age", "3600")
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeAPIError(w, "请求参数错误", http.StatusBadRequest, "")
		return
	}

	if s.cfg.StaticDir != "" {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, filepath.Join(s.cfg.StaticDir, "index.html"))
			return
		}
		filePath := filepath.Join(s.cfg.StaticDir, filepath.Clean(r.URL.Path))
		if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
			http.ServeFile(w, r, filePath)
			return
		}
		// SPA fallback: serve index.html for unmatched routes
		http.ServeFile(w, r, filepath.Join(s.cfg.StaticDir, "index.html"))
		return
	}

	if r.URL.Path != "/" {
		s.writeAPIError(w, "请求的资源不存在", http.StatusNotFound, "")
		return
	}

	resp := map[string]any{
		"name":        "MeloVault 音乐服务",
		"version":     "3.0.0",
		"description": "Go backend is running with auth support",
	}
	s.writeAPISuccess(w, resp, "success", http.StatusOK)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeAPIError(w, "请求参数错误", http.StatusBadRequest, "")
		return
	}

	cookies := s.getCookies()
	downloadAbs, _ := filepath.Abs(s.cfg.DownloadsDir)
	health := map[string]any{
		"service":   "running",
		"timestamp": time.Now().Unix(),
		"cookie_status": func() string {
			if s.cookieManager.IsValid(cookies) {
				return "valid"
			}
			return "invalid"
		}(),
		"downloads_dir": downloadAbs,
		"version":       "3.0.0",
		"database": func() string {
			if s.db != nil {
				return "connected"
			}
			return "not configured"
		}(),
	}
	s.writeAPISuccess(w, health, "API服务运行正常", http.StatusOK)
}

func (s *Server) handleSong(w http.ResponseWriter, r *http.Request) {
	if !isMethodAllowed(r.Method) {
		s.writeAPIError(w, "请求参数错误", http.StatusBadRequest, "")
		return
	}

	data := parseRequestData(r)
	source := firstNonEmpty(data["ids"], data["id"], data["url"])
	level := firstNonEmpty(data["level"], "lossless")
	infoType := firstNonEmpty(data["type"], "url")

	if source == "" {
		s.writeAPIError(w, "必须提供 'ids'、'id' 或 'url' 参数", http.StatusBadRequest, "")
		return
	}
	if _, ok := validQualities[level]; !ok {
		s.writeAPIError(w, "无效的音质参数，支持: standard, exhigh, lossless, hires, sky, jyeffect, jymaster", http.StatusBadRequest, "")
		return
	}
	if _, ok := validSongTypes[infoType]; !ok {
		s.writeAPIError(w, "无效的类型参数，支持: url, name, lyric, json", http.StatusBadRequest, "")
		return
	}

	musicIDStr := s.extractMusicID(source)
	musicID, err := strconv.ParseInt(musicIDStr, 10, 64)
	if err != nil || musicID <= 0 {
		s.writeAPIError(w, "音乐ID格式无效", http.StatusBadRequest, "")
		return
	}
	ctx := r.Context()
	cookies := s.getCookies()

	switch infoType {
	case "url":
		songData, usedLevel, err := s.getPlayableSongURL(ctx, musicID, level, cookies)
		if err != nil {
			s.writeAPIError(w, "获取音乐URL失败，可能是版权限制或音质不支持", http.StatusNotFound, "")
			return
		}
		rawSongURL := normalizePlayableMediaURL(asString(songData["url"]))
		songURL := rawSongURL
		proxied := false
		if !isSecureBrowserMediaURL(rawSongURL) {
			songURL = buildStreamProxyURL(musicID, usedLevel)
			proxied = true
		}
		size := asInt64(songData["size"])
		resp := map[string]any{
			"id":              asInt64(songData["id"]),
			"url":             songURL,
			"source_url":      rawSongURL,
			"proxied":         proxied,
			"level":           usedLevel,
			"requested_level": level,
			"fallback":        usedLevel != level,
			"quality_name":    qualityDisplayName(usedLevel),
			"size":            size,
			"size_formatted":  formatFileSize(size),
			"type":            asString(songData["type"]),
			"bitrate":         asInt64(songData["br"]),
		}
		s.writeAPISuccess(w, resp, "获取歌曲URL成功", http.StatusOK)
	case "name":
		result, err := s.api.GetSongDetail(ctx, musicID)
		if err != nil {
			s.writeAPIError(w, "获取歌曲信息失败: "+err.Error(), http.StatusInternalServerError, "")
			return
		}
		s.writeAPISuccess(w, result, "获取歌曲信息成功", http.StatusOK)
	case "lyric":
		result, err := s.api.GetLyric(ctx, musicID, cookies)
		if err != nil {
			s.writeAPIError(w, "获取歌词失败: "+err.Error(), http.StatusInternalServerError, "")
			return
		}
		s.writeAPISuccess(w, result, "获取歌词成功", http.StatusOK)
	case "json":
		songInfo, err := s.api.GetSongDetail(ctx, musicID)
		if err != nil {
			s.writeAPIError(w, "未找到歌曲信息", http.StatusNotFound, "")
			return
		}
		lyricInfo, _ := s.api.GetLyric(ctx, musicID, cookies)

		songs := asSlice(songInfo["songs"])
		if len(songs) == 0 {
			s.writeAPIError(w, "未找到歌曲信息", http.StatusNotFound, "")
			return
		}

		songData := asMap(songs[0])
		al := asMap(songData["al"])
		resp := map[string]any{
			"id":              musicIDStr,
			"name":            asString(songData["name"]),
			"ar_name":         strings.Join(extractArtistNames(songData["ar"]), ", "),
			"al_name":         asString(al["name"]),
			"pic":             asString(al["picUrl"]),
			"level":           level,
			"requested_level": level,
			"fallback":        false,
			"lyric":           asString(asMap(lyricInfo["lrc"])["lyric"]),
			"tlyric":          asString(asMap(lyricInfo["tlyric"])["lyric"]),
		}
		urlData, usedLevel, urlErr := s.getPlayableSongURL(ctx, musicID, level, cookies)
		if urlErr == nil {
			resp["url"] = asString(urlData["url"])
			resp["size"] = formatFileSize(asInt64(urlData["size"]))
			resp["level"] = usedLevel
			resp["fallback"] = usedLevel != level
		} else {
			resp["url"] = ""
			resp["size"] = "获取失败"
		}

		s.writeAPISuccess(w, resp, "获取歌曲信息成功", http.StatusOK)
	}
}

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeAPIError(w, "请求参数错误", http.StatusBadRequest, "")
		return
	}

	data := parseRequestData(r)
	source := firstNonEmpty(data["ids"], data["id"], data["url"])
	level := firstNonEmpty(data["level"], data["quality"], "lossless")

	if source == "" {
		s.writeAPIError(w, "必须提供 'ids'、'id' 或 'url' 参数", http.StatusBadRequest, "")
		return
	}
	if _, ok := validQualities[level]; !ok {
		s.writeAPIError(w, "无效的音质参数，支持: standard, exhigh, lossless, hires, sky, jyeffect, jymaster", http.StatusBadRequest, "")
		return
	}

	musicIDStr := s.extractMusicID(source)
	musicID, err := strconv.ParseInt(musicIDStr, 10, 64)
	if err != nil || musicID <= 0 {
		s.writeAPIError(w, "音乐ID格式无效", http.StatusBadRequest, "")
		return
	}

	ctx := r.Context()
	cookies := s.getCookies()
	songData, _, err := s.getPlayableSongURL(ctx, musicID, level, cookies)
	if err != nil {
		s.writeAPIError(w, "获取音乐流失败，可能是版权限制或音质不支持", http.StatusNotFound, "")
		return
	}

	sourceURL := normalizePlayableMediaURL(asString(songData["url"]))
	if strings.TrimSpace(sourceURL) == "" {
		s.writeAPIError(w, "未找到可播放音频流", http.StatusNotFound, "")
		return
	}

	req, err := http.NewRequestWithContext(context.WithoutCancel(ctx), http.MethodGet, sourceURL, nil)
	if err != nil {
		s.writeAPIError(w, "音频流地址无效", http.StatusBadGateway, "")
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	copyRequestHeader(req.Header, r.Header, "Range")
	copyRequestHeader(req.Header, r.Header, "If-Range")
	copyRequestHeader(req.Header, r.Header, "Accept")
	copyRequestHeader(req.Header, r.Header, "Accept-Encoding")

	resp, err := s.api.HTTPClient().Do(req)
	if err != nil {
		s.writeAPIError(w, "拉取音频流失败", http.StatusBadGateway, "")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode != http.StatusPartialContent {
			s.writeAPIError(w, fmt.Sprintf("音频流请求失败: status=%d", resp.StatusCode), http.StatusBadGateway, "")
			return
		}
	}

	copyResponseHeader(w.Header(), resp.Header, "Accept-Ranges")
	copyResponseHeader(w.Header(), resp.Header, "Cache-Control")
	copyResponseHeader(w.Header(), resp.Header, "Content-Length")
	copyResponseHeader(w.Header(), resp.Header, "Content-Range")
	copyResponseHeader(w.Header(), resp.Header, "Content-Type")
	copyResponseHeader(w.Header(), resp.Header, "ETag")
	copyResponseHeader(w.Header(), resp.Header, "Expires")
	copyResponseHeader(w.Header(), resp.Header, "Last-Modified")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(resp.StatusCode)

	buffer := make([]byte, 32*1024)
	if _, err := io.CopyBuffer(w, resp.Body, buffer); err != nil {
		s.logger.Printf("stream proxy warning: %v", err)
	}
}

// handleSPAOrSearch serves the SPA page when StaticDir is configured (production),
// and falls back to the search API for backward compatibility in API-only mode.
func (s *Server) handleSPAOrSearch(w http.ResponseWriter, r *http.Request) {
	if s.cfg.StaticDir != "" {
		http.ServeFile(w, r, filepath.Join(s.cfg.StaticDir, "index.html"))
		return
	}
	s.handleSearch(w, r)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if !isMethodAllowed(r.Method) {
		s.writeAPIError(w, "请求参数错误", http.StatusBadRequest, "")
		return
	}

	data := parseRequestData(r)
	keyword := firstNonEmpty(data["keyword"], data["keywords"], data["q"])
	if strings.TrimSpace(keyword) == "" {
		s.writeAPIError(w, "参数 'keyword' 不能为空", http.StatusBadRequest, "")
		return
	}

	limit := 30
	if raw := strings.TrimSpace(data["limit"]); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			limit = n
		}
	}
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}

	result, err := s.api.SearchMusic(r.Context(), keyword, s.getCookies(), limit)
	if err != nil {
		s.writeAPIError(w, "搜索失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}
	for _, song := range result {
		if _, ok := song["artists"]; ok {
			song["artist_string"] = song["artists"]
		}
	}
	s.writeAPISuccess(w, result, "搜索完成", http.StatusOK)
}

func (s *Server) handlePlaylist(w http.ResponseWriter, r *http.Request) {
	if !isMethodAllowed(r.Method) {
		s.writeAPIError(w, "请求参数错误", http.StatusBadRequest, "")
		return
	}

	if s.siteSettingsSvc != nil {
		if blocked, msg := s.checkFeatureGate(r, "playlist"); blocked {
			s.writeAPIError(w, msg, http.StatusForbidden, "FEATURE_DISABLED")
			return
		}
	}

	data := parseRequestData(r)
	rawID := strings.TrimSpace(data["id"])
	if rawID == "" {
		s.writeAPIError(w, "参数 'playlist_id' 不能为空", http.StatusBadRequest, "")
		return
	}

	playlistID, err := strconv.ParseInt(s.extractMusicID(rawID), 10, 64)
	if err != nil || playlistID <= 0 {
		s.writeAPIError(w, "歌单ID格式无效", http.StatusBadRequest, "")
		return
	}

	result, err := s.api.GetPlaylistDetail(r.Context(), playlistID, s.getCookies())
	if err != nil {
		s.writeAPIError(w, "获取歌单失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}

	resp := map[string]any{
		"status":   "success",
		"playlist": result,
	}
	s.writeAPISuccess(w, resp, "获取歌单详情成功", http.StatusOK)
}

func (s *Server) handleAlbum(w http.ResponseWriter, r *http.Request) {
	if !isMethodAllowed(r.Method) {
		s.writeAPIError(w, "请求参数错误", http.StatusBadRequest, "")
		return
	}

	if s.siteSettingsSvc != nil {
		if blocked, msg := s.checkFeatureGate(r, "album"); blocked {
			s.writeAPIError(w, msg, http.StatusForbidden, "FEATURE_DISABLED")
			return
		}
	}

	data := parseRequestData(r)
	rawID := strings.TrimSpace(data["id"])
	if rawID == "" {
		s.writeAPIError(w, "参数 'album_id' 不能为空", http.StatusBadRequest, "")
		return
	}

	albumID, err := strconv.ParseInt(s.extractMusicID(rawID), 10, 64)
	if err != nil || albumID <= 0 {
		s.writeAPIError(w, "专辑ID格式无效", http.StatusBadRequest, "")
		return
	}

	result, err := s.api.GetAlbumDetail(r.Context(), albumID, s.getCookies())
	if err != nil {
		s.writeAPIError(w, "获取专辑失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}

	resp := map[string]any{
		"status": 200,
		"album":  result,
	}
	s.writeAPISuccess(w, resp, "获取专辑详情成功", http.StatusOK)
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	if !isMethodAllowed(r.Method) {
		s.writeAPIError(w, "请求参数错误", http.StatusBadRequest, "")
		return
	}

	data := parseRequestData(r)
	rawID := strings.TrimSpace(data["id"])
	quality := firstNonEmpty(data["quality"], "lossless")
	returnFormat := firstNonEmpty(data["format"], "file")

	if rawID == "" {
		s.writeAPIError(w, "参数 'music_id' 不能为空", http.StatusBadRequest, "")
		return
	}
	if _, ok := validQualities[quality]; !ok {
		s.writeAPIError(w, "无效的音质参数，支持: standard, exhigh, lossless, hires, sky, jyeffect, jymaster", http.StatusBadRequest, "")
		return
	}
	if returnFormat != "file" && returnFormat != "json" {
		s.writeAPIError(w, "返回格式只支持 'file' 或 'json'", http.StatusBadRequest, "")
		return
	}

	musicID, err := strconv.ParseInt(s.extractMusicID(rawID), 10, 64)
	if err != nil || musicID <= 0 {
		s.writeAPIError(w, "音乐ID格式无效", http.StatusBadRequest, "")
		return
	}

	ctx := r.Context()
	cookies := s.getCookies()

	songInfo, err := s.api.GetSongDetail(ctx, musicID)
	if err != nil {
		s.writeAPIError(w, "未找到音乐信息", http.StatusNotFound, "")
		return
	}
	songs := asSlice(songInfo["songs"])
	if len(songs) == 0 {
		s.writeAPIError(w, "未找到音乐信息", http.StatusNotFound, "")
		return
	}
	songData := asMap(songs[0])

	urlData, usedQuality, err := s.getPlayableSongURL(ctx, musicID, quality, cookies)
	if err != nil {
		s.writeAPIError(w, "无法获取音乐下载链接，可能是版权限制或音质不支持", http.StatusNotFound, "")
		return
	}
	downloadURL := asString(urlData["url"])

	songName := asString(songData["name"])
	artists := strings.Join(extractArtistNames(songData["ar"]), ", ")
	album := asString(asMap(songData["al"])["name"])
	picURL := asString(asMap(songData["al"])["picUrl"])
	fileType := detectFileType(asString(urlData["type"]), downloadURL)
	fileSize := asInt64(urlData["size"])
	duration := asInt64(songData["dt"])

	downloadName := sanitizeFilename(fmt.Sprintf("%s - %s", artists, songName)) + "." + fileType

	if returnFormat == "json" {
		resp := map[string]any{
			"music_id":            strconv.FormatInt(musicID, 10),
			"name":                songName,
			"artist":              artists,
			"album":               album,
			"quality":             usedQuality,
			"requested_quality":   quality,
			"fallback":            usedQuality != quality,
			"quality_name":        qualityDisplayName(usedQuality),
			"file_type":           fileType,
			"file_size":           fileSize,
			"file_size_formatted": formatFileSize(fileSize),
			"filename":            downloadName,
			"duration":            duration,
			"pic_url":             picURL,
		}
		s.writeAPISuccess(w, resp, "下载完成", http.StatusOK)
		return
	}

	tmpFile, err := s.downloadToTempWithRetry(ctx, downloadURL, fileType)
	if err != nil {
		s.writeAPIError(w, "下载失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}
	defer os.Remove(tmpFile)

	meta := SongMeta{
		Title:    songName,
		Artist:   artists,
		Album:    album,
		CoverURL: picURL,
	}
	if err := embedMetadata(ctx, tmpFile, fileType, meta, s.api.HTTPClient()); err != nil {
		s.logger.Printf("embed metadata warning: %v", err)
	}

	fi, err := os.Stat(tmpFile)
	if err != nil {
		s.writeAPIError(w, "文件处理失败", http.StatusInternalServerError, "")
		return
	}

	contentType := mime.TypeByExtension("." + fileType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+url.QueryEscape(downloadName))
	w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
	w.Header().Set("X-Download-Message", "Download completed successfully")
	w.Header().Set("X-Download-Filename", url.QueryEscape(downloadName))

	f, err := os.Open(tmpFile)
	if err != nil {
		s.writeAPIError(w, "文件读取失败", http.StatusInternalServerError, "")
		return
	}
	defer f.Close()
	io.Copy(w, f)
}

func (s *Server) handleAPIInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeAPIError(w, "请求参数错误", http.StatusBadRequest, "")
		return
	}

	downloadsAbs, _ := filepath.Abs(s.cfg.DownloadsDir)
	info := map[string]any{
		"name":        "MeloVault 音乐服务",
		"version":     "3.0.0",
		"description": "提供网易云音乐相关API服务，支持用户认证",
		"endpoints": map[string]any{
			"public": map[string]string{
				"/health":             "GET - 健康检查",
				"/song":               "GET/POST - 获取歌曲信息",
				"/api/stream":         "GET - 同源音频流代理",
				"/playlist":           "GET/POST - 获取歌单详情",
				"/album":              "GET/POST - 获取专辑详情",
				"/api/info":           "GET - API信息",
				"/api/auth/register":  "POST - 邮箱注册",
				"/api/auth/login":     "POST - 邮箱登录",
				"/api/auth/linuxdo":   "GET - Linuxdo OAuth登录",
				"/api/auth/refresh":   "POST - 刷新令牌",
				"/api/auth/send-code":     "POST - 发送验证码",
				"/api/public/search":      "GET/POST - 搜索音乐（公开，供博客等嵌入）",
			},
			"protected": map[string]string{
				"/api/search":              "GET/POST - 搜索音乐（需登录）",
				"/download":                "GET/POST - 下载音乐",
				"/api/auth/me":             "GET - 获取当前用户",
				"/api/user/profile":        "PUT - 更新个人信息",
				"/api/user/password":       "PUT - 修改密码",
				"/api/favorites":           "GET/POST - 收藏列表/添加收藏",
				"/api/favorites/:id":       "DELETE - 取消收藏",
				"/api/favorites/check/:id": "GET - 检查是否收藏",
				"/api/downloads":           "GET/POST/DELETE - 下载历史",
			},
		},
		"supported_qualities": []string{
			"standard", "exhigh", "lossless", "hires", "sky", "jyeffect", "jymaster",
		},
		"config": map[string]any{
			"downloads_dir":   downloadsAbs,
			"request_timeout": "30s",
		},
		"uptime_sec": int64(time.Since(s.startedAt).Seconds()),
	}
	s.writeAPISuccess(w, info, "API信息获取成功", http.StatusOK)
}

func (s *Server) getCookies() map[string]string {
	if s.accountPool != nil {
		cookies, err := s.accountPool.Next(context.Background())
		if err != nil {
			s.logger.Printf("account pool next failed: %v", err)
		}
		if len(cookies) > 0 {
			return cookies
		}
	}
	if cookies := s.getRuntimeCookies(); len(cookies) > 0 {
		return cookies
	}
	cookies, err := s.cookieManager.ParseFromFile()
	if err != nil {
		s.logger.Printf("read cookie failed: %v", err)
		return map[string]string{}
	}
	if len(cookies) > 0 {
		s.setRuntimeCookies(cookies)
	}
	return cookies
}

func (s *Server) getRuntimeCookies() map[string]string {
	s.runtimeCookieMu.RLock()
	defer s.runtimeCookieMu.RUnlock()
	return cloneCookieMap(s.runtimeCookies)
}

func (s *Server) setRuntimeCookies(cookies map[string]string) {
	copied := cloneCookieMap(cookies)
	if len(copied) == 0 {
		return
	}
	s.runtimeCookieMu.Lock()
	s.runtimeCookies = copied
	s.runtimeCookieMu.Unlock()
}

func cloneCookieMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" || v == "" {
			continue
		}
		out[k] = v
	}
	return out
}

func (s *Server) extractMusicID(idOrURL string) string {
	value := strings.TrimSpace(idOrURL)
	if value == "" {
		return ""
	}

	if strings.Contains(value, "163cn.tv") {
		if resolved, err := resolveShortURL(value); err == nil && strings.TrimSpace(resolved) != "" {
			value = resolved
		}
	}

	if strings.Contains(value, "music.163.com") {
		if parsed, err := url.Parse(value); err == nil {
			if id := parsed.Query().Get("id"); strings.TrimSpace(id) != "" {
				return strings.TrimSpace(id)
			}

			if parsed.Fragment != "" {
				frag := parsed.Fragment
				if idx := strings.Index(frag, "id="); idx >= 0 {
					sub := frag[idx+3:]
					if i := strings.Index(sub, "&"); i >= 0 {
						sub = sub[:i]
					}
					if strings.TrimSpace(sub) != "" {
						return strings.TrimSpace(sub)
					}
				}
			}
		}
	}

	if matched := idRegex.FindString(value); matched != "" {
		return matched
	}
	return value
}

func (s *Server) getFileLock(path string) *sync.Mutex {
	lockAny, _ := s.fileLocks.LoadOrStore(path, &sync.Mutex{})
	return lockAny.(*sync.Mutex)
}

func (s *Server) downloadToTempWithRetry(ctx context.Context, sourceURL, fileType string) (string, error) {
	const maxRetries = 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		tmpFile, err := s.downloadToTemp(ctx, sourceURL, fileType)
		if err == nil {
			return tmpFile, nil
		}
		lastErr = err
		s.logger.Printf("download attempt %d/%d failed: %v", attempt, maxRetries, err)

		if attempt < maxRetries {
			backoff := time.Duration(attempt) * 2 * time.Second
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(backoff):
			}
		}
	}
	return "", fmt.Errorf("all %d download attempts failed, last error: %w", maxRetries, lastErr)
}

func (s *Server) downloadToTemp(ctx context.Context, sourceURL, fileType string) (string, error) {
	tmpFile, err := os.CreateTemp("", "melovault-*."+fileType)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	req, err := http.NewRequestWithContext(context.WithoutCancel(ctx), http.MethodGet, sourceURL, nil)
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := s.api.HTTPClient().Do(req)
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		tmpFile.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("download request failed: status=%d", resp.StatusCode)
	}

	buffer := make([]byte, 32*1024)
	_, copyErr := io.CopyBuffer(tmpFile, resp.Body, buffer)
	closeErr := tmpFile.Close()
	if copyErr != nil {
		os.Remove(tmpPath)
		return "", copyErr
	}
	if closeErr != nil {
		os.Remove(tmpPath)
		return "", closeErr
	}

	return tmpPath, nil
}

// checkFeatureGate returns (blocked, message). If blocked is true the request should be denied.
func (s *Server) checkFeatureGate(r *http.Request, feature string) (bool, string) {
	f, err := s.siteSettingsSvc.Get(r.Context())
	if err != nil {
		return false, ""
	}

	var enabled, adminOnly bool
	switch feature {
	case "playlist":
		enabled, adminOnly = f.PlaylistParseEnabled, f.PlaylistParseAdminOnly
	case "album":
		enabled, adminOnly = f.AlbumParseEnabled, f.AlbumParseAdminOnly
	default:
		return false, ""
	}

	if !enabled {
		return true, "该功能已被管理员关闭"
	}
	if adminOnly {
		userID, ok := GetUserID(r.Context())
		if !ok || userID == 0 {
			return true, "该功能仅管理员可用，请先登录"
		}
		user, err := s.auth.GetUserByID(r.Context(), userID)
		if err != nil || (user.Role != "admin" && user.Role != "superadmin") {
			return true, "该功能仅管理员可用"
		}
	}
	return false, ""
}

func (s *Server) writeAPISuccess(w http.ResponseWriter, data any, message string, statusCode int) {
	resp := map[string]any{
		"status":  statusCode,
		"success": true,
		"message": message,
	}
	if data != nil {
		resp["data"] = data
	}
	s.writeJSON(w, statusCode, resp)
}

func (s *Server) writeAPIError(w http.ResponseWriter, message string, statusCode int, errorCode string) {
	resp := map[string]any{
		"status":  statusCode,
		"success": false,
		"message": message,
	}
	if errorCode != "" {
		resp["error_code"] = errorCode
	}
	s.writeJSON(w, statusCode, resp)
}

func (s *Server) writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(data)
}

func resolveShortURL(raw string) (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequest(http.MethodGet, raw, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	loc := strings.TrimSpace(resp.Header.Get("Location"))
	if loc != "" {
		return loc, nil
	}
	return raw, nil
}

func parseRequestData(r *http.Request) map[string]string {
	out := make(map[string]string)
	if r.Method == http.MethodGet {
		for key, values := range r.URL.Query() {
			if len(values) > 0 {
				out[key] = values[0]
			}
		}
		return out
	}

	_ = r.ParseForm()
	for key, values := range r.PostForm {
		if len(values) > 0 {
			out[key] = values[0]
		}
	}

	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	if strings.Contains(contentType, "application/json") {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err == nil && len(body) > 0 {
			var jsonData map[string]any
			if json.Unmarshal(body, &jsonData) == nil {
				for k, v := range jsonData {
					out[k] = fmt.Sprintf("%v", v)
				}
			}
		}
	}

	return out
}

func isMethodAllowed(method string) bool {
	return method == http.MethodGet || method == http.MethodPost
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func qualityFallbackOrder(preferred string) []string {
	base := []string{"jymaster", "jyeffect", "sky", "hires", "lossless", "exhigh", "standard"}
	preferred = strings.TrimSpace(preferred)
	if preferred == "" {
		return base
	}

	seen := make(map[string]struct{}, len(base)+1)
	out := make([]string, 0, len(base)+1)
	out = append(out, preferred)
	seen[preferred] = struct{}{}
	for _, q := range base {
		if _, ok := seen[q]; ok {
			continue
		}
		seen[q] = struct{}{}
		out = append(out, q)
	}
	return out
}

func (s *Server) getPlayableSongURL(ctx context.Context, musicID int64, preferredQuality string, cookies map[string]string) (map[string]any, string, error) {
	var lastErr error
	for _, q := range qualityFallbackOrder(preferredQuality) {
		result, err := s.api.GetSongURL(ctx, musicID, q, cookies)
		if err != nil {
			lastErr = err
			continue
		}
		dataRaw := asSlice(result["data"])
		if len(dataRaw) == 0 {
			continue
		}
		songData := asMap(dataRaw[0])
		if strings.TrimSpace(asString(songData["url"])) == "" {
			continue
		}
		level := firstNonEmpty(asString(songData["level"]), q)
		return songData, level, nil
	}
	if lastErr != nil {
		return nil, "", lastErr
	}
	return nil, "", fmt.Errorf("no playable url found")
}

func formatFileSize(sizeBytes int64) string {
	if sizeBytes <= 0 {
		return "0B"
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	size := float64(sizeBytes)
	unit := 0
	for size >= 1024 && unit < len(units)-1 {
		size /= 1024
		unit++
	}
	return fmt.Sprintf("%.2f%s", size, units[unit])
}

func qualityDisplayName(quality string) string {
	names := map[string]string{
		"standard": "标准音质",
		"exhigh":   "极高音质",
		"lossless": "无损音质",
		"hires":    "Hi-Res音质",
		"sky":      "沉浸环绕声",
		"jyeffect": "高清环绕声",
		"jymaster": "超清母带",
		"dolby":    "杜比全景声",
	}
	if name, ok := names[quality]; ok {
		return name
	}
	return "未知音质(" + quality + ")"
}

func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "unknown"
	}

	illegal := map[rune]struct{}{
		'<': {}, '>': {}, ':': {}, '"': {}, '/': {}, '\\': {}, '|': {}, '?': {}, '*': {},
	}
	builder := strings.Builder{}
	builder.Grow(len(name))
	for _, r := range name {
		if _, bad := illegal[r]; bad {
			continue
		}
		if r < 32 {
			continue
		}
		builder.WriteRune(r)
	}
	out := strings.TrimSpace(builder.String())
	if out == "" {
		return "unknown"
	}
	if len(out) > 180 {
		return out[:180]
	}
	return out
}

func detectFileType(fromAPI, sourceURL string) string {
	ft := strings.ToLower(strings.TrimSpace(fromAPI))
	if ft != "" {
		return ft
	}
	urlLower := strings.ToLower(sourceURL)
	switch {
	case strings.Contains(urlLower, ".flac"):
		return "flac"
	case strings.Contains(urlLower, ".m4a"):
		return "m4a"
	default:
		return "mp3"
	}
}

func normalizePlayableMediaURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "//") {
		return "https:" + raw
	}
	if strings.HasPrefix(lower, "http://") && strings.Contains(lower, ".music.126.net/") {
		return "https://" + raw[len("http://"):]
	}
	return raw
}

func isSecureBrowserMediaURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	if strings.HasPrefix(raw, "/") {
		return true
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return strings.EqualFold(parsed.Scheme, "https")
}

func buildStreamProxyURL(musicID int64, quality string) string {
	params := url.Values{}
	params.Set("id", strconv.FormatInt(musicID, 10))
	params.Set("level", quality)
	return "/api/stream?" + params.Encode()
}

func copyRequestHeader(dst, src http.Header, key string) {
	if value := strings.TrimSpace(src.Get(key)); value != "" {
		dst.Set(key, value)
	}
}

func copyResponseHeader(dst, src http.Header, key string) {
	if value := strings.TrimSpace(src.Get(key)); value != "" {
		dst.Set(key, value)
	}
}

func extractArtistNames(raw any) []string {
	items := asSlice(raw)
	artists := make([]string, 0, len(items))
	for _, item := range items {
		name := asString(asMap(item)["name"])
		if name != "" {
			artists = append(artists, name)
		}
	}
	return artists
}

func asMap(v any) map[string]any {
	m, ok := v.(map[string]any)
	if !ok || m == nil {
		return map[string]any{}
	}
	return m
}

func asSlice(v any) []any {
	s, ok := v.([]any)
	if !ok || s == nil {
		return []any{}
	}
	return s
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatInt(int64(t), 10)
	case int64:
		return strconv.FormatInt(t, 10)
	case int:
		return strconv.Itoa(t)
	default:
		return ""
	}
}

func asInt64(v any) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case float64:
		return int64(t)
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		if err == nil {
			return n
		}
		return 0
	default:
		return 0
	}
}

type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (s *statusResponseWriter) WriteHeader(statusCode int) {
	s.statusCode = statusCode
	s.ResponseWriter.WriteHeader(statusCode)
}
