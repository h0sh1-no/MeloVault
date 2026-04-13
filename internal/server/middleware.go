package server

import (
	"context"
	"net/http"
	"strings"
)

// contextKey is a type for context keys.
type contextKey string

const (
	// UserIDKey is the context key for user ID.
	UserIDKey contextKey = "userID"
	// UsernameKey is the context key for username.
	UsernameKey contextKey = "username"
)

// AuthMiddleware validates JWT token and injects user info into context.
func (s *Server) AuthMiddleware(next http.Handler) http.Handler {
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
			s.writeAPIError(w, "无效的认证格式", http.StatusUnauthorized, "INVALID_AUTH_FORMAT")
			return
		}

		tokenStr := strings.TrimSpace(parts[1])
		claims, err := s.auth.ValidateToken(tokenStr)
		if err != nil {
			s.writeAPIError(w, "无效或过期的令牌", http.StatusUnauthorized, "INVALID_TOKEN")
			return
		}

		// Inject user info into context
		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, UsernameKey, claims.Username)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuthMiddleware validates JWT token if present but doesn't require it.
func (s *Server) OptionalAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.auth == nil {
			next.ServeHTTP(w, r)
			return
		}
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			next.ServeHTTP(w, r)
			return
		}

		tokenStr := strings.TrimSpace(parts[1])
		claims, err := s.auth.ValidateToken(tokenStr)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		// Inject user info into context
		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, UsernameKey, claims.Username)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserID extracts user ID from context.
func GetUserID(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(UserIDKey).(int64)
	return id, ok
}

// GetUsername extracts username from context.
func GetUsername(ctx context.Context) (string, bool) {
	name, ok := ctx.Value(UsernameKey).(string)
	return name, ok
}

// classifyAction maps a URL path to an activity action type.
func classifyAction(path string) string {
	switch {
	case strings.HasPrefix(path, "/api/auth/login") ||
		strings.HasPrefix(path, "/api/auth/linuxdo"):
		return "login"
	case strings.HasPrefix(path, "/song") || strings.HasPrefix(path, "/Song_V1"):
		return "play"
	case strings.HasPrefix(path, "/search") || strings.HasPrefix(path, "/Search") ||
		strings.HasPrefix(path, "/api/public/search"):
		return "search"
	case strings.HasPrefix(path, "/download") || strings.HasPrefix(path, "/Download") ||
		strings.HasPrefix(path, "/api/downloads"):
		return "download"
	case strings.HasPrefix(path, "/api/favorites"):
		return "favorite"
	case strings.HasPrefix(path, "/api/playlists") || strings.HasPrefix(path, "/api/shared/playlist"):
		return "playlist"
	case strings.HasPrefix(path, "/playlist") || strings.HasPrefix(path, "/Playlist") ||
		strings.HasPrefix(path, "/album") || strings.HasPrefix(path, "/Album"):
		return "browse"
	default:
		return ""
	}
}
