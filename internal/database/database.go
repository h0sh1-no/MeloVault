// Package database manages PostgreSQL connection pooling and schema migrations.
package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds database connection configuration.
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

// DSN returns the PostgreSQL connection string.
func (c Config) DSN() string {
	sslMode := c.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, sslMode)
}

// Pool wraps pgxpool.Pool for database connections.
type Pool struct {
	*pgxpool.Pool
}

// NewPool creates a new database connection pool.
func NewPool(cfg Config) (*Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}

	poolCfg.MaxConns = 25
	poolCfg.MinConns = 5
	poolCfg.MaxConnLifetime = time.Hour
	poolCfg.MaxConnIdleTime = 30 * time.Minute
	poolCfg.HealthCheckPeriod = time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	log.Printf("database connected: %s:%d/%s", cfg.Host, cfg.Port, cfg.Database)
	return &Pool{pool}, nil
}

// Migrate runs database migrations.
func (p *Pool) Migrate(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id BIGSERIAL PRIMARY KEY,
			username VARCHAR(50) UNIQUE NOT NULL,
			email VARCHAR(255) UNIQUE,
			password_hash VARCHAR(255),
			avatar VARCHAR(500),
			provider VARCHAR(20) NOT NULL DEFAULT 'email',
			provider_id VARCHAR(255),
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS favorites (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			song_id BIGINT NOT NULL,
			song_name VARCHAR(500),
			artists VARCHAR(500),
			album VARCHAR(500),
			pic_url VARCHAR(1000),
			created_at TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE(user_id, song_id)
		)`,
		`CREATE TABLE IF NOT EXISTS download_history (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			song_id BIGINT NOT NULL,
			song_name VARCHAR(500),
			artists VARCHAR(500),
			quality VARCHAR(20),
			file_type VARCHAR(10),
			file_size BIGINT,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS email_codes (
			id BIGSERIAL PRIMARY KEY,
			email VARCHAR(255) NOT NULL,
			code VARCHAR(6) NOT NULL,
			purpose VARCHAR(20) NOT NULL,
			expires_at TIMESTAMPTZ NOT NULL,
			used BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_favorites_user_id ON favorites(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_favorites_song_id ON favorites(song_id)`,
		`CREATE INDEX IF NOT EXISTS idx_download_history_user_id ON download_history(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_email_codes_email ON email_codes(email)`,
		// Role column migration (idempotent)
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(20) NOT NULL DEFAULT 'user'`,
		`CREATE INDEX IF NOT EXISTS idx_users_role ON users(role)`,
		`CREATE TABLE IF NOT EXISTS user_settings (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			settings JSONB NOT NULL DEFAULT '{}',
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE(user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS activity_logs (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
			action VARCHAR(30) NOT NULL,
			ip VARCHAR(45),
			province VARCHAR(50),
			city VARCHAR(50),
			user_agent VARCHAR(500),
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_activity_logs_user_id ON activity_logs(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_activity_logs_action ON activity_logs(action)`,
		`CREATE INDEX IF NOT EXISTS idx_activity_logs_created_at ON activity_logs(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_activity_logs_province ON activity_logs(province)`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login_ip VARCHAR(45)`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ`,
		`CREATE TABLE IF NOT EXISTS legal_documents (
			id BIGSERIAL PRIMARY KEY,
			type VARCHAR(30) NOT NULL,
			title VARCHAR(200) NOT NULL,
			content TEXT NOT NULL DEFAULT '',
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_legal_documents_type ON legal_documents(type)`,
		`CREATE INDEX IF NOT EXISTS idx_legal_documents_active ON legal_documents(is_active)`,
		// Playlists
		`CREATE TABLE IF NOT EXISTS playlists (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			name VARCHAR(200) NOT NULL,
			description TEXT DEFAULT '',
			cover_url VARCHAR(1000) DEFAULT '',
			is_public BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_playlists_user_id ON playlists(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_playlists_public ON playlists(is_public)`,
		`CREATE TABLE IF NOT EXISTS playlist_songs (
			id BIGSERIAL PRIMARY KEY,
			playlist_id BIGINT NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
			song_id BIGINT NOT NULL,
			song_name VARCHAR(500),
			artists VARCHAR(500),
			album VARCHAR(500),
			pic_url VARCHAR(1000),
			position INT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE(playlist_id, song_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_playlist_songs_playlist_id ON playlist_songs(playlist_id)`,
		`CREATE TABLE IF NOT EXISTS site_settings (
			id INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
			settings JSONB NOT NULL DEFAULT '{}',
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`INSERT INTO site_settings (id, settings) VALUES (1, '{"playlist_parse_enabled":true,"playlist_parse_admin_only":false,"album_parse_enabled":true,"album_parse_admin_only":false,"allow_register":true,"allow_email_register":true,"allow_linuxdo_register":true,"allow_email_login":true,"allow_linuxdo_login":true}') ON CONFLICT (id) DO NOTHING`,
		`CREATE TABLE IF NOT EXISTS netease_accounts (
			id BIGSERIAL PRIMARY KEY,
			nickname VARCHAR(200) DEFAULT '',
			cookie_string TEXT NOT NULL,
			music_u TEXT DEFAULT '',
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			last_used_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		// Backward-compatible migration for existing deployments.
		`ALTER TABLE IF EXISTS netease_accounts ALTER COLUMN music_u TYPE TEXT`,
		`CREATE INDEX IF NOT EXISTS idx_netease_accounts_active ON netease_accounts(is_active)`,
	}

	for i, q := range queries {
		if _, err := p.Exec(ctx, q); err != nil {
			return fmt.Errorf("migration %d: %w", i+1, err)
		}
	}

	log.Println("database migrations completed")
	return nil
}

// Close closes the database connection pool.
func (p *Pool) Close() {
	p.Pool.Close()
}
