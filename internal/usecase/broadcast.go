package usecase

import (
	"context"

	"barber_bot/internal/port"
)

// BroadcastRecipients возвращает Telegram ID клиентов, которым можно слать рассылку (не забаненные) (B3).
func BroadcastRecipients(ctx context.Context, clientRepo port.ClientRepository, banRepo port.BanRepository) ([]int64, error) {
	banned, err := banRepo.ListBannedTelegramIDs(ctx)
	if err != nil {
		return nil, err
	}
	bm := make(map[int64]bool)
	for _, id := range banned {
		bm[id] = true
	}
	clients, err := clientRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	var ids []int64
	for _, c := range clients {
		if !bm[c.TelegramID] {
			ids = append(ids, c.TelegramID)
		}
	}
	return ids, nil
}
