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
// Несколько TG-аккаунтов могут относиться к одному барберу (таблица barber_telegram_ids).
func (r *BarberRepo) GetByTelegramID(ctx context.Context, telegramID int64) (*domain.Barber, error) {
	var barberID int64
	err := r.db.QueryRowContext(ctx,
		`SELECT barber_id FROM barber_telegram_ids WHERE telegram_id = ?`,
		telegramID,
	).Scan(&barberID)
	if err == nil {
		var b domain.Barber
		if err := r.db.QueryRowContext(ctx, `SELECT id, telegram_id FROM barbers WHERE id = ?`, barberID).Scan(&b.ID, &b.TelegramID); err != nil {
			if err == sql.ErrNoRows {
				return nil, nil
			}
			return nil, err
		}
		return &b, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}
	// Обратная совместимость: до миграции barber_telegram_ids барберы искались по barbers.telegram_id
	var b domain.Barber
	err = r.db.QueryRowContext(ctx,
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

// EnsureBarbers привязывает все переданные Telegram ID к одному барберу (один барбер — несколько аккаунтов).
func (r *BarberRepo) EnsureBarbers(ctx context.Context, telegramIDs []int64) error {
	if len(telegramIDs) == 0 {
		return nil
	}
	firstID := telegramIDs[0]
	_, err := r.db.ExecContext(ctx, `INSERT OR IGNORE INTO barbers (telegram_id) VALUES (?)`, firstID)
	if err != nil {
		return err
	}
	var canonicalID int64
	if err := r.db.QueryRowContext(ctx, `SELECT id FROM barbers WHERE telegram_id = ?`, firstID).Scan(&canonicalID); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}
	for _, tid := range telegramIDs {
		_, err := r.db.ExecContext(ctx,
			`INSERT OR IGNORE INTO barber_telegram_ids (barber_id, telegram_id) VALUES (?, ?)`,
			canonicalID, tid,
		)
		if err != nil {
			return err
		}
	}
	// Переносим расписание и визиты с других записей барберов на канонического
	_, _ = r.db.ExecContext(ctx, `UPDATE barber_working_days SET barber_id = ? WHERE barber_id != ?`, canonicalID, canonicalID)
	_, _ = r.db.ExecContext(ctx, `UPDATE visits SET barber_id = ? WHERE barber_id != ?`, canonicalID, canonicalID)
	// Удаляем лишние строки барберов
	_, _ = r.db.ExecContext(ctx, `DELETE FROM barbers WHERE id != ?`, canonicalID)
	return nil
}
