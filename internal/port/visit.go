package port

import (
	"context"

	"barber_bot/internal/domain"
)

// VisitRepository — хранилище визитов.
type VisitRepository interface {
	GetByID(ctx context.Context, id int64) (*domain.Visit, error)
	GetServicesByVisitID(ctx context.Context, visitID int64) ([]*domain.Service, error)
	ListByClient(ctx context.Context, clientID int64, fromUnix, toUnix int64) ([]*domain.Visit, error)
	ListByBarber(ctx context.Context, barberID int64, fromUnix, toUnix int64) ([]*domain.Visit, error)
	// VisitsByBarberInRange возвращает визиты барбера в диапазоне [from, to] для проверки занятости слотов.
	VisitsByBarberInRange(ctx context.Context, barberID int64, fromUnix, toUnix int64) ([]*domain.Visit, error)
	// ListByClientTelegramID возвращает запланированные визиты клиента по telegram_id (JOIN с clients). Надёжно для «Мои записи».
	ListByClientTelegramID(ctx context.Context, clientTelegramID int64, fromUnix, toUnix int64) ([]*domain.Visit, error)
	// ListScheduledInRange возвращает запланированные визиты в диапазоне [from, to] (для напоминаний).
	ListScheduledInRange(ctx context.Context, fromUnix, toUnix int64) ([]*domain.Visit, error)
	CountByClient(ctx context.Context, clientID int64) (int, error)
	Save(ctx context.Context, v *domain.Visit, serviceIDs []int64) error
	UpdateStatus(ctx context.Context, id int64, status string) error
}
