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

// GetDefaultSchedule возвращает расписание по умолчанию для барбера. nil, nil если не задано.
func (r *ScheduleRepo) GetDefaultSchedule(ctx context.Context, barberID int64) (*domain.DefaultSchedule, error) {
	var s domain.DefaultSchedule
	err := r.db.QueryRowContext(ctx,
		`SELECT barber_id, start_time, end_time, slot_step_min FROM barber_default_schedule WHERE barber_id = ?`,
		barberID,
	).Scan(&s.BarberID, &s.StartTime, &s.EndTime, &s.SlotStepMin)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// SetDefaultSchedule сохраняет расписание по умолчанию (upsert).
func (r *ScheduleRepo) SetDefaultSchedule(ctx context.Context, s *domain.DefaultSchedule) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO barber_default_schedule (barber_id, start_time, end_time, slot_step_min)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(barber_id) DO UPDATE SET start_time=excluded.start_time, end_time=excluded.end_time, slot_step_min=excluded.slot_step_min`,
		s.BarberID, s.StartTime, s.EndTime, s.SlotStepMin,
	)
	return err
}

// IsDayOff возвращает true, если дата (YYYY-MM-DD) — выходной для барбера.
func (r *ScheduleRepo) IsDayOff(ctx context.Context, barberID int64, dateStr string) (bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT 1 FROM barber_days_off WHERE barber_id = ? AND off_date = ? LIMIT 1`,
		barberID, dateStr,
	).Scan(&n)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// AddDayOff добавляет выходной. offDate в формате YYYY-MM-DD.
func (r *ScheduleRepo) AddDayOff(ctx context.Context, barberID int64, offDate string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO barber_days_off (barber_id, off_date) VALUES (?, ?)`,
		barberID, offDate,
	)
	return err
}

// RemoveDayOff удаляет выходной на дату.
func (r *ScheduleRepo) RemoveDayOff(ctx context.Context, barberID int64, offDate string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM barber_days_off WHERE barber_id = ? AND off_date = ?`,
		barberID, offDate,
	)
	return err
}

// ListDaysOff возвращает список выходных барбера (отсортированы по дате).
func (r *ScheduleRepo) ListDaysOff(ctx context.Context, barberID int64) ([]*domain.DayOff, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, barber_id, off_date FROM barber_days_off WHERE barber_id = ? ORDER BY off_date`,
		barberID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*domain.DayOff
	for rows.Next() {
		d := &domain.DayOff{}
		if err := rows.Scan(&d.ID, &d.BarberID, &d.OffDate); err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}

// GetScheduleOverride возвращает особое расписание на дату. nil, nil если не задано.
func (r *ScheduleRepo) GetScheduleOverride(ctx context.Context, barberID int64, dateStr string) (*domain.ScheduleOverride, error) {
	var o domain.ScheduleOverride
	err := r.db.QueryRowContext(ctx,
		`SELECT barber_id, work_date, start_time, end_time, slot_step_min FROM barber_schedule_override WHERE barber_id = ? AND work_date = ?`,
		barberID, dateStr,
	).Scan(&o.BarberID, &o.WorkDate, &o.StartTime, &o.EndTime, &o.SlotStepMin)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// SetScheduleOverride сохраняет особый день (upsert).
func (r *ScheduleRepo) SetScheduleOverride(ctx context.Context, o *domain.ScheduleOverride) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO barber_schedule_override (barber_id, work_date, start_time, end_time, slot_step_min)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(barber_id, work_date) DO UPDATE SET start_time=excluded.start_time, end_time=excluded.end_time, slot_step_min=excluded.slot_step_min`,
		o.BarberID, o.WorkDate, o.StartTime, o.EndTime, o.SlotStepMin,
	)
	return err
}

// ListScheduleOverrides возвращает список особых дней барбера (отсортированы по дате).
func (r *ScheduleRepo) ListScheduleOverrides(ctx context.Context, barberID int64) ([]*domain.ScheduleOverride, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT barber_id, work_date, start_time, end_time, slot_step_min FROM barber_schedule_override WHERE barber_id = ? ORDER BY work_date`,
		barberID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*domain.ScheduleOverride
	for rows.Next() {
		o := &domain.ScheduleOverride{}
		if err := rows.Scan(&o.BarberID, &o.WorkDate, &o.StartTime, &o.EndTime, &o.SlotStepMin); err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, rows.Err()
}

// RemoveScheduleOverride удаляет особый день на дату.
func (r *ScheduleRepo) RemoveScheduleOverride(ctx context.Context, barberID int64, dateStr string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM barber_schedule_override WHERE barber_id = ? AND work_date = ?`,
		barberID, dateStr,
	)
	return err
}
