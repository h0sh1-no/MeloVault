// Package accountpool manages a pool of Netease accounts for API access.
package accountpool

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/h0sh1-no/MeloVault/internal/cookie"
	"github.com/h0sh1-no/MeloVault/internal/database"
)

// Account represents a stored NetEase account in the pool.
type Account struct {
	ID           int64      `json:"id"`
	Nickname     string     `json:"nickname"`
	CookieString string     `json:"cookie_string,omitempty"`
	MusicU       string     `json:"music_u"`
	IsActive     bool       `json:"is_active"`
	LastUsedAt   *time.Time `json:"last_used_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// Service manages the NetEase account pool with round-robin rotation.
type Service struct {
	db      *database.Pool
	counter uint64
}

// NewService creates a new account pool service.
func NewService(db *database.Pool) *Service {
	return &Service{db: db}
}

// Add parses the cookie string and stores a new account in the pool.
// Returns the new account ID.
func (s *Service) Add(ctx context.Context, nickname, cookieStr string) (int64, error) {
	cookieStr = strings.TrimSpace(cookieStr)
	if cookieStr == "" {
		return 0, fmt.Errorf("cookie string is empty")
	}

	parsed := cookie.ParseCookieString(cookieStr)
	if len(parsed) == 0 {
		return 0, fmt.Errorf("cannot parse cookie string")
	}

	musicU := strings.TrimSpace(parsed["MUSIC_U"])
	// Keep compatibility with existing DB schema where music_u may be VARCHAR(500).
	// A few upstream cookie variants can exceed this length.
	if len(musicU) > 500 {
		musicU = musicU[:500]
	}

	// Deduplicate by MUSIC_U if present
	if musicU != "" {
		var existing int64
		err := s.db.QueryRow(ctx,
			`SELECT id FROM netease_accounts WHERE music_u = $1`, musicU,
		).Scan(&existing)
		if err == nil && existing > 0 {
			// Update existing account's cookie
			_, err = s.db.Exec(ctx,
				`UPDATE netease_accounts SET cookie_string = $1, is_active = TRUE, updated_at = NOW() WHERE id = $2`,
				cookieStr, existing,
			)
			if err != nil {
				return 0, fmt.Errorf("update existing account: %w", err)
			}
			return existing, nil
		}
	}

	var id int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO netease_accounts (nickname, cookie_string, music_u)
		 VALUES ($1, $2, $3) RETURNING id`,
		strings.TrimSpace(nickname), cookieStr, musicU,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert account: %w", err)
	}
	return id, nil
}

// List returns all accounts. Cookie strings are truncated for safety in list view.
func (s *Service) List(ctx context.Context) ([]Account, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, COALESCE(nickname, ''), COALESCE(music_u, ''), is_active, last_used_at, created_at, updated_at
		 FROM netease_accounts ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.ID, &a.Nickname, &a.MusicU, &a.IsActive, &a.LastUsedAt, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}
		accounts = append(accounts, a)
	}
	if accounts == nil {
		accounts = []Account{}
	}
	return accounts, rows.Err()
}

// Remove deletes an account from the pool.
func (s *Service) Remove(ctx context.Context, id int64) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM netease_accounts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete account: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("account not found")
	}
	return nil
}

// ToggleActive enables or disables an account.
func (s *Service) ToggleActive(ctx context.Context, id int64, active bool) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE netease_accounts SET is_active = $1, updated_at = NOW() WHERE id = $2`,
		active, id,
	)
	if err != nil {
		return fmt.Errorf("toggle account: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("account not found")
	}
	return nil
}

// UpdateNickname updates the display name of an account.
func (s *Service) UpdateNickname(ctx context.Context, id int64, nickname string) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE netease_accounts SET nickname = $1, updated_at = NOW() WHERE id = $2`,
		strings.TrimSpace(nickname), id,
	)
	if err != nil {
		return fmt.Errorf("update nickname: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("account not found")
	}
	return nil
}

// Next picks the next active account via round-robin and returns its parsed cookies.
// Returns nil, nil when no active accounts exist (caller should fall back).
func (s *Service) Next(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, cookie_string FROM netease_accounts WHERE is_active = TRUE ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("query active accounts: %w", err)
	}
	defer rows.Close()

	type entry struct {
		id           int64
		cookieString string
	}
	var active []entry
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.id, &e.cookieString); err != nil {
			return nil, fmt.Errorf("scan active account: %w", err)
		}
		active = append(active, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(active) == 0 {
		return nil, nil
	}

	idx := atomic.AddUint64(&s.counter, 1) % uint64(len(active))
	chosen := active[idx]

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, _ = s.db.Exec(bgCtx,
			`UPDATE netease_accounts SET last_used_at = NOW() WHERE id = $1`, chosen.id,
		)
	}()

	return cookie.ParseCookieString(chosen.cookieString), nil
}

// Count returns total and active account counts.
func (s *Service) Count(ctx context.Context) (total, active int, err error) {
	err = s.db.QueryRow(ctx,
		`SELECT COUNT(*), COUNT(*) FILTER (WHERE is_active = TRUE) FROM netease_accounts`,
	).Scan(&total, &active)
	if err != nil {
		return 0, 0, fmt.Errorf("count accounts: %w", err)
	}
	return total, active, nil
}
