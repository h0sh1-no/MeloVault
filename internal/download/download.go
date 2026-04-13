// Package download tracks user music download history.
package download

import (
	"context"
	"time"

	"github.com/h0sh1-no/MeloVault/internal/database"
)

// History represents a download history entry.
type History struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	SongID    int64     `json:"song_id"`
	SongName  string    `json:"song_name"`
	Artists   string    `json:"artists"`
	Quality   string    `json:"quality"`
	FileType  string    `json:"file_type"`
	FileSize  int64     `json:"file_size"`
	CreatedAt time.Time `json:"created_at"`
}

// Service provides download history operations.
type Service struct {
	db *database.Pool
}

// NewService creates a new download history service.
func NewService(db *database.Pool) *Service {
	return &Service{db: db}
}

// Record records a download in history.
func (s *Service) Record(ctx context.Context, userID, songID int64, songName, artists, quality, fileType string, fileSize int64) (*History, error) {
	var h History
	err := s.db.QueryRow(ctx,
		`INSERT INTO download_history (user_id, song_id, song_name, artists, quality, file_type, file_size)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, user_id, song_id, song_name, artists, quality, file_type, file_size, created_at`,
		userID, songID, songName, artists, quality, fileType, fileSize,
	).Scan(&h.ID, &h.UserID, &h.SongID, &h.SongName, &h.Artists, &h.Quality, &h.FileType, &h.FileSize, &h.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &h, nil
}

// List lists user's download history with pagination.
func (s *Service) List(ctx context.Context, userID int64, page, pageSize int) ([]History, int64, error) {
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	var total int64
	err := s.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM download_history WHERE user_id = $1", userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := s.db.Query(ctx,
		`SELECT id, user_id, song_id, song_name, artists, quality, file_type, file_size, created_at
		 FROM download_history WHERE user_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var history []History
	for rows.Next() {
		var h History
		if err := rows.Scan(&h.ID, &h.UserID, &h.SongID, &h.SongName, &h.Artists,
			&h.Quality, &h.FileType, &h.FileSize, &h.CreatedAt); err != nil {
			return nil, 0, err
		}
		history = append(history, h)
	}

	return history, total, rows.Err()
}

// Clear clears all download history for a user.
func (s *Service) Clear(ctx context.Context, userID int64) error {
	_, err := s.db.Exec(ctx, "DELETE FROM download_history WHERE user_id = $1", userID)
	return err
}

// Delete deletes a specific download history entry.
func (s *Service) Delete(ctx context.Context, userID, historyID int64) error {
	_, err := s.db.Exec(ctx,
		"DELETE FROM download_history WHERE id = $1 AND user_id = $2", historyID, userID)
	return err
}
