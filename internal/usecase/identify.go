package usecase

import (
	"context"
	"time"

	"barber_bot/internal/config"
	"barber_bot/internal/domain"
	"barber_bot/internal/port"
)

// IdentifyResult — результат идентификации пользователя.
type IdentifyResult struct {
	Role     string        // "client" или "barber"
	Client   *domain.Client
	BarberID int64 // 0 если не барбер
}

// IdentifyUser определяет пользователя по Telegram ID: создаёт/обновляет клиента, возвращает роль и данные.
func IdentifyUser(
	ctx context.Context,
	cfg *config.Config,
	clientRepo port.ClientRepository,
	barberRepo port.BarberRepository,
	telegramID int64,
	name, username string,
) (*IdentifyResult, error) {
	now := time.Now().Unix()

	client, err := clientRepo.GetByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, err
	}
	if client == nil {
		client = &domain.Client{
			TelegramID: telegramID,
			Name:       name,
			Username:   username,
			CreatedAt:  now,
		}
	} else {
		client.Name = name
		client.Username = username
	}
	if err := clientRepo.Save(ctx, client); err != nil {
		return nil, err
	}

	if cfg.IsBarber(telegramID) {
		barber, err := barberRepo.GetByTelegramID(ctx, telegramID)
		if err != nil {
			return nil, err
		}
		barberID := int64(0)
		if barber != nil {
			barberID = barber.ID
		}
		return &IdentifyResult{Role: "barber", Client: client, BarberID: barberID}, nil
	}

	return &IdentifyResult{Role: "client", Client: client, BarberID: 0}, nil
}
