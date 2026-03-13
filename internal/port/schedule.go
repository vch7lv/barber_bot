package port

import (
	"context"

	"barber_bot/internal/domain"
)

// ScheduleRepository — хранилище рабочих дней барбера (явный список дат с окном времени на каждую).
type ScheduleRepository interface {
	GetWorkingDay(ctx context.Context, barberID int64, dateStr string) (*domain.WorkingDay, error)
	ListWorkingDays(ctx context.Context, barberID int64) ([]*domain.WorkingDay, error)
	SetWorkingDay(ctx context.Context, w *domain.WorkingDay) error
	RemoveWorkingDay(ctx context.Context, barberID int64, dateStr string) error
}
