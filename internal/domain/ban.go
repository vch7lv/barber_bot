package domain

// Ban — бан клиента по Telegram ID.
type Ban struct {
	ID               int64
	ClientTelegramID int64
	BannedAt         int64
	Reason           string
}
