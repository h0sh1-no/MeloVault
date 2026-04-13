package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// handleGetSiteSettings returns public feature flags (secrets stripped).
func (s *Server) handleGetSiteSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}
	features, err := s.siteSettingsSvc.Get(r.Context())
	if err != nil {
		s.writeAPIError(w, "获取站点设置失败", http.StatusInternalServerError, "")
		return
	}

	// Also check env fallback to determine linuxdo_configured
	pub := features.PublicView()
	if configured, ok := pub["linuxdo_configured"].(bool); !configured || !ok {
		if s.cfg.LinuxdoClientID != "" && s.cfg.LinuxdoClientSecret != "" && s.cfg.LinuxdoRedirectURI != "" {
			pub["linuxdo_configured"] = true
		}
	}
	// Also check env fallback to determine smtp_configured
	if configured, ok := pub["smtp_configured"].(bool); !configured || !ok {
		if s.cfg.SMTPHost != "" && s.cfg.SMTPUser != "" && s.cfg.SMTPPassword != "" {
			pub["smtp_configured"] = true
		}
	}

	s.writeAPISuccess(w, pub, "ok", http.StatusOK)
}

// handleAdminSiteSettings handles GET and PUT for admin site settings.
func (s *Server) handleAdminSiteSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleAdminGetSiteSettings(w, r)
	case http.MethodPut:
		s.handleUpdateSiteSettings(w, r)
	default:
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
	}
}

// handleAdminGetSiteSettings returns full settings including OAuth credentials.
func (s *Server) handleAdminGetSiteSettings(w http.ResponseWriter, r *http.Request) {
	features, err := s.siteSettingsSvc.Get(r.Context())
	if err != nil {
		s.writeAPIError(w, "获取站点设置失败", http.StatusInternalServerError, "")
		return
	}
	s.writeAPISuccess(w, features, "ok", http.StatusOK)
}

func (s *Server) handleUpdateSiteSettings(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		s.writeAPIError(w, "无效的请求数据", http.StatusBadRequest, "")
		return
	}

	var partial map[string]any
	if err := json.Unmarshal(body, &partial); err != nil {
		s.writeAPIError(w, "无效的JSON数据", http.StatusBadRequest, "")
		return
	}

	features, err := s.siteSettingsSvc.Update(r.Context(), partial)
	if err != nil {
		s.writeAPIError(w, "更新站点设置失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}

	// Sync Netease Real IP to the API client at runtime.
	s.api.SetRealIP(features.NeteaseRealIP)

	// Sync SMTP config to auth service at runtime.
	if features.SmtpConfigured() {
		s.auth.SetSMTPConfig(features.SMTPHost, features.SMTPPort, features.SMTPUser, features.SMTPPassword, features.SMTPFrom)
	}

	s.writeAPISuccess(w, features, "站点设置已保存", http.StatusOK)
}

// handleAdminTestEmail sends a test email using current SMTP configuration.
func (s *Server) handleAdminTestEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}

	var req struct {
		Email string `json:"email"`
	}
	if err := parseJSONBody(r, &req); err != nil {
		s.writeAPIError(w, "无效的请求数据", http.StatusBadRequest, "")
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" {
		s.writeAPIError(w, "邮箱地址不能为空", http.StatusBadRequest, "")
		return
	}

	// Resolve and apply SMTP config before sending
	smtpHost, smtpPort, smtpUser, smtpPassword, smtpFrom := s.resolveSMTPConfig(r)
	if smtpHost == "" {
		s.writeAPIError(w, "SMTP 未配置，请先保存 SMTP 配置或设置环境变量", http.StatusBadRequest, "")
		return
	}
	s.auth.SetSMTPConfig(smtpHost, smtpPort, smtpUser, smtpPassword, smtpFrom)

	if err := s.auth.SendTestEmail(req.Email); err != nil {
		s.writeAPIError(w, "发送测试邮件失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}

	s.writeAPISuccess(w, nil, "测试邮件已发送", http.StatusOK)
}
