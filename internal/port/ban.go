package port

import (
	"context"

	"barber_bot/internal/domain"
)

// BanRepository — хранилище банов.
type BanRepository interface {
	IsBanned(ctx context.Context, clientTelegramID int64) (bool, error)
	ListBannedTelegramIDs(ctx context.Context) ([]int64, error)
	Ban(ctx context.Context, b *domain.Ban) error
	Unban(ctx context.Context, clientTelegramID int64) error
}
