package domain

// Barber — барбер (админ). Telegram ID задаётся в конфиге; в БД хранится для связей с визитами и расписанием.
type Barber struct {
	ID         int64
	TelegramID int64
}
