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
	result := make([]VisitWithServices, 0, len(visits))
	for _, v := range visits {
		svc, err := visitRepo.GetServicesByVisitID(ctx, v.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, VisitWithServices{Visit: v, Services: svc})
	}
	return result, nil
}

// UpcomingStartUnix возвращает unix начала «предстоящих» визитов.
// Небольшой сдвиг назад (2 ч) чтобы не терять записи из-за расхождения времени сервера и часового пояса клиента.
func UpcomingStartUnix(now time.Time) int64 {
	return now.Unix() - 2*3600
}
