package port

import (
	"context"

	"barber_bot/internal/domain"
)

// ScheduleRepository — хранилище расписания барбера (по умолчанию + выходные по датам + особые дни).
type ScheduleRepository interface {
	GetDefaultSchedule(ctx context.Context, barberID int64) (*domain.DefaultSchedule, error)
	SetDefaultSchedule(ctx context.Context, s *domain.DefaultSchedule) error
	IsDayOff(ctx context.Context, barberID int64, dateStr string) (bool, error)
	AddDayOff(ctx context.Context, barberID int64, offDate string) error
	RemoveDayOff(ctx context.Context, barberID int64, offDate string) error
	ListDaysOff(ctx context.Context, barberID int64) ([]*domain.DayOff, error)

	GetScheduleOverride(ctx context.Context, barberID int64, dateStr string) (*domain.ScheduleOverride, error)
	SetScheduleOverride(ctx context.Context, o *domain.ScheduleOverride) error
	ListScheduleOverrides(ctx context.Context, barberID int64) ([]*domain.ScheduleOverride, error)
	RemoveScheduleOverride(ctx context.Context, barberID int64, dateStr string) error
}
