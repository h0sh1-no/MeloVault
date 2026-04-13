// Package legal manages legal documents such as terms of service and privacy policy.
package legal

import (
	"context"
	"errors"
	"time"

	"github.com/h0sh1-no/MeloVault/internal/database"
)

var (
	ErrNotFound = errors.New("document not found")
)

// Document represents a legal document (e.g. terms of service, privacy policy).
type Document struct {
	ID        int64     `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	IsActive  bool      `json:"is_active"`
	CreatedBy *int64    `json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Service provides legal document management.
type Service struct {(db *database.Pool) *Service {
	return &Service{db: db}
}

// GetActiveDocument returns the most recently updated active document of a given type.
func (s *Service) GetActiveDocument(ctx context.Context, docType string) (*Document, error) {
	row := s.db.QueryRow(ctx,
		`SELECT id, type, title, content, is_active, created_by, created_at, updated_at
		 FROM legal_documents
		 WHERE type = $1 AND is_active = TRUE
		 ORDER BY updated_at DESC
		 LIMIT 1`, docType)

	var doc Document
	err := row.Scan(&doc.ID, &doc.Type, &doc.Title, &doc.Content,
		&doc.IsActive, &doc.CreatedBy, &doc.CreatedAt, &doc.UpdatedAt)
	if err != nil {
		return nil, ErrNotFound
	}
	return &doc, nil
}

// SaveDocument creates or updates a legal document.
// It deactivates all existing documents of the same type and inserts a new active one.
func (s *Service) SaveDocument(ctx context.Context, docType, title, content string, userID int64) (*Document, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`UPDATE legal_documents SET is_active = FALSE, updated_at = NOW() WHERE type = $1 AND is_active = TRUE`,
		docType)
	if err != nil {
		return nil, err
	}

	var doc Document
	err = tx.QueryRow(ctx,
		`INSERT INTO legal_documents (type, title, content, is_active, created_by)
		 VALUES ($1, $2, $3, TRUE, $4)
		 RETURNING id, type, title, content, is_active, created_by, created_at, updated_at`,
		docType, title, content, userID).
		Scan(&doc.ID, &doc.Type, &doc.Title, &doc.Content,
			&doc.IsActive, &doc.CreatedBy, &doc.CreatedAt, &doc.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &doc, nil
}

// ListDocuments returns all documents of a given type, ordered by newest first.
func (s *Service) ListDocuments(ctx context.Context, docType string) ([]Document, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, type, title, content, is_active, created_by, created_at, updated_at
		 FROM legal_documents
		 WHERE type = $1
		 ORDER BY created_at DESC`, docType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []Document
	for rows.Next() {
		var doc Document
		if err := rows.Scan(&doc.ID, &doc.Type, &doc.Title, &doc.Content,
			&doc.IsActive, &doc.CreatedBy, &doc.CreatedAt, &doc.UpdatedAt); err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, nil
}
