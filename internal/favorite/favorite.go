// Package favorite manages user song favorites.
package favorite

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/h0sh1-no/MeloVault/internal/database"
)

var ErrAlreadyFavorited = errors.New("song already favorited")
var ErrNotFavorited = errors.New("song not in favorites")

// Favorite represents a user's favorite song.
type Favorite struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	SongID    int64     `json:"song_id"`
	SongName  string    `json:"song_name"`
	Artists   string    `json:"artists"`
	Album     string    `json:"album"`
	PicURL    string    `json:"pic_url"`
	CreatedAt time.Time `json:"created_at"`
}

// Service provides favorite operations.
type Service struct {
	db *database.Pool
}

// NewService creates a new favorite service.
func NewService(db *database.Pool) *Service {
	return &Service{db: db}
}

// Add adds a song to user's favorites.
func (s *Service) Add(ctx context.Context, userID, songID int64, songName, artists, album, picURL string) (*Favorite, error) {
	picURL = normalizeNeteaseImageURL(picURL)
	var fav Favorite
	err := s.db.QueryRow(ctx,
		`INSERT INTO favorites (user_id, song_id, song_name, artists, album, pic_url)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (user_id, song_id) DO UPDATE SET song_name = $3, artists = $4, album = $5, pic_url = $6
		 RETURNING id, user_id, song_id, song_name, artists, album, pic_url, created_at`,
		userID, songID, songName, artists, album, picURL,
	).Scan(&fav.ID, &fav.UserID, &fav.SongID, &fav.SongName, &fav.Artists, &fav.Album, &fav.PicURL, &fav.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &fav, nil
}

// Remove removes a song from user's favorites.
func (s *Service) Remove(ctx context.Context, userID, songID int64) error {
	result, err := s.db.Exec(ctx,
		"DELETE FROM favorites WHERE user_id = $1 AND song_id = $2", userID, songID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFavorited
	}
	return nil
}

// IsFavorited checks if a song is in user's favorites.
func (s *Service) IsFavorited(ctx context.Context, userID, songID int64) (bool, error) {
	var exists bool
	err := s.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM favorites WHERE user_id = $1 AND song_id = $2)",
		userID, songID).Scan(&exists)
	return exists, err
}

// List lists user's favorites with pagination.
func (s *Service) List(ctx context.Context, userID int64, page, pageSize int) ([]Favorite, int64, error) {
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	var total int64
	err := s.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM favorites WHERE user_id = $1", userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := s.db.Query(ctx,
		`SELECT id, user_id, song_id, song_name, artists, album, pic_url, created_at
		 FROM favorites WHERE user_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var favorites []Favorite
	for rows.Next() {
		var fav Favorite
		if err := rows.Scan(&fav.ID, &fav.UserID, &fav.SongID, &fav.SongName,
			&fav.Artists, &fav.Album, &fav.PicURL, &fav.CreatedAt); err != nil {
			return nil, 0, err
		}
		fav.PicURL = normalizeNeteaseImageURL(fav.PicURL)
		favorites = append(favorites, fav)
	}

	return favorites, total, rows.Err()
}

// GetBySongIDs checks multiple songs if they are favorited.
func (s *Service) GetBySongIDs(ctx context.Context, userID int64, songIDs []int64) (map[int64]bool, error) {
	result := make(map[int64]bool)
	if len(songIDs) == 0 {
		return result, nil
	}

	rows, err := s.db.Query(ctx,
		"SELECT song_id FROM favorites WHERE user_id = $1 AND song_id = ANY($2)",
		userID, songIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var songID int64
		if err := rows.Scan(&songID); err != nil {
			return nil, err
		}
		result[songID] = true
	}

	return result, rows.Err()
}

var _ = pgx.ErrNoRows

func normalizeNeteaseImageURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "//") && strings.Contains(lower, ".music.126.net/") {
		return "https:" + raw
	}
	if strings.HasPrefix(lower, "http://") && strings.Contains(lower, ".music.126.net/") {
		return "https://" + raw[len("http://"):]
	}
	return raw
}
