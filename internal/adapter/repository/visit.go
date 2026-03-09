package repository

import (
	"context"
	"database/sql"

	"barber_bot/internal/domain"
)

// VisitRepo реализует port.VisitRepository для SQLite.
type VisitRepo struct {
	db *sql.DB
}

// NewVisitRepo создаёт репозиторий визитов.
func NewVisitRepo(db *sql.DB) *VisitRepo {
	return &VisitRepo{db: db}
}

// GetByID возвращает визит по ID или nil.
func (r *VisitRepo) GetByID(ctx context.Context, id int64) (*domain.Visit, error) {
	var v domain.Visit
	err := r.db.QueryRowContext(ctx,
		`SELECT id, client_id, barber_id, starts_at, duration_min, created_at, status FROM visits WHERE id = ?`,
		id,
	).Scan(&v.ID, &v.ClientID, &v.BarberID, &v.StartsAt, &v.DurationMin, &v.CreatedAt, &v.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// GetServicesByVisitID возвращает услуги, входящие в визит.
func (r *VisitRepo) GetServicesByVisitID(ctx context.Context, visitID int64) ([]*domain.Service, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT s.id, s.name, s.price_cents, s.duration_min, s.sort_order, s.created_at
		 FROM visit_services vs JOIN services s ON vs.service_id = s.id WHERE vs.visit_id = ? ORDER BY s.sort_order`,
		visitID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*domain.Service
	for rows.Next() {
		s := &domain.Service{}
		if err := rows.Scan(&s.ID, &s.Name, &s.PriceCents, &s.DurationMin, &s.SortOrder, &s.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// ListByClient возвращает визиты клиента в диапазоне [fromUnix, toUnix].
func (r *VisitRepo) ListByClient(ctx context.Context, clientID int64, fromUnix, toUnix int64) ([]*domain.Visit, error) {
	return r.listVisits(ctx, `SELECT id, client_id, barber_id, starts_at, duration_min, created_at, status FROM visits WHERE client_id = ? AND starts_at >= ? AND starts_at <= ? AND status = 'scheduled' ORDER BY starts_at`,
		clientID, fromUnix, toUnix)
}

// ListByClientTelegramID возвращает запланированные визиты клиента по telegram_id (JOIN с clients).
// Используется для «Мои записи», чтобы не зависеть от возможного расхождения client_id.
func (r *VisitRepo) ListByClientTelegramID(ctx context.Context, clientTelegramID int64, fromUnix, toUnix int64) ([]*domain.Visit, error) {
	return r.listVisits(ctx,
		`SELECT v.id, v.client_id, v.barber_id, v.starts_at, v.duration_min, v.created_at, v.status
		 FROM visits v INNER JOIN clients c ON v.client_id = c.id
		 WHERE c.telegram_id = ? AND v.starts_at >= ? AND v.starts_at <= ? AND v.status = 'scheduled' ORDER BY v.starts_at`,
		clientTelegramID, fromUnix, toUnix)
}

// ListByBarber возвращает визиты барбера в диапазоне.
func (r *VisitRepo) ListByBarber(ctx context.Context, barberID int64, fromUnix, toUnix int64) ([]*domain.Visit, error) {
	return r.listVisits(ctx, `SELECT id, client_id, barber_id, starts_at, duration_min, created_at, status FROM visits WHERE barber_id = ? AND starts_at >= ? AND starts_at <= ? ORDER BY starts_at`,
		barberID, fromUnix, toUnix)
}

// VisitsByBarberInRange возвращает все запланированные визиты барбера в диапазоне (для проверки слотов).
func (r *VisitRepo) VisitsByBarberInRange(ctx context.Context, barberID int64, fromUnix, toUnix int64) ([]*domain.Visit, error) {
	return r.listVisits(ctx, `SELECT id, client_id, barber_id, starts_at, duration_min, created_at, status FROM visits WHERE barber_id = ? AND starts_at >= ? AND starts_at <= ? AND status = 'scheduled' ORDER BY starts_at`,
		barberID, fromUnix, toUnix)
}

// ListScheduledInRange возвращает запланированные визиты в диапазоне [from, to].
func (r *VisitRepo) ListScheduledInRange(ctx context.Context, fromUnix, toUnix int64) ([]*domain.Visit, error) {
	return r.listVisits(ctx, `SELECT id, client_id, barber_id, starts_at, duration_min, created_at, status FROM visits WHERE status = 'scheduled' AND starts_at >= ? AND starts_at <= ? ORDER BY starts_at`,
		fromUnix, toUnix)
}

// CountByClient возвращает количество визитов клиента (все статусы).
func (r *VisitRepo) CountByClient(ctx context.Context, clientID int64) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM visits WHERE client_id = ?`, clientID).Scan(&n)
	return n, err
}

func (r *VisitRepo) listVisits(ctx context.Context, query string, args ...any) ([]*domain.Visit, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*domain.Visit
	for rows.Next() {
		v := &domain.Visit{}
		if err := rows.Scan(&v.ID, &v.ClientID, &v.BarberID, &v.StartsAt, &v.DurationMin, &v.CreatedAt, &v.Status); err != nil {
			return nil, err
		}
		list = append(list, v)
	}
	return list, rows.Err()
}

// Save создаёт визит и связи visit_services.
func (r *VisitRepo) Save(ctx context.Context, v *domain.Visit, serviceIDs []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.ExecContext(ctx,
		`INSERT INTO visits (client_id, barber_id, starts_at, duration_min, created_at, status) VALUES (?, ?, ?, ?, ?, ?)`,
		v.ClientID, v.BarberID, v.StartsAt, v.DurationMin, v.CreatedAt, v.Status,
	)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	v.ID = id
	for _, sid := range serviceIDs {
		_, err = tx.ExecContext(ctx, `INSERT INTO visit_services (visit_id, service_id) VALUES (?, ?)`, id, sid)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// UpdateStatus обновляет статус визита.
func (r *VisitRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE visits SET status = ? WHERE id = ?`, status, id)
	return err
}
