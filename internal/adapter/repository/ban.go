package repository

import (
	"context"
	"database/sql"

	"barber_bot/internal/domain"
)

// BanRepo реализует port.BanRepository для SQLite.
type BanRepo struct {
	db *sql.DB
}

// NewBanRepo создаёт репозиторий банов.
func NewBanRepo(db *sql.DB) *BanRepo {
	return &BanRepo{db: db}
}

// ListBannedTelegramIDs возвращает Telegram ID всех забаненных.
func (r *BanRepo) ListBannedTelegramIDs(ctx context.Context) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT client_telegram_id FROM bans`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// IsBanned возвращает true, если клиент с данным Telegram ID забанен.
func (r *BanRepo) IsBanned(ctx context.Context, clientTelegramID int64) (bool, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `SELECT id FROM bans WHERE client_telegram_id = ?`, clientTelegramID).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Ban добавляет бан.
func (r *BanRepo) Ban(ctx context.Context, b *domain.Ban) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO bans (client_telegram_id, banned_at, reason) VALUES (?, ?, ?)`,
		b.ClientTelegramID, b.BannedAt, b.Reason,
	)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	b.ID = id
	return nil
}

// Unban снимает бан по Telegram ID клиента.
func (r *BanRepo) Unban(ctx context.Context, clientTelegramID int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM bans WHERE client_telegram_id = ?`, clientTelegramID)
	return err
}
