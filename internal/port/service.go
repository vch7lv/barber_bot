package port

import (
	"context"

	"barber_bot/internal/domain"
)

// ServiceRepository — хранилище услуг прайс-листа.
type ServiceRepository interface {
	GetByID(ctx context.Context, id int64) (*domain.Service, error)
	List(ctx context.Context) ([]*domain.Service, error)
	Save(ctx context.Context, s *domain.Service) error
	Delete(ctx context.Context, id int64) error
}
