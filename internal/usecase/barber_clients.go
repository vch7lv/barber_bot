package usecase

import (
	"context"

	"barber_bot/internal/domain"
	"barber_bot/internal/port"
)

// ClientWithVisitCount — клиент и количество его визитов.
type ClientWithVisitCount struct {
	Client *domain.Client
	Count  int
}

// ListClientsWithVisitCount возвращает всех клиентов с количеством визитов (B6).
func ListClientsWithVisitCount(ctx context.Context, clientRepo port.ClientRepository, visitRepo port.VisitRepository) ([]ClientWithVisitCount, error) {
	clients, err := clientRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ClientWithVisitCount, 0, len(clients))
	for _, c := range clients {
		n, err := visitRepo.CountByClient(ctx, c.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, ClientWithVisitCount{Client: c, Count: n})
	}
	return result, nil
}
