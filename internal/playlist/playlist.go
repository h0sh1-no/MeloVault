// Package playlist manages user-created playlists and their songs.
package playlist

import (
	"context"
	"errors"
	"time"

	"github.com/h0sh1-no/MeloVault/internal/database"
)

var (
	ErrNotFound    = errors.New("playlist not found")
	ErrNotOwner    = errors.New("not playlist owner")
	ErrSongExists  = errors.New("song already in playlist")
	ErrSongMissing = errors.New("song not in playlist")
	ErrLimitExceed = errors.New("playlist limit exceeded")
)

// Playlist represents a user-created playlist.
type Playlist struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CoverURL    string    `json:"cover_url"`
	IsPublic    bool      `json:"is_public"`
	SongCount   int       `json:"song_count"`
	Creator     string    `json:"creator,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Song represents a song entry within a playlist.
type Song struct {
	ID         int64     `json:"id"`
	PlaylistID int64     `json:"playlist_id"`
	SongID     int64     `json:"song_id"`
	SongName   string    `json:"song_name"`
	Artists    string    `json:"artists"`
	Album      string    `json:"album"`
	PicURL     string    `json:"pic_url"`
	Position   int       `json:"position"`
	CreatedAt  time.Time `json:"created_at"`
}

// Service provides playlist CRUD operations.
type Service struct {
	return &Service{db: db}
}

const maxPlaylistsPerUser = 50

// Create creates a new playlist for the given user.
func (s *Service) Create(ctx context.Context, userID int64, name, description, coverURL string) (*Playlist, error) {
	var count int
	err := s.db.QueryRow(ctx, "SELECT COUNT(*) FROM playlists WHERE user_id = $1", userID).Scan(&count)
	if err != nil {
		return nil, err
	}
	if count >= maxPlaylistsPerUser {
		return nil, ErrLimitExceed
	}

	var p Playlist
	err = s.db.QueryRow(ctx,
		`INSERT INTO playlists (user_id, name, description, cover_url)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, user_id, name, description, cover_url, is_public, created_at, updated_at`,
		userID, name, description, coverURL,
	).Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.CoverURL, &p.IsPublic, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	p.SongCount = 0
	return &p, nil
}

// Update modifies a playlist's metadata.
func (s *Service) Update(ctx context.Context, userID, playlistID int64, name, description, coverURL string, isPublic *bool) (*Playlist, error) {
	var ownerID int64
	err := s.db.QueryRow(ctx, "SELECT user_id FROM playlists WHERE id = $1", playlistID).Scan(&ownerID)
	if err != nil {
		return nil, ErrNotFound
	}
	if ownerID != userID {
		return nil, ErrNotOwner
	}

	var p Playlist
	err = s.db.QueryRow(ctx,
		`UPDATE playlists SET
			name = COALESCE(NULLIF($1, ''), name),
			description = CASE WHEN $2::text IS NOT NULL THEN $2 ELSE description END,
			cover_url = CASE WHEN $3::text IS NOT NULL THEN $3 ELSE cover_url END,
			is_public = COALESCE($4, is_public),
			updated_at = NOW()
		 WHERE id = $5
		 RETURNING id, user_id, name, description, cover_url, is_public, created_at, updated_at`,
		name, description, coverURL, isPublic, playlistID,
	).Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.CoverURL, &p.IsPublic, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}

	err = s.db.QueryRow(ctx, "SELECT COUNT(*) FROM playlist_songs WHERE playlist_id = $1", playlistID).Scan(&p.SongCount)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Delete removes a playlist owned by the given user.
func (s *Service) Delete(ctx context.Context, userID, playlistID int64) error {
	var ownerID int64
	err := s.db.QueryRow(ctx, "SELECT user_id FROM playlists WHERE id = $1", playlistID).Scan(&ownerID)
	if err != nil {
		return ErrNotFound
	}
	if ownerID != userID {
		return ErrNotOwner
	}
	_, err = s.db.Exec(ctx, "DELETE FROM playlists WHERE id = $1", playlistID)
	return err
}

// ListByUser returns paginated playlists for a user.
func (s *Service) ListByUser(ctx context.Context, userID int64, page, pageSize int) ([]Playlist, int64, error) {
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	var total int64
	err := s.db.QueryRow(ctx, "SELECT COUNT(*) FROM playlists WHERE user_id = $1", userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := s.db.Query(ctx,
		`SELECT p.id, p.user_id, p.name, p.description, p.cover_url, p.is_public, p.created_at, p.updated_at,
		        (SELECT COUNT(*) FROM playlist_songs ps WHERE ps.playlist_id = p.id) AS song_count
		 FROM playlists p
		 WHERE p.user_id = $1
		 ORDER BY p.updated_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var playlists []Playlist
	for rows.Next() {
		var p Playlist
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.CoverURL, &p.IsPublic,
			&p.CreatedAt, &p.UpdatedAt, &p.SongCount); err != nil {
			return nil, 0, err
		}
		playlists = append(playlists, p)
	}
	return playlists, total, rows.Err()
}

// GetByID retrieves a playlist by ID.
func (s *Service) GetByID(ctx context.Context, playlistID int64) (*Playlist, error) {
	var p Playlist
	err := s.db.QueryRow(ctx,
		`SELECT p.id, p.user_id, p.name, p.description, p.cover_url, p.is_public, p.created_at, p.updated_at,
		        u.username,
		        (SELECT COUNT(*) FROM playlist_songs ps WHERE ps.playlist_id = p.id) AS song_count
		 FROM playlists p
		 JOIN users u ON u.id = p.user_id
		 WHERE p.id = $1`,
		playlistID,
	).Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.CoverURL, &p.IsPublic,
		&p.CreatedAt, &p.UpdatedAt, &p.Creator, &p.SongCount)
	if err != nil {
		return nil, ErrNotFound
	}
	return &p, nil
}

// GetPublicByID retrieves a playlist by ID only if it is public.
func (s *Service) GetPublicByID(ctx context.Context, playlistID int64) (*Playlist, error) {
	p, err := s.GetByID(ctx, playlistID)
	if err != nil {
		return nil, err
	}
	if !p.IsPublic {
		return nil, ErrNotFound
	}
	return p, nil
}

// AddSong adds a song to a playlist.
func (s *Service) AddSong(ctx context.Context, userID, playlistID int64, songID int64, songName, artists, album, picURL string) (*Song, error) {
	var ownerID int64
	err := s.db.QueryRow(ctx, "SELECT user_id FROM playlists WHERE id = $1", playlistID).Scan(&ownerID)
	if err != nil {
		return nil, ErrNotFound
	}
	if ownerID != userID {
		return nil, ErrNotOwner
	}

	var maxPos int
	_ = s.db.QueryRow(ctx, "SELECT COALESCE(MAX(position), 0) FROM playlist_songs WHERE playlist_id = $1", playlistID).Scan(&maxPos)

	var song Song
	err = s.db.QueryRow(ctx,
		`INSERT INTO playlist_songs (playlist_id, song_id, song_name, artists, album, pic_url, position)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (playlist_id, song_id) DO UPDATE SET song_name = $3, artists = $4, album = $5, pic_url = $6
		 RETURNING id, playlist_id, song_id, song_name, artists, album, pic_url, position, created_at`,
		playlistID, songID, songName, artists, album, picURL, maxPos+1,
	).Scan(&song.ID, &song.PlaylistID, &song.SongID, &song.SongName, &song.Artists, &song.Album, &song.PicURL, &song.Position, &song.CreatedAt)
	if err != nil {
		return nil, err
	}

	_, _ = s.db.Exec(ctx, "UPDATE playlists SET updated_at = NOW() WHERE id = $1", playlistID)
	return &song, nil
}

// RemoveSong removes a song from a playlist.
func (s *Service) RemoveSong(ctx context.Context, userID, playlistID, songID int64) error {
	var ownerID int64
	err := s.db.QueryRow(ctx, "SELECT user_id FROM playlists WHERE id = $1", playlistID).Scan(&ownerID)
	if err != nil {
		return ErrNotFound
	}
	if ownerID != userID {
		return ErrNotOwner
	}

	result, err := s.db.Exec(ctx,
		"DELETE FROM playlist_songs WHERE playlist_id = $1 AND song_id = $2", playlistID, songID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrSongMissing
	}

	_, _ = s.db.Exec(ctx, "UPDATE playlists SET updated_at = NOW() WHERE id = $1", playlistID)
	return nil
}

// ListSongs returns all songs in a playlist ordered by position.
func (s *Service) ListSongs(ctx context.Context, playlistID int64, limit int) ([]Song, error) {
	if limit <= 0 {
		limit = 1000
	}
	rows, err := s.db.Query(ctx,
		`SELECT id, playlist_id, song_id, song_name, artists, album, pic_url, position, created_at
		 FROM playlist_songs
		 WHERE playlist_id = $1
		 ORDER BY position ASC
		 LIMIT $2`,
		playlistID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var songs []Song
	for rows.Next() {
		var song Song
		if err := rows.Scan(&song.ID, &song.PlaylistID, &song.SongID, &song.SongName,
			&song.Artists, &song.Album, &song.PicURL, &song.Position, &song.CreatedAt); err != nil {
			return nil, err
		}
		songs = append(songs, song)
	}
	return songs, rows.Err()
}
