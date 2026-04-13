// Package settings manages per-user preference settings.
package settings

import (
	"context"
	"encoding/json"
	"time"

	"github.com/h0sh1-no/MeloVault/internal/database"
)

// Settings holds user playback preferences.
type Settings struct {
	StreamingQuality string  `json:"streaming_quality"`
	DownloadQuality  string  `json:"download_quality"`
	Volume           float64 `json:"volume"`
	RepeatMode       string  `json:"repeat_mode"`
}

// UserSettings associates settings with a user record.
type UserSettings struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Settings  Settings  `json:"settings"`
	UpdatedAt time.Time `json:"updated_at"`
}

var validQualities = map[string]bool{
	"standard": true, "exhigh": true, "lossless": true,
	"hires": true, "sky": true, "jyeffect": true, "jymaster": true,
}

var validRepeatModes = map[string]bool{
	"none": true, "one": true, "all": true,
}

// DefaultSettings returns the default playback preferences.
func DefaultSettings() Settings {
	return Settings{
		StreamingQuality: "jymaster",
		DownloadQuality:  "jymaster",
		Volume:           0.8,
		RepeatMode:       "none",
	}
}

// Service provides user settings persistence.
type Service struct {(db *database.Pool) *Service {
	return &Service{db: db}
}

func (s *Service) Get(ctx context.Context, userID int64) (*UserSettings, error) {
	var us UserSettings
	var raw []byte
	err := s.db.QueryRow(ctx,
		`SELECT id, user_id, settings, updated_at FROM user_settings WHERE user_id = $1`,
		userID,
	).Scan(&us.ID, &us.UserID, &raw, &us.UpdatedAt)

	if err != nil {
		us.Settings = DefaultSettings()
		us.UserID = userID
		us.UpdatedAt = time.Now()
		return &us, nil
	}

	if err := json.Unmarshal(raw, &us.Settings); err != nil {
		us.Settings = DefaultSettings()
	}
	return &us, nil
}

// Update merges the provided partial settings into existing ones.
func (s *Service) Update(ctx context.Context, userID int64, partial map[string]any) (*UserSettings, error) {
	current, _ := s.Get(ctx, userID)
	merged := current.Settings

	if v, ok := partial["streaming_quality"].(string); ok && validQualities[v] {
		merged.StreamingQuality = v
	}
	if v, ok := partial["download_quality"].(string); ok && validQualities[v] {
		merged.DownloadQuality = v
	}
	if v, ok := partial["volume"].(float64); ok && v >= 0 && v <= 1 {
		merged.Volume = v
	}
	if v, ok := partial["repeat_mode"].(string); ok && validRepeatModes[v] {
		merged.RepeatMode = v
	}

	raw, err := json.Marshal(merged)
	if err != nil {
		return nil, err
	}

	var us UserSettings
	var settingsBytes []byte
	err = s.db.QueryRow(ctx,
		`INSERT INTO user_settings (user_id, settings, updated_at)
		 VALUES ($1, $2, NOW())
		 ON CONFLICT (user_id) DO UPDATE SET settings = $2, updated_at = NOW()
		 RETURNING id, user_id, settings, updated_at`,
		userID, raw,
	).Scan(&us.ID, &us.UserID, &settingsBytes, &us.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(settingsBytes, &us.Settings); err != nil {
		us.Settings = merged
	}
	return &us, nil
}
