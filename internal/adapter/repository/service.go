package repository

import (
	"context"
	"database/sql"

	"barber_bot/internal/domain"
)

// ServiceRepo реализует port.ServiceRepository для SQLite.
type ServiceRepo struct {
	db *sql.DB
}

// NewServiceRepo создаёт репозиторий услуг.
func NewServiceRepo(db *sql.DB) *ServiceRepo {
	return &ServiceRepo{db: db}
}

// GetByID возвращает услугу по ID или nil.
func (r *ServiceRepo) GetByID(ctx context.Context, id int64) (*domain.Service, error) {
	var s domain.Service
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, price_cents, duration_min, sort_order, created_at FROM services WHERE id = ?`,
		id,
	).Scan(&s.ID, &s.Name, &s.PriceCents, &s.DurationMin, &s.SortOrder, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// List возвращает все услуги, отсортированные по sort_order.
func (r *ServiceRepo) List(ctx context.Context) ([]*domain.Service, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, price_cents, duration_min, sort_order, created_at FROM services ORDER BY sort_order, id`)
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

// Save создаёт или обновляет услугу.
func (r *ServiceRepo) Save(ctx context.Context, s *domain.Service) error {
	if s.ID == 0 {
		res, err := r.db.ExecContext(ctx,
			`INSERT INTO services (name, price_cents, duration_min, sort_order, created_at) VALUES (?, ?, ?, ?, ?)`,
			s.Name, s.PriceCents, s.DurationMin, s.SortOrder, s.CreatedAt,
		)
		if err != nil {
			return err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return err
		}
		s.ID = id
		return nil
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE services SET name = ?, price_cents = ?, duration_min = ?, sort_order = ? WHERE id = ?`,
		s.Name, s.PriceCents, s.DurationMin, s.SortOrder, s.ID,
	)
	return err
}

// Delete удаляет услугу по ID.
func (r *ServiceRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM services WHERE id = ?`, id)
	return err
}
