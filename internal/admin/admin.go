// Package admin provides admin-level user and system management.
package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
	"github.com/h0sh1-no/MeloVault/internal/database"
)

// UserDetail is the admin view of a user with aggregate stats.
type UserDetail struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     *string   `json:"email"`
	Avatar    *string   `json:"avatar"`
	Provider  string    `json:"provider"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	FavCount  int64     `json:"fav_count"`
	DownCount int64     `json:"down_count"`
}

// DownloadRecord extends download history with username info.
type DownloadRecord struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	SongID    int64     `json:"song_id"`
	SongName  string    `json:"song_name"`
	Artists   string    `json:"artists"`
	Quality   string    `json:"quality"`
	FileType  string    `json:"file_type"`
	FileSize  int64     `json:"file_size"`
	CreatedAt time.Time `json:"created_at"`
}

// Stats holds system-wide statistics.
type Stats struct {
	TotalUsers     int64 `json:"total_users"`
	TotalFavorites int64 `json:"total_favorites"`
	TotalDownloads int64 `json:"total_downloads"`
	TodayNewUsers  int64 `json:"today_new_users"`
}

// Service provides admin operations.
type Service struct {
	db *database.Pool
}

// NewService creates a new admin service.
func NewService(db *database.Pool) *Service {
	return &Service{db: db}
}

// IsAnyUserExists checks whether any user account has been created.
func (s *Service) IsAnyUserExists(ctx context.Context) (bool, error) {
	var exists bool
	err := s.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM users)").Scan(&exists)
	return exists, err
}

// InitSuperAdmin creates the first superadmin account and returns the new user ID.
// Returns an error if any account already exists.
func (s *Service) InitSuperAdmin(ctx context.Context, username, email, password string) (int64, error) {
	exists, err := s.IsAnyUserExists(ctx)
	if err != nil {
		return 0, fmt.Errorf("check users: %w", err)
	}
	if exists {
		return 0, fmt.Errorf("system already initialized")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("hash password: %w", err)
	}

	var userID int64
	err = s.db.QueryRow(ctx,
		`INSERT INTO users (username, email, password_hash, provider, role)
		 VALUES ($1, $2, $3, 'email', 'superadmin')
		 RETURNING id`,
		username, email, string(hash),
	).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("insert superadmin: %w", err)
	}
	return userID, nil
}

// ListUsers returns a paginated, searchable list of users with stats.
func (s *Service) ListUsers(ctx context.Context, page, pageSize int, search string) ([]UserDetail, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int64
	if search != "" {
		err := s.db.QueryRow(ctx,
			"SELECT COUNT(*) FROM users WHERE username ILIKE $1 OR COALESCE(email,'') ILIKE $1",
			"%"+search+"%").Scan(&total)
		if err != nil {
			return nil, 0, err
		}
	} else {
		if err := s.db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&total); err != nil {
			return nil, 0, err
		}
	}

	baseQuery := `
		SELECT u.id, u.username, u.email, u.avatar, u.provider, u.role, u.created_at,
		       COUNT(DISTINCT f.id) AS fav_count,
		       COUNT(DISTINCT d.id) AS down_count
		FROM users u
		LEFT JOIN favorites f ON f.user_id = u.id
		LEFT JOIN download_history d ON d.user_id = u.id`

	var rows pgx.Rows
	var err error
	if search != "" {
		rows, err = s.db.Query(ctx, baseQuery+`
		WHERE u.username ILIKE $1 OR COALESCE(u.email,'') ILIKE $1
		GROUP BY u.id ORDER BY u.created_at DESC LIMIT $2 OFFSET $3`,
			"%"+search+"%", pageSize, offset)
	} else {
		rows, err = s.db.Query(ctx, baseQuery+`
		GROUP BY u.id ORDER BY u.created_at DESC LIMIT $1 OFFSET $2`,
			pageSize, offset)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []UserDetail
	for rows.Next() {
		var u UserDetail
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Avatar, &u.Provider, &u.Role,
			&u.CreatedAt, &u.FavCount, &u.DownCount); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	if users == nil {
		users = []UserDetail{}
	}
	return users, total, rows.Err()
}

// GetUser fetches a single user's detail.
func (s *Service) GetUser(ctx context.Context, id int64) (*UserDetail, error) {
	var u UserDetail
	err := s.db.QueryRow(ctx, `
		SELECT u.id, u.username, u.email, u.avatar, u.provider, u.role, u.created_at,
		       COUNT(DISTINCT f.id) AS fav_count,
		       COUNT(DISTINCT d.id) AS down_count
		FROM users u
		LEFT JOIN favorites f ON f.user_id = u.id
		LEFT JOIN download_history d ON d.user_id = u.id
		WHERE u.id = $1
		GROUP BY u.id`, id,
	).Scan(&u.ID, &u.Username, &u.Email, &u.Avatar, &u.Provider, &u.Role,
		&u.CreatedAt, &u.FavCount, &u.DownCount)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// UpdateUser updates a user's username and role.
func (s *Service) UpdateUser(ctx context.Context, id int64, username, role string) error {
	var currentRole string
	if err := s.db.QueryRow(ctx, "SELECT role FROM users WHERE id = $1", id).Scan(&currentRole); err != nil {
		return err
	}
	if currentRole == "superadmin" && role != "superadmin" {
		return fmt.Errorf("cannot change superadmin role")
	}
	_, err := s.db.Exec(ctx,
		"UPDATE users SET username = $1, role = $2, updated_at = NOW() WHERE id = $3",
		username, role, id)
	return err
}

// DeleteUser removes a user that is NOT superadmin.
func (s *Service) DeleteUser(ctx context.Context, id int64) error {
	var role string
	err := s.db.QueryRow(ctx, "SELECT role FROM users WHERE id = $1", id).Scan(&role)
	if err != nil {
		return fmt.Errorf("user not found")
	}
	if role == "superadmin" {
		return fmt.Errorf("cannot delete superadmin account")
	}
	_, err = s.db.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	return err
}

// CreateUser creates a new user with the given username, email, password, and role.
func (s *Service) CreateUser(ctx context.Context, username, email, password, role string) (int64, error) {
	var exists bool
	if err := s.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)", username).Scan(&exists); err != nil {
		return 0, fmt.Errorf("check username: %w", err)
	}
	if exists {
		return 0, fmt.Errorf("username already exists")
	}

	if email != "" {
		if err := s.db.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", email).Scan(&exists); err != nil {
			return 0, fmt.Errorf("check email: %w", err)
		}
		if exists {
			return 0, fmt.Errorf("email already exists")
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("hash password: %w", err)
	}

	var userID int64
	var emailVal *string
	if email != "" {
		emailVal = &email
	}
	err = s.db.QueryRow(ctx,
		`INSERT INTO users (username, email, password_hash, provider, role)
		 VALUES ($1, $2, $3, 'email', $4)
		 RETURNING id`,
		username, emailVal, string(hash), role,
	).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("insert user: %w", err)
	}
	return userID, nil
}

// ResetPassword sets a new password for a user (admin operation, no old password required).
func (s *Service) ResetPassword(ctx context.Context, id int64, newPassword string) error {
	var provider string
	err := s.db.QueryRow(ctx, "SELECT provider FROM users WHERE id = $1", id).Scan(&provider)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	_, err = s.db.Exec(ctx,
		"UPDATE users SET password_hash = $1, provider = 'email', updated_at = NOW() WHERE id = $2",
		string(hash), id)
	return err
}

// GetUserDownloads returns download history for a specific user.
func (s *Service) GetUserDownloads(ctx context.Context, userID int64, page, pageSize int) ([]DownloadRecord, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int64
	if err := s.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM download_history WHERE user_id = $1", userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.Query(ctx, `
		SELECT d.id, d.user_id, u.username, d.song_id, d.song_name, d.artists,
		       d.quality, d.file_type, d.file_size, d.created_at
		FROM download_history d
		JOIN users u ON u.id = d.user_id
		WHERE d.user_id = $1
		ORDER BY d.created_at DESC LIMIT $2 OFFSET $3`, userID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var records []DownloadRecord
	for rows.Next() {
		var r DownloadRecord
		if err := rows.Scan(&r.ID, &r.UserID, &r.Username, &r.SongID, &r.SongName,
			&r.Artists, &r.Quality, &r.FileType, &r.FileSize, &r.CreatedAt); err != nil {
			return nil, 0, err
		}
		records = append(records, r)
	}
	if records == nil {
		records = []DownloadRecord{}
	}
	return records, total, rows.Err()
}

// GetStats returns system-wide aggregate statistics.
func (s *Service) GetStats(ctx context.Context) (*Stats, error) {
	var st Stats
	queries := []struct {
		q   string
		dst *int64
	}{
		{"SELECT COUNT(*) FROM users", &st.TotalUsers},
		{"SELECT COUNT(*) FROM favorites", &st.TotalFavorites},
		{"SELECT COUNT(*) FROM download_history", &st.TotalDownloads},
		{"SELECT COUNT(*) FROM users WHERE created_at >= CURRENT_DATE", &st.TodayNewUsers},
	}
	for _, q := range queries {
		if err := s.db.QueryRow(ctx, q.q).Scan(q.dst); err != nil {
			return nil, err
		}
	}
	return &st, nil
}

// GetDownloads returns a paginated list of all download records with username.
func (s *Service) GetDownloads(ctx context.Context, page, pageSize int, search string) ([]DownloadRecord, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int64
	if search != "" {
		if err := s.db.QueryRow(ctx,
			"SELECT COUNT(*) FROM download_history d JOIN users u ON u.id=d.user_id WHERE d.song_name ILIKE $1 OR u.username ILIKE $1",
			"%"+search+"%").Scan(&total); err != nil {
			return nil, 0, err
		}
	} else {
		if err := s.db.QueryRow(ctx, "SELECT COUNT(*) FROM download_history").Scan(&total); err != nil {
			return nil, 0, err
		}
	}

	base := `SELECT d.id, d.user_id, u.username, d.song_id, d.song_name, d.artists,
	         d.quality, d.file_type, d.file_size, d.created_at
	         FROM download_history d JOIN users u ON u.id = d.user_id`
	var rows pgx.Rows
	var err error
	if search != "" {
		rows, err = s.db.Query(ctx, base+
			` WHERE d.song_name ILIKE $1 OR u.username ILIKE $1
			 ORDER BY d.created_at DESC LIMIT $2 OFFSET $3`,
			"%"+search+"%", pageSize, offset)
	} else {
		rows, err = s.db.Query(ctx, base+
			` ORDER BY d.created_at DESC LIMIT $1 OFFSET $2`,
			pageSize, offset)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var records []DownloadRecord
	for rows.Next() {
		var r DownloadRecord
		if err := rows.Scan(&r.ID, &r.UserID, &r.Username, &r.SongID, &r.SongName,
			&r.Artists, &r.Quality, &r.FileType, &r.FileSize, &r.CreatedAt); err != nil {
			return nil, 0, err
		}
		records = append(records, r)
	}
	if records == nil {
		records = []DownloadRecord{}
	}
	return records, total, rows.Err()
}
