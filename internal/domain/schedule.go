package domain

// WorkingDay — рабочий день барбера: дата явно добавлена с окном времени. Шаг слотов всегда 1 час (в коде).
// WorkDate в формате YYYY-MM-DD, время — HH:MM (МСК).
type WorkingDay struct {
	BarberID  int64
	WorkDate  string
	StartTime string
	EndTime   string
}
