package domain

// Client — клиент бота (пользователь Telegram).
type Client struct {
	ID         int64
	TelegramID int64
	Name       string
	Username   string
	Contact    string
	CreatedAt  int64
}
