package usecase

import (
	"context"
	"time"

	"barber_bot/internal/domain"
	"barber_bot/internal/port"
)

// VisitWithServices — визит с списком услуг.
type VisitWithServices struct {
	Visit   *domain.Visit
	Services []*domain.Service
}

// MyVisits возвращает предстоящие визиты клиента (status=scheduled) от from до to.
// Поиск по telegram_id через JOIN с clients, чтобы «Мои записи» всегда находили визиты этого пользователя (в т.ч. барбер в режиме клиента).
func MyVisits(ctx context.Context, clientTelegramID int64, fromUnix, toUnix int64, visitRepo port.VisitRepository) ([]VisitWithServices, error) {
	visits, err := visitRepo.ListByClientTelegramID(ctx, clientTelegramID, fromUnix, toUnix)
	if err != nil {
		return nil, err
	}
	nowUnix := time.Now().Unix()
	result := make([]VisitWithServices, 0, len(visits))
	for _, v := range visits {
		endUnix := v.StartsAt + int64(v.DurationMin)*60
		if endUnix <= nowUnix {
			continue
		}
		svc, err := visitRepo.GetServicesByVisitID(ctx, v.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, VisitWithServices{Visit: v, Services: svc})
	}
	return result, nil
}
