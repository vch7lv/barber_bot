package usecase

import (
	"context"

	"barber_bot/internal/domain"
	"barber_bot/internal/port"
)

// PriceList возвращает список услуг прайс-листа, отсортированный по sort_order.
func PriceList(ctx context.Context, serviceRepo port.ServiceRepository) ([]*domain.Service, error) {
	return serviceRepo.List(ctx)
}
