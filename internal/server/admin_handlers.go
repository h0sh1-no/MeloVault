package server

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/h0sh1-no/MeloVault/internal/activity"
	"github.com/h0sh1-no/MeloVault/internal/cookie"
	"github.com/h0sh1-no/MeloVault/internal/netease"
)

// ── Setup (one-time, public) ──────────────────────────────────────────────────

// handleSetupStatus returns whether the system has been initialized.
func (s *Server) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}
	if s.adminSvc == nil {
		s.writeAPISuccess(w, map[string]any{"initialized": false}, "ok", http.StatusOK)
		return
	}
	exists, err := s.adminSvc.IsAnyUserExists(r.Context())
	if err != nil {
		s.writeAPIError(w, "查询失败", http.StatusInternalServerError, "")
		return
	}
	s.writeAPISuccess(w, map[string]any{"initialized": exists}, "ok", http.StatusOK)
}

// handleSetupInit registers the first superadmin. Only succeeds when no account exists.
func (s *Server) handleSetupInit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}
	if s.adminSvc == nil || s.auth == nil {
		s.writeAPIError(w, "数据库未配置", http.StatusServiceUnavailable, "")
		return
	}

	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := parseJSONBody(r, &req); err != nil {
		s.writeAPIError(w, "无效的请求数据", http.StatusBadRequest, "")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)
	req.Password = strings.TrimSpace(req.Password)

	if req.Username == "" || req.Email == "" || req.Password == "" {
		s.writeAPIError(w, "用户名、邮箱和密码不能为空", http.StatusBadRequest, "")
		return
	}
	if len(req.Username) < 2 || len(req.Username) > 50 {
		s.writeAPIError(w, "用户名长度须在2-50个字符之间", http.StatusBadRequest, "")
		return
	}
	if len(req.Password) < 6 {
		s.writeAPIError(w, "密码至少6个字符", http.StatusBadRequest, "")
		return
	}

	userID, err := s.adminSvc.InitSuperAdmin(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		if strings.Contains(err.Error(), "already initialized") {
			s.writeAPIError(w, "系统已初始化，无法重复创建超级管理员", http.StatusConflict, "ALREADY_INITIALIZED")
			return
		}
		s.writeAPIError(w, "初始化失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}

	user, err := s.auth.GetUserByID(r.Context(), userID)
	if err != nil {
		s.writeAPIError(w, "获取用户信息失败", http.StatusInternalServerError, "")
		return
	}
	tokens, err := s.auth.GenerateTokenPair(user)
	if err != nil {
		s.writeAPIError(w, "生成令牌失败", http.StatusInternalServerError, "")
		return
	}

	s.writeAPISuccess(w, map[string]any{
		"user":   sanitizeUser(user),
		"tokens": tokens,
	}, "超级管理员创建成功", http.StatusCreated)
}

// ── Admin middleware helper ───────────────────────────────────────────────────

// AdminMiddleware requires a valid JWT and role admin/superadmin.
func (s *Server) AdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.auth == nil {
			s.writeAPIError(w, "数据库未配置", http.StatusServiceUnavailable, "")
			return
		}
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			s.writeAPIError(w, "未提供认证令牌", http.StatusUnauthorized, "UNAUTHORIZED")
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			s.writeAPIError(w, "无效的认证格式", http.StatusUnauthorized, "")
			return
		}
		claims, err := s.auth.ValidateToken(strings.TrimSpace(parts[1]))
		if err != nil {
			s.writeAPIError(w, "无效或过期的令牌", http.StatusUnauthorized, "INVALID_TOKEN")
			return
		}
		user, err := s.auth.GetUserByID(r.Context(), claims.UserID)
		if err != nil || (user.Role != "admin" && user.Role != "superadmin") {
			s.writeAPIError(w, "权限不足，需要管理员身份", http.StatusForbidden, "FORBIDDEN")
			return
		}
		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, UsernameKey, claims.Username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// adminMiddlewareFunc wraps AdminMiddleware for http.HandleFunc.
func (s *Server) adminMiddlewareFunc(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.AdminMiddleware(http.HandlerFunc(handler)).ServeHTTP(w, r)
	}
}

// ── Admin: Stats ─────────────────────────────────────────────────────────────

func (s *Server) handleAdminStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}
	stats, err := s.adminSvc.GetStats(r.Context())
	if err != nil {
		s.writeAPIError(w, "获取统计失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}
	s.writeAPISuccess(w, stats, "ok", http.StatusOK)
}

// ── Admin: Users ─────────────────────────────────────────────────────────────

func (s *Server) handleAdminUsersRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleAdminListUsers(w, r)
	case http.MethodPost:
		s.handleAdminCreateUser(w, r)
	default:
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
	}
}

func (s *Server) handleAdminCreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := parseJSONBody(r, &req); err != nil {
		s.writeAPIError(w, "无效的请求数据", http.StatusBadRequest, "")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)
	req.Password = strings.TrimSpace(req.Password)
	req.Role = strings.TrimSpace(req.Role)

	if req.Username == "" || req.Password == "" {
		s.writeAPIError(w, "用户名和密码不能为空", http.StatusBadRequest, "")
		return
	}
	if len(req.Username) < 2 || len(req.Username) > 50 {
		s.writeAPIError(w, "用户名长度须在2-50个字符之间", http.StatusBadRequest, "")
		return
	}
	if len(req.Password) < 6 {
		s.writeAPIError(w, "密码至少6个字符", http.StatusBadRequest, "")
		return
	}
	allowedRoles := map[string]bool{"user": true, "admin": true, "superadmin": true}
	if req.Role == "" {
		req.Role = "user"
	}
	if !allowedRoles[req.Role] {
		s.writeAPIError(w, "无效的角色值", http.StatusBadRequest, "")
		return
	}

	userID, err := s.adminSvc.CreateUser(r.Context(), req.Username, req.Email, req.Password, req.Role)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "already exists") {
			s.writeAPIError(w, "创建失败: "+msg, http.StatusConflict, "")
			return
		}
		s.writeAPIError(w, "创建失败: "+msg, http.StatusInternalServerError, "")
		return
	}
	s.writeAPISuccess(w, map[string]any{"id": userID}, "用户创建成功", http.StatusCreated)
}

func (s *Server) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	page := parseIntParam(r, "page", 1)
	pageSize := parseIntParam(r, "page_size", 20)
	search := strings.TrimSpace(r.URL.Query().Get("search"))

	users, total, err := s.adminSvc.ListUsers(r.Context(), page, pageSize, search)
	if err != nil {
		s.writeAPIError(w, "获取用户列表失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}
	s.writeAPISuccess(w, map[string]any{
		"list":      users,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}, "ok", http.StatusOK)
}

func (s *Server) handleAdminUserDetail(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/admin/users/")

	if strings.Contains(trimmed, "/activity") {
		s.handleAdminUserActivity(w, r)
		return
	}
	if strings.Contains(trimmed, "/password") {
		s.handleAdminResetPassword(w, r)
		return
	}
	if strings.Contains(trimmed, "/downloads") {
		s.handleAdminUserDownloads(w, r)
		return
	}

	idStr := trimmed
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id == 0 {
		s.writeAPIError(w, "无效的用户ID", http.StatusBadRequest, "")
		return
	}

	switch r.Method {
	case http.MethodGet:
		user, err := s.adminSvc.GetUser(r.Context(), id)
		if err != nil {
			s.writeAPIError(w, "用户不存在", http.StatusNotFound, "")
			return
		}
		s.writeAPISuccess(w, user, "ok", http.StatusOK)

	case http.MethodPut:
		var req struct {
			Username string `json:"username"`
			Role     string `json:"role"`
		}
		if err := parseJSONBody(r, &req); err != nil {
			s.writeAPIError(w, "无效的请求数据", http.StatusBadRequest, "")
			return
		}
		req.Username = strings.TrimSpace(req.Username)
		req.Role = strings.TrimSpace(req.Role)
		if req.Username == "" {
			s.writeAPIError(w, "用户名不能为空", http.StatusBadRequest, "")
			return
		}
		allowedRoles := map[string]bool{"user": true, "admin": true, "superadmin": true}
		if req.Role != "" && !allowedRoles[req.Role] {
			s.writeAPIError(w, "无效的角色值", http.StatusBadRequest, "")
			return
		}
		if req.Role == "" {
			s.writeAPIError(w, "角色不能为空", http.StatusBadRequest, "")
			return
		}
		if err := s.adminSvc.UpdateUser(r.Context(), id, req.Username, req.Role); err != nil {
			s.writeAPIError(w, "更新失败: "+err.Error(), http.StatusInternalServerError, "")
			return
		}
		s.writeAPISuccess(w, nil, "更新成功", http.StatusOK)

	case http.MethodDelete:
		if err := s.adminSvc.DeleteUser(r.Context(), id); err != nil {
			if strings.Contains(err.Error(), "user not found") {
				s.writeAPIError(w, "删除失败: "+err.Error(), http.StatusNotFound, "")
				return
			}
			s.writeAPIError(w, "删除失败: "+err.Error(), http.StatusBadRequest, "")
			return
		}
		s.writeAPISuccess(w, nil, "删除成功", http.StatusOK)

	default:
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
	}
}

func (s *Server) handleAdminResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/admin/users/")
	idStr := strings.TrimSuffix(trimmed, "/password")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id == 0 {
		s.writeAPIError(w, "无效的用户ID", http.StatusBadRequest, "")
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	if err := parseJSONBody(r, &req); err != nil {
		s.writeAPIError(w, "无效的请求数据", http.StatusBadRequest, "")
		return
	}
	req.Password = strings.TrimSpace(req.Password)
	if len(req.Password) < 6 {
		s.writeAPIError(w, "密码至少6个字符", http.StatusBadRequest, "")
		return
	}
	if err := s.adminSvc.ResetPassword(r.Context(), id, req.Password); err != nil {
		s.writeAPIError(w, "重置密码失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}
	s.writeAPISuccess(w, nil, "密码重置成功", http.StatusOK)
}

func (s *Server) handleAdminUserDownloads(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/admin/users/")
	idStr := strings.TrimSuffix(trimmed, "/downloads")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id == 0 {
		s.writeAPIError(w, "无效的用户ID", http.StatusBadRequest, "")
		return
	}
	page := parseIntParam(r, "page", 1)
	pageSize := parseIntParam(r, "page_size", 20)

	records, total, err := s.adminSvc.GetUserDownloads(r.Context(), id, page, pageSize)
	if err != nil {
		s.writeAPIError(w, "获取下载记录失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}
	s.writeAPISuccess(w, map[string]any{
		"list":      records,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}, "ok", http.StatusOK)
}

// ── Admin: Downloads ─────────────────────────────────────────────────────────

func (s *Server) handleAdminDownloads(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}
	page := parseIntParam(r, "page", 1)
	pageSize := parseIntParam(r, "page_size", 20)
	search := strings.TrimSpace(r.URL.Query().Get("search"))

	records, total, err := s.adminSvc.GetDownloads(r.Context(), page, pageSize, search)
	if err != nil {
		s.writeAPIError(w, "获取下载记录失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}
	s.writeAPISuccess(w, map[string]any{
		"list":      records,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}, "ok", http.StatusOK)
}

// ── Admin: Netease Cookie / QR Login ─────────────────────────────────────────

// handleAdminNeteaseQRKey generates a new QR login key and returns it + the
// URL to encode into a QR code.
func (s *Server) handleAdminNeteaseQRKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}
	key, loginURL, err := s.api.GetLoginQRKey(r.Context())
	if err != nil {
		s.writeAPIError(w, "生成二维码失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}
	s.writeAPISuccess(w, map[string]any{
		"key":       key,
		"login_url": loginURL,
	}, "ok", http.StatusOK)
}

// handleAdminNeteaseQRCheck polls QR scan status.
// On success (code 803) it saves the cookies automatically.
func (s *Server) handleAdminNeteaseQRCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}
	key := strings.TrimSpace(r.URL.Query().Get("key"))
	if key == "" {
		s.writeAPIError(w, "缺少 key 参数", http.StatusBadRequest, "")
		return
	}

	code, cookieStr, err := s.api.CheckLoginQRStatus(r.Context(), key)
	if err != nil {
		s.writeAPIError(w, "查询二维码状态失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}

	messages := map[int]string{
		netease.QRExpired:    "二维码已过期，请重新生成",
		netease.QRWaiting:    "等待扫码",
		netease.QRScanned:    "已扫码，等待确认",
		netease.QRAuthorized: "登录成功",
	}
	msg := messages[code]
	if msg == "" {
		msg = "未知状态"
	}

	savedToPool := false
	savedToCookieFile := false
	savedToMemory := false
	accountID := int64(0)
	poolError := ""
	cookieFileError := ""
	warning := ""

	if code == netease.QRAuthorized {
		cookieStr = strings.TrimSpace(cookieStr)
		if cookieStr == "" {
			warning = "登录已授权，但未获取到 Cookie，请重新生成二维码并重试"
			s.logger.Printf("qr login authorized but cookie missing, key=%s", key)
		} else {
			parsed := cookie.ParseCookieString(cookieStr)
			if len(parsed) == 0 {
				warning = "登录已授权，但 Cookie 解析失败，请重试扫码"
				s.logger.Printf("qr login authorized but cookie parse failed, key=%s", key)
			} else if s.accountPool != nil {
				nickname := strings.TrimSpace(r.URL.Query().Get("nickname"))
				if nickname == "" {
					nickname = "扫码登录 " + time.Now().Format("01-02 15:04")
				}

				persistCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
				id, err := s.accountPool.Add(persistCtx, nickname, cookieStr)
				cancel()
				if err != nil {
					poolError = err.Error()
					s.logger.Printf("save account to pool failed: %v", err)
					if err2 := s.cookieManager.Write(parsed); err2 != nil {
						cookieFileError = err2.Error()
						s.logger.Printf("fallback save cookie file failed: %v", err2)
						s.setRuntimeCookies(parsed)
						savedToMemory = true
						warning = "登录成功，但入池失败且写入 Cookie 文件失败，已回退为内存 Cookie（重启后失效）"
					} else {
						savedToCookieFile = true
						warning = "登录成功，但写入号池失败，已回退写入 Cookie 文件"
					}
				} else {
					accountID = id
					savedToPool = true
					s.setRuntimeCookies(parsed)
					s.logger.Printf("netease account added to pool via QR login, id=%d", id)
					// Keep cookie.txt synchronized as fallback when pool becomes empty.
					if err := s.cookieManager.Write(parsed); err == nil {
						savedToCookieFile = true
					} else {
						s.logger.Printf("sync cookie file failed after pool add: %v", err)
					}
				}
			} else {
				if err := s.cookieManager.Write(parsed); err != nil {
					cookieFileError = err.Error()
					s.logger.Printf("save cookie failed: %v", err)
					s.setRuntimeCookies(parsed)
					savedToMemory = true
					warning = "登录成功，但写入 Cookie 文件失败，已回退为内存 Cookie（重启后失效）"
				} else {
					savedToCookieFile = true
					s.setRuntimeCookies(parsed)
					s.logger.Println("netease cookie updated via QR login")
				}
			}
		}
	}

	resp := map[string]any{
		"code":                 code,
		"message":              msg,
		"cookie_received":      cookieStr != "",
		"saved_to_pool":        savedToPool,
		"saved_to_cookie_file": savedToCookieFile,
		"saved_to_memory":      savedToMemory,
	}
	if poolError != "" {
		resp["pool_error"] = poolError
	}
	if cookieFileError != "" {
		resp["cookie_file_error"] = cookieFileError
	}
	if accountID > 0 {
		resp["account_id"] = accountID
	}
	if warning != "" {
		resp["warning"] = warning
	}

	s.writeAPISuccess(w, resp, msg, http.StatusOK)
}

// handleAdminNeteaseCookie manually sets the Netease cookie string.
func (s *Server) handleAdminNeteaseCookie(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}
	var req struct {
		Cookie string `json:"cookie"`
	}
	if err := parseJSONBody(r, &req); err != nil {
		s.writeAPIError(w, "无效的请求数据", http.StatusBadRequest, "")
		return
	}
	req.Cookie = strings.TrimSpace(req.Cookie)
	if req.Cookie == "" {
		s.writeAPIError(w, "Cookie 不能为空", http.StatusBadRequest, "")
		return
	}

	mgr := cookie.NewManager(s.cfg.CookieFile)
	parsed := cookie.ParseCookieString(req.Cookie)
	if len(parsed) == 0 {
		s.writeAPIError(w, "无法解析 Cookie，格式应为 key=value; key2=value2", http.StatusBadRequest, "")
		return
	}
	if err := mgr.Write(parsed); err != nil {
		s.logger.Printf("save cookie file failed, fallback runtime cookies: %v", err)
		s.setRuntimeCookies(parsed)
		s.writeAPISuccess(w, map[string]any{
			"keys":            len(parsed),
			"saved_to_memory": true,
		}, "Cookie 文件写入失败，已临时写入内存（重启后失效）", http.StatusOK)
		return
	}
	s.setRuntimeCookies(parsed)
	s.writeAPISuccess(w, map[string]any{"keys": len(parsed)}, "Cookie 保存成功", http.StatusOK)
}

// ── Admin: Analytics ─────────────────────────────────────────────────────────

func (s *Server) handleAnalyticsOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}
	if s.activitySvc == nil {
		s.writeAPIError(w, "服务未配置", http.StatusServiceUnavailable, "")
		return
	}
	stats, err := s.activitySvc.GetOverviewStats(r.Context())
	if err != nil {
		s.writeAPIError(w, "获取统计失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}
	s.writeAPISuccess(w, stats, "ok", http.StatusOK)
}

func (s *Server) handleAnalyticsActivity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}
	if s.activitySvc == nil {
		s.writeAPIError(w, "服务未配置", http.StatusServiceUnavailable, "")
		return
	}
	page := parseIntParam(r, "page", 1)
	pageSize := parseIntParam(r, "page_size", 20)
	filters := activity.Filters{
		Action: strings.TrimSpace(r.URL.Query().Get("action")),
		IP:     strings.TrimSpace(r.URL.Query().Get("ip")),
		Search: strings.TrimSpace(r.URL.Query().Get("search")),
	}
	if uidStr := r.URL.Query().Get("user_id"); uidStr != "" {
		if uid, err := strconv.ParseInt(uidStr, 10, 64); err == nil {
			filters.UserID = uid
		}
	}

	logs, total, err := s.activitySvc.GetActivityLogs(r.Context(), page, pageSize, filters)
	if err != nil {
		s.writeAPIError(w, "获取活动日志失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}
	s.writeAPISuccess(w, map[string]any{
		"list":      logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}, "ok", http.StatusOK)
}

func (s *Server) handleAnalyticsOnline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}
	if s.activitySvc == nil {
		s.writeAPIError(w, "服务未配置", http.StatusServiceUnavailable, "")
		return
	}
	minutes := parseIntParam(r, "minutes", 15)
	users, err := s.activitySvc.GetRecentActiveUsers(r.Context(), minutes)
	if err != nil {
		s.writeAPIError(w, "获取在线用户失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}
	s.writeAPISuccess(w, users, "ok", http.StatusOK)
}

func (s *Server) handleAnalyticsProvinces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}
	if s.activitySvc == nil {
		s.writeAPIError(w, "服务未配置", http.StatusServiceUnavailable, "")
		return
	}
	days := parseIntParam(r, "days", 30)
	stats, err := s.activitySvc.GetProvinceStats(r.Context(), days)
	if err != nil {
		s.writeAPIError(w, "获取省份统计失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}
	s.writeAPISuccess(w, stats, "ok", http.StatusOK)
}

func (s *Server) handleAnalyticsTrends(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}
	if s.activitySvc == nil {
		s.writeAPIError(w, "服务未配置", http.StatusServiceUnavailable, "")
		return
	}
	days := parseIntParam(r, "days", 7)
	trends, err := s.activitySvc.GetTrends(r.Context(), days)
	if err != nil {
		s.writeAPIError(w, "获取趋势数据失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}
	s.writeAPISuccess(w, trends, "ok", http.StatusOK)
}

func (s *Server) handleAdminUserActivity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}
	if s.activitySvc == nil {
		s.writeAPIError(w, "服务未配置", http.StatusServiceUnavailable, "")
		return
	}
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/admin/users/")
	idStr := strings.TrimSuffix(trimmed, "/activity")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id == 0 {
		s.writeAPIError(w, "无效的用户ID", http.StatusBadRequest, "")
		return
	}

	page := parseIntParam(r, "page", 1)
	pageSize := parseIntParam(r, "page_size", 20)
	action := strings.TrimSpace(r.URL.Query().Get("action"))

	logs, total, err := s.activitySvc.GetUserActivityFiltered(r.Context(), id, action, page, pageSize)
	if err != nil {
		s.writeAPIError(w, "获取用户活动失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}
	s.writeAPISuccess(w, map[string]any{
		"list":      logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}, "ok", http.StatusOK)
}

// ── Admin: Netease Account Pool ──────────────────────────────────────────────

func (s *Server) handleAdminNeteaseAccountsRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleAdminListNeteaseAccounts(w, r)
	case http.MethodPost:
		s.handleAdminAddNeteaseAccount(w, r)
	default:
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
	}
}

func (s *Server) handleAdminListNeteaseAccounts(w http.ResponseWriter, r *http.Request) {
	if s.accountPool == nil {
		s.writeAPIError(w, "数据库未配置", http.StatusServiceUnavailable, "")
		return
	}
	accounts, err := s.accountPool.List(r.Context())
	if err != nil {
		s.writeAPIError(w, "获取账号列表失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}
	total, active, _ := s.accountPool.Count(r.Context())
	s.writeAPISuccess(w, map[string]any{
		"list":   accounts,
		"total":  total,
		"active": active,
	}, "ok", http.StatusOK)
}

func (s *Server) handleAdminAddNeteaseAccount(w http.ResponseWriter, r *http.Request) {
	if s.accountPool == nil {
		s.writeAPIError(w, "数据库未配置", http.StatusServiceUnavailable, "")
		return
	}
	var req struct {
		Nickname string `json:"nickname"`
		Cookie   string `json:"cookie"`
	}
	if err := parseJSONBody(r, &req); err != nil {
		s.writeAPIError(w, "无效的请求数据", http.StatusBadRequest, "")
		return
	}
	req.Cookie = strings.TrimSpace(req.Cookie)
	if req.Cookie == "" {
		s.writeAPIError(w, "Cookie 不能为空", http.StatusBadRequest, "")
		return
	}
	if strings.TrimSpace(req.Nickname) == "" {
		req.Nickname = "手动添加 " + time.Now().Format("01-02 15:04")
	}

	id, err := s.accountPool.Add(r.Context(), req.Nickname, req.Cookie)
	if err != nil {
		s.writeAPIError(w, "添加账号失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}
	s.writeAPISuccess(w, map[string]any{"id": id}, "账号添加成功", http.StatusCreated)
}

func (s *Server) handleAdminNeteaseAccountDetail(w http.ResponseWriter, r *http.Request) {
	if s.accountPool == nil {
		s.writeAPIError(w, "数据库未配置", http.StatusServiceUnavailable, "")
		return
	}

	trimmed := strings.TrimPrefix(r.URL.Path, "/api/admin/netease/accounts/")
	id, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil || id == 0 {
		s.writeAPIError(w, "无效的账号ID", http.StatusBadRequest, "")
		return
	}

	switch r.Method {
	case http.MethodPut:
		var req struct {
			Nickname *string `json:"nickname"`
			IsActive *bool   `json:"is_active"`
		}
		if err := parseJSONBody(r, &req); err != nil {
			s.writeAPIError(w, "无效的请求数据", http.StatusBadRequest, "")
			return
		}
		if req.IsActive != nil {
			if err := s.accountPool.ToggleActive(r.Context(), id, *req.IsActive); err != nil {
				s.writeAPIError(w, "更新失败: "+err.Error(), http.StatusInternalServerError, "")
				return
			}
		}
		if req.Nickname != nil {
			if err := s.accountPool.UpdateNickname(r.Context(), id, *req.Nickname); err != nil {
				s.writeAPIError(w, "更新失败: "+err.Error(), http.StatusInternalServerError, "")
				return
			}
		}
		s.writeAPISuccess(w, nil, "更新成功", http.StatusOK)

	case http.MethodDelete:
		if err := s.accountPool.Remove(r.Context(), id); err != nil {
			s.writeAPIError(w, "删除失败: "+err.Error(), http.StatusInternalServerError, "")
			return
		}
		s.writeAPISuccess(w, nil, "删除成功", http.StatusOK)

	default:
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
	}
}
