package port

import (
	"context"

	"barber_bot/internal/domain"
)

// ShopAddressRepository — хранилище адреса салона (один на бота).
type ShopAddressRepository interface {
	Get(ctx context.Context) (*domain.ShopAddress, error)
	Set(ctx context.Context, a *domain.ShopAddress) error
}
