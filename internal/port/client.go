package port

import (
	"context"

	"barber_bot/internal/domain"
)

// ClientRepository — хранилище клиентов.
type ClientRepository interface {
	GetByID(ctx context.Context, id int64) (*domain.Client, error)
	GetByTelegramID(ctx context.Context, telegramID int64) (*domain.Client, error)
	ListAll(ctx context.Context) ([]*domain.Client, error)
	Save(ctx context.Context, c *domain.Client) error
}
