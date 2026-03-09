package repository

import (
	"context"
	"database/sql"
	"time"
)

// AuditRepo реализует port.AuditLogRepository для SQLite.
type AuditRepo struct {
	db *sql.DB
}

// NewAuditRepo создаёт репозиторий аудит-лога.
func NewAuditRepo(db *sql.DB) *AuditRepo {
	return &AuditRepo{db: db}
}

// Append добавляет запись в аудит-лог.
func (r *AuditRepo) Append(ctx context.Context, eventType, payload string) error {
	at := time.Now().Unix()
	_, err := r.db.ExecContext(ctx, `INSERT INTO audit_log (at, event_type, payload) VALUES (?, ?, ?)`, at, eventType, payload)
	return err
}
