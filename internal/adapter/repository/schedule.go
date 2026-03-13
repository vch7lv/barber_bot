package repository

import (
	"context"
	"database/sql"

	"barber_bot/internal/domain"
)

// ScheduleRepo реализует port.ScheduleRepository для SQLite.
type ScheduleRepo struct {
	db *sql.DB
}

// NewScheduleRepo создаёт репозиторий расписания.
func NewScheduleRepo(db *sql.DB) *ScheduleRepo {
	return &ScheduleRepo{db: db}
}

// GetWorkingDay возвращает рабочий день на дату. nil, nil если дата не добавлена.
func (r *ScheduleRepo) GetWorkingDay(ctx context.Context, barberID int64, dateStr string) (*domain.WorkingDay, error) {
	var w domain.WorkingDay
	err := r.db.QueryRowContext(ctx,
		`SELECT barber_id, work_date, start_time, end_time FROM barber_working_days WHERE barber_id = ? AND work_date = ?`,
		barberID, dateStr,
	).Scan(&w.BarberID, &w.WorkDate, &w.StartTime, &w.EndTime)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// ListWorkingDays возвращает список рабочих дней барбера (отсортированы по дате).
func (r *ScheduleRepo) ListWorkingDays(ctx context.Context, barberID int64) ([]*domain.WorkingDay, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT barber_id, work_date, start_time, end_time FROM barber_working_days WHERE barber_id = ? ORDER BY work_date`,
		barberID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*domain.WorkingDay
	for rows.Next() {
		w := &domain.WorkingDay{}
		if err := rows.Scan(&w.BarberID, &w.WorkDate, &w.StartTime, &w.EndTime); err != nil {
			return nil, err
		}
		list = append(list, w)
	}
	return list, rows.Err()
}

// SetWorkingDay сохраняет рабочий день (upsert).
func (r *ScheduleRepo) SetWorkingDay(ctx context.Context, w *domain.WorkingDay) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO barber_working_days (barber_id, work_date, start_time, end_time)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(barber_id, work_date) DO UPDATE SET start_time=excluded.start_time, end_time=excluded.end_time`,
		w.BarberID, w.WorkDate, w.StartTime, w.EndTime,
	)
	return err
}

// RemoveWorkingDay удаляет рабочий день на дату.
func (r *ScheduleRepo) RemoveWorkingDay(ctx context.Context, barberID int64, dateStr string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM barber_working_days WHERE barber_id = ? AND work_date = ?`,
		barberID, dateStr,
	)
	return err
}
