package repository

import (
	"context"
	"database/sql"

	"barber_bot/internal/domain"
)

// BarberRepo реализует port.BarberRepository для SQLite.
type BarberRepo struct {
	db *sql.DB
}

// NewBarberRepo создаёт репозиторий барберов.
func NewBarberRepo(db *sql.DB) *BarberRepo {
	return &BarberRepo{db: db}
}

// GetByTelegramID возвращает барбера по Telegram ID или nil.
func (r *BarberRepo) GetByTelegramID(ctx context.Context, telegramID int64) (*domain.Barber, error) {
	var b domain.Barber
	err := r.db.QueryRowContext(ctx,
		`SELECT id, telegram_id FROM barbers WHERE telegram_id = ?`,
		telegramID,
	).Scan(&b.ID, &b.TelegramID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// EnsureBarbers создаёт записи барберов по списку Telegram ID, если их ещё нет.
func (r *BarberRepo) EnsureBarbers(ctx context.Context, telegramIDs []int64) error {
	for _, tid := range telegramIDs {
		_, err := r.db.ExecContext(ctx,
			`INSERT OR IGNORE INTO barbers (telegram_id) VALUES (?)`,
			tid,
		)
		if err != nil {
			return err
		}
	}
	return nil
}
