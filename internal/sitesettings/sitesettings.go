// Package sitesettings manages global site configuration and feature flags.
package sitesettings

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"github.com/h0sh1-no/MeloVault/internal/database"
)

// Features holds global site configuration and feature flags.
type Features struct {
	PlaylistParseEnabled   bool `json:"playlist_parse_enabled"`
	PlaylistParseAdminOnly bool `json:"playlist_parse_admin_only"`
	AlbumParseEnabled      bool `json:"album_parse_enabled"`
	AlbumParseAdminOnly    bool `json:"album_parse_admin_only"`

	AllowRegister        bool `json:"allow_register"`
	AllowEmailRegister   bool `json:"allow_email_register"`
	AllowLinuxdoRegister bool `json:"allow_linuxdo_register"`
	AllowEmailLogin      bool `json:"allow_email_login"`
	AllowLinuxdoLogin    bool `json:"allow_linuxdo_login"`

	LinuxdoClientID     string `json:"linuxdo_client_id"`
	LinuxdoClientSecret string `json:"linuxdo_client_secret"`
	LinuxdoRedirectURI  string `json:"linuxdo_redirect_uri"`

	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUser     string `json:"smtp_user"`
	SMTPPassword string `json:"smtp_password"`
	SMTPFrom     string `json:"smtp_from"`

	SiteURL string `json:"site_url"`

	NeteaseRealIP string `json:"netease_real_ip"`
}

// LinuxdoConfigured reports whether all three LinuxDO OAuth fields are set.
func (f *Features) LinuxdoConfigured() bool {
	return f.LinuxdoClientID != "" && f.LinuxdoClientSecret != "" && f.LinuxdoRedirectURI != ""
}

// SmtpConfigured reports whether all required SMTP fields are set.
func (f *Features) SmtpConfigured() bool {
	return f.SMTPHost != "" && f.SMTPUser != "" && f.SMTPPassword != ""
}

// PublicView returns a copy safe for unauthenticated callers (no secrets).
func (f *Features) PublicView() map[string]any {
	return map[string]any{
		"playlist_parse_enabled":   f.PlaylistParseEnabled,
		"playlist_parse_admin_only": f.PlaylistParseAdminOnly,
		"album_parse_enabled":      f.AlbumParseEnabled,
		"album_parse_admin_only":   f.AlbumParseAdminOnly,
		"allow_register":           f.AllowRegister,
		"allow_email_register":     f.AllowEmailRegister,
		"allow_linuxdo_register":   f.AllowLinuxdoRegister,
		"allow_email_login":        f.AllowEmailLogin,
		"allow_linuxdo_login":      f.AllowLinuxdoLogin,
		"linuxdo_configured":       f.LinuxdoConfigured(),
		"smtp_configured":          f.SmtpConfigured(),
		"site_url":                 f.SiteURL,
	}
}

// DefaultFeatures returns the default site feature configuration.
func DefaultFeatures() Features {
	return Features{
		PlaylistParseEnabled:   true,
		PlaylistParseAdminOnly: false,
		AlbumParseEnabled:      true,
		AlbumParseAdminOnly:    false,

		AllowRegister:        true,
		AllowEmailRegister:   true,
		AllowLinuxdoRegister: true,
		AllowEmailLogin:      true,
		AllowLinuxdoLogin:    true,

		SMTPPort: 587,
	}
}

// Service provides cached access to site settings.
type Service struct {(db *database.Pool) *Service {
	return &Service{db: db}
}

func (s *Service) Get(ctx context.Context) (*Features, error) {
	s.mu.RLock()
	if s.cache != nil {
		f := *s.cache
		s.mu.RUnlock()
		return &f, nil
	}
	s.mu.RUnlock()

	return s.load(ctx)
}

func (s *Service) load(ctx context.Context) (*Features, error) {
	var raw []byte
	err := s.db.QueryRow(ctx, `SELECT settings FROM site_settings WHERE id = 1`).Scan(&raw)
	if err != nil {
		f := DefaultFeatures()
		return &f, nil
	}

	f := DefaultFeatures()
	if err := json.Unmarshal(raw, &f); err != nil {
		f = DefaultFeatures()
	}

	s.mu.Lock()
	s.cache = &f
	s.mu.Unlock()

	return &f, nil
}

// Update applies partial changes to the site settings.
func (s *Service) Update(ctx context.Context, partial map[string]any) (*Features, error) {
	current, err := s.load(ctx)
	if err != nil {
		return nil, err
	}

	if v, ok := partial["playlist_parse_enabled"].(bool); ok {
		current.PlaylistParseEnabled = v
	}
	if v, ok := partial["playlist_parse_admin_only"].(bool); ok {
		current.PlaylistParseAdminOnly = v
	}
	if v, ok := partial["album_parse_enabled"].(bool); ok {
		current.AlbumParseEnabled = v
	}
	if v, ok := partial["album_parse_admin_only"].(bool); ok {
		current.AlbumParseAdminOnly = v
	}
	if v, ok := partial["allow_register"].(bool); ok {
		current.AllowRegister = v
	}
	if v, ok := partial["allow_email_register"].(bool); ok {
		current.AllowEmailRegister = v
	}
	if v, ok := partial["allow_linuxdo_register"].(bool); ok {
		current.AllowLinuxdoRegister = v
	}
	if v, ok := partial["allow_email_login"].(bool); ok {
		current.AllowEmailLogin = v
	}
	if v, ok := partial["allow_linuxdo_login"].(bool); ok {
		current.AllowLinuxdoLogin = v
	}
	if v, ok := partial["linuxdo_client_id"].(string); ok {
		current.LinuxdoClientID = v
	}
	if v, ok := partial["linuxdo_client_secret"].(string); ok && v != "" {
		current.LinuxdoClientSecret = v
	}
	if v, ok := partial["linuxdo_redirect_uri"].(string); ok {
		current.LinuxdoRedirectURI = v
	}
	if v, ok := partial["site_url"].(string); ok {
		current.SiteURL = strings.TrimRight(v, "/")
	}
	if v, ok := partial["netease_real_ip"].(string); ok {
		current.NeteaseRealIP = strings.TrimSpace(v)
	}
	if v, ok := partial["smtp_host"].(string); ok {
		current.SMTPHost = strings.TrimSpace(v)
	}
	if v, ok := partial["smtp_port"]; ok {
		switch p := v.(type) {
		case float64:
			current.SMTPPort = int(p)
		case int:
			current.SMTPPort = p
		}
	}
	if v, ok := partial["smtp_user"].(string); ok {
		current.SMTPUser = strings.TrimSpace(v)
	}
	if v, ok := partial["smtp_password"].(string); ok && v != "" {
		current.SMTPPassword = v
	}
	if v, ok := partial["smtp_from"].(string); ok {
		current.SMTPFrom = strings.TrimSpace(v)
	}

	raw, err := json.Marshal(current)
	if err != nil {
		return nil, err
	}

	_, err = s.db.Exec(ctx,
		`UPDATE site_settings SET settings = $1, updated_at = NOW() WHERE id = 1`, raw)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.cache = current
	s.mu.Unlock()

	return current, nil
}

// Invalidate clears the cache so the next Get will reload from DB.
func (s *Service) Invalidate() {
	s.mu.Lock()
	s.cache = nil
	s.mu.Unlock()
}
