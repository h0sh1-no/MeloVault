// Package activity provides user activity logging and analytics.
package activity

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/h0sh1-no/MeloVault/internal/database"
	"github.com/h0sh1-no/MeloVault/internal/ipgeo"
)

// LogEntry represents a single activity log record.
type LogEntry struct {
	ID        int64            `json:"id"`
	UserID    *int64           `json:"user_id"`
	Username  *string          `json:"username"`
	Action    string           `json:"action"`
	IP        string           `json:"ip"`
	Province  string           `json:"province"`
	City      string           `json:"city"`
	UserAgent string           `json:"user_agent"`
	Metadata  json.RawMessage  `json:"metadata"`
	CreatedAt time.Time        `json:"created_at"`
}

// ProvinceStat holds aggregated activity count per province.
type ProvinceStat struct {
	Province string `json:"province"`
	Count    int64  `json:"count"`
}

// TrendPoint holds daily aggregated activity metrics.
type TrendPoint struct {
	Date       string `json:"date"`
	PlayCount  int64  `json:"play_count"`
	DownCount  int64  `json:"down_count"`
	UserCount  int64  `json:"user_count"`
	LoginCount int64  `json:"login_count"`
}

// ActiveUser represents a recently active user with location info.
type ActiveUser struct {
	UserID   int64     `json:"user_id"`
	Username string    `json:"username"`
	Avatar   *string   `json:"avatar"`
	LastIP   string    `json:"last_ip"`
	Province string    `json:"province"`
	LastSeen time.Time `json:"last_seen"`
	Action   string    `json:"action"`
}

// OverviewStats holds dashboard-level aggregate statistics.
type OverviewStats struct {
	TotalUsers      int64 `json:"total_users"`
	TotalFavorites  int64 `json:"total_favorites"`
	TotalDownloads  int64 `json:"total_downloads"`
	TotalPlays      int64 `json:"total_plays"`
	TodayNewUsers   int64 `json:"today_new_users"`
	TodayPlays      int64 `json:"today_plays"`
	TodayDownloads  int64 `json:"today_downloads"`
	TodayActiveUsers int64 `json:"today_active_users"`
	OnlineUsers     int64 `json:"online_users"`
}

// Filters defines optional query filters for activity log retrieval.
type Filters struct {
	Action   string
	UserID   int64
	IP       string
	Search   string
}

// Service provides activity logging and analytics operations.
type Service struct {

// NewService creates an activity service with IP geolocation support.
func NewService(db *database.Pool) *Service {
	return &Service{
		db:  db,
		geo: ipgeo.NewResolver(),
	}
}

// LogActivity records a user action with IP geolocation metadata.
func (s *Service) LogActivity(ctx context.Context, userID *int64, action, ip, ua string, metadata map[string]any) {
	geo := s.geo.Resolve(ip)
	metaJSON, _ := json.Marshal(metadata)
	if metaJSON == nil {
		metaJSON = []byte("{}")
	}
	_, _ = s.db.Exec(ctx,
		`INSERT INTO activity_logs (user_id, action, ip, province, city, user_agent, metadata)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		userID, action, ip, geo.Province, geo.City, truncate(ua, 500), metaJSON)
}

// UpdateLastLogin updates the user's last login IP and timestamp.
func (s *Service) UpdateLastLogin(ctx context.Context, userID int64, ip string) {
	_, _ = s.db.Exec(ctx,
		`UPDATE users SET last_login_ip = $1, last_login_at = NOW() WHERE id = $2`,
		ip, userID)
}

// GetActivityLogs returns paginated activity logs with optional filters.
func (s *Service) GetActivityLogs(ctx context.Context, page, pageSize int, filters Filters) ([]LogEntry, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	where, args := buildWhereClause(filters)

	var total int64
	countQ := "SELECT COUNT(*) FROM activity_logs a LEFT JOIN users u ON u.id = a.user_id" + where
	if err := s.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	dataQ := `SELECT a.id, a.user_id, u.username, a.action, a.ip, a.province, a.city,
	          a.user_agent, a.metadata, a.created_at
	          FROM activity_logs a LEFT JOIN users u ON u.id = a.user_id` +
		where + fmt.Sprintf(` ORDER BY a.created_at DESC LIMIT %d OFFSET %d`, pageSize, offset)

	rows, err := s.db.Query(ctx, dataQ, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []LogEntry
	for rows.Next() {
		var l LogEntry
		if err := rows.Scan(&l.ID, &l.UserID, &l.Username, &l.Action, &l.IP,
			&l.Province, &l.City, &l.UserAgent, &l.Metadata, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, l)
	}
	if logs == nil {
		logs = []LogEntry{}
	}
	return logs, total, rows.Err()
}

// GetRecentActiveUsers returns users who were active within the given time window.
func (s *Service) GetRecentActiveUsers(ctx context.Context, minutes int) ([]ActiveUser, error) {
	if minutes <= 0 {
		minutes = 15
	}
	rows, err := s.db.Query(ctx, `
		SELECT DISTINCT ON (a.user_id)
			a.user_id, u.username, u.avatar, a.ip, a.province, a.created_at, a.action
		FROM activity_logs a
		JOIN users u ON u.id = a.user_id
		WHERE a.user_id IS NOT NULL
		  AND a.created_at >= NOW() - INTERVAL '1 minute' * $1
		ORDER BY a.user_id, a.created_at DESC`, minutes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []ActiveUser
	for rows.Next() {
		var u ActiveUser
		if err := rows.Scan(&u.UserID, &u.Username, &u.Avatar, &u.LastIP,
			&u.Province, &u.LastSeen, &u.Action); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	if users == nil {
		users = []ActiveUser{}
	}
	return users, rows.Err()
}

// GetProvinceStats returns activity counts grouped by province.
func (s *Service) GetProvinceStats(ctx context.Context, days int) ([]ProvinceStat, error) {
	if days <= 0 {
		days = 30
	}
	rows, err := s.db.Query(ctx, `
		SELECT COALESCE(province, '未知') AS province, COUNT(*) AS count
		FROM activity_logs
		WHERE created_at >= NOW() - INTERVAL '1 day' * $1
		  AND province IS NOT NULL AND province != ''
		GROUP BY province
		ORDER BY count DESC`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []ProvinceStat
	for rows.Next() {
		var st ProvinceStat
		if err := rows.Scan(&st.Province, &st.Count); err != nil {
			return nil, err
		}
		stats = append(stats, st)
	}
	if stats == nil {
		stats = []ProvinceStat{}
	}
	return stats, rows.Err()
}

// GetTrends returns daily activity trend data over the given number of days.
func (s *Service) GetTrends(ctx context.Context, days int) ([]TrendPoint, error) {
	if days <= 0 {
		days = 7
	}
	rows, err := s.db.Query(ctx, `
		WITH dates AS (
			SELECT generate_series(
				(CURRENT_DATE - ($1 - 1) * INTERVAL '1 day')::date,
				CURRENT_DATE::date,
				'1 day'::interval
			)::date AS d
		)
		SELECT
			d::text AS date,
			COALESCE(SUM(CASE WHEN a.action = 'play' THEN 1 ELSE 0 END), 0) AS play_count,
			COALESCE(SUM(CASE WHEN a.action = 'download' THEN 1 ELSE 0 END), 0) AS down_count,
			(SELECT COUNT(*) FROM users WHERE created_at::date = d) AS user_count,
			COALESCE(SUM(CASE WHEN a.action = 'login' THEN 1 ELSE 0 END), 0) AS login_count
		FROM dates
		LEFT JOIN activity_logs a ON a.created_at::date = d
		GROUP BY d
		ORDER BY d ASC`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trends []TrendPoint
	for rows.Next() {
		var t TrendPoint
		if err := rows.Scan(&t.Date, &t.PlayCount, &t.DownCount, &t.UserCount, &t.LoginCount); err != nil {
			return nil, err
		}
		trends = append(trends, t)
	}
	if trends == nil {
		trends = []TrendPoint{}
	}
	return trends, rows.Err()
}

// GetUserActivity returns paginated activity for a specific user.
func (s *Service) GetUserActivity(ctx context.Context, userID int64, page, pageSize int) ([]LogEntry, int64, error) {
	return s.GetUserActivityFiltered(ctx, userID, "", page, pageSize)
}

// GetUserActivityFiltered returns paginated activity for a user, optionally filtered by action type.
func (s *Service) GetUserActivityFiltered(ctx context.Context, userID int64, action string, page, pageSize int) ([]LogEntry, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int64
	if action != "" {
		if err := s.db.QueryRow(ctx,
			"SELECT COUNT(*) FROM activity_logs WHERE user_id = $1 AND action = $2",
			userID, action).Scan(&total); err != nil {
			return nil, 0, err
		}
	} else {
		if err := s.db.QueryRow(ctx,
			"SELECT COUNT(*) FROM activity_logs WHERE user_id = $1", userID).Scan(&total); err != nil {
			return nil, 0, err
		}
	}

	var rows pgx.Rows
	var err error
	baseQuery := `
		SELECT a.id, a.user_id, u.username, a.action, a.ip, a.province, a.city,
		       a.user_agent, a.metadata, a.created_at
		FROM activity_logs a
		LEFT JOIN users u ON u.id = a.user_id`
	if action != "" {
		rows, err = s.db.Query(ctx, baseQuery+`
			WHERE a.user_id = $1 AND a.action = $2
			ORDER BY a.created_at DESC
			LIMIT $3 OFFSET $4`, userID, action, pageSize, offset)
	} else {
		rows, err = s.db.Query(ctx, baseQuery+`
			WHERE a.user_id = $1
			ORDER BY a.created_at DESC
			LIMIT $2 OFFSET $3`, userID, pageSize, offset)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []LogEntry
	for rows.Next() {
		var l LogEntry
		if err := rows.Scan(&l.ID, &l.UserID, &l.Username, &l.Action, &l.IP,
			&l.Province, &l.City, &l.UserAgent, &l.Metadata, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, l)
	}
	if logs == nil {
		logs = []LogEntry{}
	}
	return logs, total, rows.Err()
}

// GetOverviewStats returns aggregate dashboard statistics.
func (s *Service) GetOverviewStats(ctx context.Context) (*OverviewStats, error) {
	var st OverviewStats
	queries := []struct {
		q   string
		dst *int64
	}{
		{"SELECT COUNT(*) FROM users", &st.TotalUsers},
		{"SELECT COUNT(*) FROM favorites", &st.TotalFavorites},
		{"SELECT COUNT(*) FROM download_history", &st.TotalDownloads},
		{"SELECT COUNT(*) FROM activity_logs WHERE action = 'play'", &st.TotalPlays},
		{"SELECT COUNT(*) FROM users WHERE created_at >= CURRENT_DATE", &st.TodayNewUsers},
		{"SELECT COUNT(*) FROM activity_logs WHERE action = 'play' AND created_at >= CURRENT_DATE", &st.TodayPlays},
		{"SELECT COUNT(*) FROM activity_logs WHERE action = 'download' AND created_at >= CURRENT_DATE", &st.TodayDownloads},
		{"SELECT COUNT(DISTINCT user_id) FROM activity_logs WHERE user_id IS NOT NULL AND created_at >= CURRENT_DATE", &st.TodayActiveUsers},
		{"SELECT COUNT(DISTINCT user_id) FROM activity_logs WHERE user_id IS NOT NULL AND created_at >= NOW() - INTERVAL '15 minutes'", &st.OnlineUsers},
	}
	for _, q := range queries {
		if err := s.db.QueryRow(ctx, q.q).Scan(q.dst); err != nil {
			if err == pgx.ErrNoRows {
				*q.dst = 0
				continue
			}
			return nil, err
		}
	}
	return &st, nil
}

func buildWhereClause(f Filters) (string, []any) {
	var conditions []string
	var args []any
	idx := 1

	if f.Action != "" {
		conditions = append(conditions, fmt.Sprintf("a.action = $%d", idx))
		args = append(args, f.Action)
		idx++
	}
	if f.UserID > 0 {
		conditions = append(conditions, fmt.Sprintf("a.user_id = $%d", idx))
		args = append(args, f.UserID)
		idx++
	}
	if f.IP != "" {
		conditions = append(conditions, fmt.Sprintf("a.ip = $%d", idx))
		args = append(args, f.IP)
		idx++
	}
	if f.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(u.username ILIKE $%d OR a.ip ILIKE $%d)", idx, idx))
		args = append(args, "%"+f.Search+"%")
		idx++
	}

	if len(conditions) == 0 {
		return "", nil
	}
	where := " WHERE "
	for i, c := range conditions {
		if i > 0 {
			where += " AND "
		}
		where += c
	}
	return where, args
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
