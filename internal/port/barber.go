package port

import (
	"context"

	"barber_bot/internal/domain"
)

// BarberRepository — хранилище барберов (для связей с визитами и расписанием).
type BarberRepository interface {
	GetByTelegramID(ctx context.Context, telegramID int64) (*domain.Barber, error)
	EnsureBarbers(ctx context.Context, telegramIDs []int64) error
}
