package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/h0sh1-no/MeloVault/internal/auth"
)

// resolveLinuxdoConfig returns LinuxDO OAuth credentials.
// It reads from site_settings first; if empty, falls back to environment config.
func (s *Server) resolveLinuxdoConfig(r *http.Request) auth.LinuxdoOAuthConfig {
	if s.siteSettingsSvc != nil {
		f, err := s.siteSettingsSvc.Get(r.Context())
		if err == nil && f.LinuxdoConfigured() {
			return auth.LinuxdoOAuthConfig{
				ClientID:     f.LinuxdoClientID,
				ClientSecret: f.LinuxdoClientSecret,
				RedirectURI:  f.LinuxdoRedirectURI,
			}
		}
	}
	return auth.LinuxdoOAuthConfig{
		ClientID:     s.cfg.LinuxdoClientID,
		ClientSecret: s.cfg.LinuxdoClientSecret,
		RedirectURI:  s.cfg.LinuxdoRedirectURI,
	}
}

// resolveSMTPConfig returns SMTP credentials.
// It reads from site_settings first; if empty, falls back to environment config.
func (s *Server) resolveSMTPConfig(r *http.Request) (host string, port int, user, password, from string) {
	if s.siteSettingsSvc != nil {
		f, err := s.siteSettingsSvc.Get(r.Context())
		if err == nil && f.SmtpConfigured() {
			return f.SMTPHost, f.SMTPPort, f.SMTPUser, f.SMTPPassword, f.SMTPFrom
		}
	}
	return s.cfg.SMTPHost, s.cfg.SMTPPort, s.cfg.SMTPUser, s.cfg.SMTPPassword, ""
}

// resolveFrontendURL returns site_url from site_settings if set,
// otherwise falls back to the FRONTEND_URL env config.
func (s *Server) resolveFrontendURL(r *http.Request) string {
	if s.siteSettingsSvc != nil {
		f, err := s.siteSettingsSvc.Get(r.Context())
		if err == nil && f.SiteURL != "" {
			return f.SiteURL
		}
	}
	return s.cfg.FrontendURL
}

// requireSetupInitialized ensures at least one account exists before opening
// public registration flows.
func (s *Server) requireSetupInitialized(w http.ResponseWriter, r *http.Request) bool {
	if s.adminSvc == nil {
		return true
	}
	initialized, err := s.adminSvc.IsAnyUserExists(r.Context())
	if err != nil {
		s.writeAPIError(w, "查询系统初始化状态失败", http.StatusInternalServerError, "")
		return false
	}
	if !initialized {
		s.writeAPIError(w, "系统尚未初始化，请先创建超级管理员账户", http.StatusForbidden, "SETUP_REQUIRED")
		return false
	}
	return true
}

// handleRegister handles user registration with email and password.
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}
	if s.auth == nil {
		s.writeAPIError(w, "数据库未配置", http.StatusServiceUnavailable, "")
		return
	}
	if !s.requireSetupInitialized(w, r) {
		return
	}

	if s.siteSettingsSvc != nil {
		f, err := s.siteSettingsSvc.Get(r.Context())
		if err == nil {
			if !f.AllowRegister {
				s.writeAPIError(w, "管理员已关闭注册功能", http.StatusForbidden, "REGISTER_DISABLED")
				return
			}
			if !f.AllowEmailRegister {
				s.writeAPIError(w, "管理员已关闭邮箱注册", http.StatusForbidden, "EMAIL_REGISTER_DISABLED")
				return
			}
		}
	}

	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Code     string `json:"code"` // verification code (optional)
	}

	if err := parseJSONBody(r, &req); err != nil {
		s.writeAPIError(w, "无效的请求数据", http.StatusBadRequest, "")
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)
	req.Password = strings.TrimSpace(req.Password)
	req.Code = strings.TrimSpace(req.Code)

	if req.Username == "" || req.Email == "" || req.Password == "" {
		s.writeAPIError(w, "用户名、邮箱和密码不能为空", http.StatusBadRequest, "")
		return
	}

	if len(req.Username) < 2 || len(req.Username) > 50 {
		s.writeAPIError(w, "用户名长度必须在2-50个字符之间", http.StatusBadRequest, "")
		return
	}

	if len(req.Password) < 6 {
		s.writeAPIError(w, "密码长度至少6个字符", http.StatusBadRequest, "")
		return
	}

	// Verify email code if SMTP is configured
	smtpHost, _, _, _, _ := s.resolveSMTPConfig(r)
	if smtpHost != "" && req.Code != "" {
		valid, err := s.auth.VerifyCode(r.Context(), req.Email, req.Code, "register")
		if err != nil {
			s.writeAPIError(w, "验证码校验失败", http.StatusInternalServerError, "")
			return
		}
		if !valid {
			s.writeAPIError(w, "验证码无效或已过期", http.StatusBadRequest, "")
			return
		}
	}

	user, tokens, err := s.auth.Register(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		switch err {
		case auth.ErrUsernameExists:
			s.writeAPIError(w, "用户名已存在", http.StatusConflict, "USERNAME_EXISTS")
		case auth.ErrEmailExists:
			s.writeAPIError(w, "邮箱已被注册", http.StatusConflict, "EMAIL_EXISTS")
		default:
			s.writeAPIError(w, "注册失败: "+err.Error(), http.StatusInternalServerError, "")
		}
		return
	}

	s.writeAPISuccess(w, map[string]any{
		"user":   sanitizeUser(user),
		"tokens": tokens,
	}, "注册成功", http.StatusCreated)
}

// handleLogin handles user login with email and password.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}
	if s.auth == nil {
		s.writeAPIError(w, "数据库未配置", http.StatusServiceUnavailable, "")
		return
	}

	if s.siteSettingsSvc != nil {
		f, err := s.siteSettingsSvc.Get(r.Context())
		if err == nil && !f.AllowEmailLogin {
			s.writeAPIError(w, "管理员已关闭邮箱登录", http.StatusForbidden, "EMAIL_LOGIN_DISABLED")
			return
		}
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := parseJSONBody(r, &req); err != nil {
		s.writeAPIError(w, "无效的请求数据", http.StatusBadRequest, "")
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.Password = strings.TrimSpace(req.Password)

	if req.Email == "" || req.Password == "" {
		s.writeAPIError(w, "邮箱和密码不能为空", http.StatusBadRequest, "")
		return
	}

	user, tokens, err := s.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if err == auth.ErrInvalidCredentials {
			s.writeAPIError(w, "邮箱或密码错误", http.StatusUnauthorized, "INVALID_CREDENTIALS")
			return
		}
		s.writeAPIError(w, "登录失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}

	s.writeAPISuccess(w, map[string]any{
		"user":   sanitizeUser(user),
		"tokens": tokens,
	}, "登录成功", http.StatusOK)
}

// handleLinuxdoLogin initiates Linuxdo OAuth flow.
func (s *Server) handleLinuxdoLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}
	if s.auth == nil {
		s.writeAPIError(w, "数据库未配置", http.StatusServiceUnavailable, "")
		return
	}
	if !s.requireSetupInitialized(w, r) {
		return
	}

	if s.siteSettingsSvc != nil {
		f, err := s.siteSettingsSvc.Get(r.Context())
		if err == nil && !f.AllowLinuxdoLogin {
			s.writeAPIError(w, "管理员已关闭 LinuxDO 登录", http.StatusForbidden, "LINUXDO_LOGIN_DISABLED")
			return
		}
	}

	oauthCfg := s.resolveLinuxdoConfig(r)
	if !oauthCfg.Configured() {
		s.writeAPIError(w, "LinuxDO 登录尚未配置", http.StatusServiceUnavailable, "LINUXDO_NOT_CONFIGURED")
		return
	}

	state, err := auth.GenerateState()
	if err != nil {
		s.writeAPIError(w, "生成状态失败", http.StatusInternalServerError, "")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	authURL := s.auth.LinuxdoOAuthURL(state, oauthCfg)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// handleLinuxdoCallback handles OAuth callback from Linuxdo.
func (s *Server) handleLinuxdoCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}
	if s.auth == nil {
		s.writeAPIError(w, "数据库未配置", http.StatusServiceUnavailable, "")
		return
	}
	if !s.requireSetupInitialized(w, r) {
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" {
		s.writeAPIError(w, "授权码缺失", http.StatusBadRequest, "")
		return
	}

	// Verify state (simplified)
	cookie, err := r.Cookie("oauth_state")
	if err != nil || cookie.Value != state {
		s.writeAPIError(w, "无效的状态参数", http.StatusBadRequest, "")
		return
	}

	// Clear state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	oauthCfg := s.resolveLinuxdoConfig(r)
	if !oauthCfg.Configured() {
		s.writeAPIError(w, "LinuxDO 登录尚未配置", http.StatusServiceUnavailable, "LINUXDO_NOT_CONFIGURED")
		return
	}

	allowReg := true
	if s.siteSettingsSvc != nil {
		f, fErr := s.siteSettingsSvc.Get(r.Context())
		if fErr == nil {
			allowReg = f.AllowRegister && f.AllowLinuxdoRegister
		}
	}

	_, tokens, err := s.auth.LinuxdoCallback(r.Context(), code, oauthCfg, allowReg)
	if err != nil {
		if err == auth.ErrRegistrationDisabled {
			redirectURL := fmt.Sprintf("%s/login?error=%s",
				s.resolveFrontendURL(r), "register_disabled")
			http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
			return
		}
		s.writeAPIError(w, "OAuth登录失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}

	redirectURL := fmt.Sprintf("%s/auth/callback?access_token=%s&refresh_token=%s",
		s.resolveFrontendURL(r), tokens.AccessToken, tokens.RefreshToken)
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

// handleRefreshToken refreshes access token.
func (s *Server) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}
	if s.auth == nil {
		s.writeAPIError(w, "数据库未配置", http.StatusServiceUnavailable, "")
		return
	}

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := parseJSONBody(r, &req); err != nil {
		s.writeAPIError(w, "无效的请求数据", http.StatusBadRequest, "")
		return
	}

	if req.RefreshToken == "" {
		s.writeAPIError(w, "刷新令牌不能为空", http.StatusBadRequest, "")
		return
	}

	claims, err := s.auth.ValidateToken(req.RefreshToken)
	if err != nil {
		s.writeAPIError(w, "无效或过期的刷新令牌", http.StatusUnauthorized, "INVALID_REFRESH_TOKEN")
		return
	}

	user, err := s.auth.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		s.writeAPIError(w, "用户不存在", http.StatusNotFound, "")
		return
	}

	tokens, err := s.auth.GenerateTokenPair(user)
	if err != nil {
		s.writeAPIError(w, "生成令牌失败", http.StatusInternalServerError, "")
		return
	}

	s.writeAPISuccess(w, tokens, "令牌刷新成功", http.StatusOK)
}

// handleGetCurrentUser returns current authenticated user.
func (s *Server) handleGetCurrentUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}

	userID, ok := GetUserID(r.Context())
	if !ok {
		s.writeAPIError(w, "未认证", http.StatusUnauthorized, "")
		return
	}

	user, err := s.auth.GetUserByID(r.Context(), userID)
	if err != nil {
		s.writeAPIError(w, "用户不存在", http.StatusNotFound, "")
		return
	}

	s.writeAPISuccess(w, sanitizeUser(user), "获取用户信息成功", http.StatusOK)
}

// handleSendCode sends verification code to email.
func (s *Server) handleSendCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}
	if s.auth == nil {
		s.writeAPIError(w, "数据库未配置", http.StatusServiceUnavailable, "")
		return
	}

	var req struct {
		Email   string `json:"email"`
		Purpose string `json:"purpose"` // register, reset_password
	}

	if err := parseJSONBody(r, &req); err != nil {
		s.writeAPIError(w, "无效的请求数据", http.StatusBadRequest, "")
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.Purpose = strings.TrimSpace(req.Purpose)

	if req.Email == "" {
		s.writeAPIError(w, "邮箱不能为空", http.StatusBadRequest, "")
		return
	}

	if req.Purpose == "" {
		req.Purpose = "register"
	}

	if req.Purpose != "register" && req.Purpose != "reset_password" {
		s.writeAPIError(w, "无效的验证码用途", http.StatusBadRequest, "")
		return
	}
	if req.Purpose == "register" && !s.requireSetupInitialized(w, r) {
		return
	}

	// Check if SMTP is configured (DB settings or env vars)
	smtpHost, smtpPort, smtpUser, smtpPassword, smtpFrom := s.resolveSMTPConfig(r)
	if smtpHost == "" {
		s.writeAPIError(w, "邮件服务未配置", http.StatusServiceUnavailable, "")
		return
	}

	// Ensure auth service uses the resolved SMTP config
	s.auth.SetSMTPConfig(smtpHost, smtpPort, smtpUser, smtpPassword, smtpFrom)

	code, err := s.auth.SendVerificationCode(r.Context(), req.Email, req.Purpose)
	if err != nil {
		s.writeAPIError(w, "发送验证码失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}

	s.writeAPISuccess(w, map[string]any{
		"message": "验证码已发送",
		// For development, return the code (remove in production)
		"code": code,
	}, "验证码发送成功", http.StatusOK)
}

// handleUpdateProfile updates user profile.
func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}

	userID, ok := GetUserID(r.Context())
	if !ok {
		s.writeAPIError(w, "未认证", http.StatusUnauthorized, "")
		return
	}

	var req struct {
		Username string `json:"username"`
		Avatar   string `json:"avatar"`
	}

	if err := parseJSONBody(r, &req); err != nil {
		s.writeAPIError(w, "无效的请求数据", http.StatusBadRequest, "")
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Avatar = strings.TrimSpace(req.Avatar)

	if req.Username == "" {
		s.writeAPIError(w, "用户名不能为空", http.StatusBadRequest, "")
		return
	}

	var avatar *string
	if req.Avatar != "" {
		avatar = &req.Avatar
	}

	if err := s.auth.UpdateProfile(r.Context(), userID, req.Username, avatar); err != nil {
		if err == auth.ErrUsernameExists {
			s.writeAPIError(w, "用户名已存在", http.StatusConflict, "USERNAME_EXISTS")
			return
		}
		s.writeAPIError(w, "更新失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}

	user, err := s.auth.GetUserByID(r.Context(), userID)
	if err != nil {
		s.writeAPIError(w, "获取用户信息失败", http.StatusInternalServerError, "")
		return
	}
	s.writeAPISuccess(w, sanitizeUser(user), "更新成功", http.StatusOK)
}

// handleChangePassword changes user password.
func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}

	userID, ok := GetUserID(r.Context())
	if !ok {
		s.writeAPIError(w, "未认证", http.StatusUnauthorized, "")
		return
	}

	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}

	if err := parseJSONBody(r, &req); err != nil {
		s.writeAPIError(w, "无效的请求数据", http.StatusBadRequest, "")
		return
	}

	req.OldPassword = strings.TrimSpace(req.OldPassword)
	req.NewPassword = strings.TrimSpace(req.NewPassword)

	if req.OldPassword == "" || req.NewPassword == "" {
		s.writeAPIError(w, "旧密码和新密码不能为空", http.StatusBadRequest, "")
		return
	}

	if len(req.NewPassword) < 6 {
		s.writeAPIError(w, "新密码长度至少6个字符", http.StatusBadRequest, "")
		return
	}

	if err := s.auth.ChangePassword(r.Context(), userID, req.OldPassword, req.NewPassword); err != nil {
		if err == auth.ErrInvalidCredentials {
			s.writeAPIError(w, "旧密码错误", http.StatusBadRequest, "INVALID_OLD_PASSWORD")
			return
		}
		s.writeAPIError(w, "修改密码失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}

	s.writeAPISuccess(w, nil, "密码修改成功", http.StatusOK)
}

// sanitizeUser removes sensitive data from user object.
func sanitizeUser(user *auth.User) map[string]any {
	result := map[string]any{
		"id":        user.ID,
		"username":  user.Username,
		"provider":  user.Provider,
		"role":      user.Role,
		"createdAt": user.CreatedAt,
	}
	if user.Email != nil {
		result["email"] = user.Email
	}
	if user.Avatar != nil {
		result["avatar"] = user.Avatar
	}
	return result
}

// parseJSONBody parses JSON request body.
func parseJSONBody(r *http.Request, v any) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

// parseIntParam parses an integer from query parameter.
func parseIntParam(r *http.Request, key string, defaultVal int) int {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(val)
	if err != nil || n < 1 {
		return defaultVal
	}
	return n
}
