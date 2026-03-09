package domain

// DefaultSchedule — рабочее время по умолчанию для барбера (применяется ко всем дням, кроме выходных).
// Время в формате HH:MM (МСК).
const (
	DefaultScheduleStart   = "11:00"
	DefaultScheduleEnd     = "22:00"
	DefaultScheduleStepMin = 30
)

type DefaultSchedule struct {
	BarberID    int64
	StartTime   string // HH:MM
	EndTime     string // HH:MM
	SlotStepMin int
}

// DayOff — дата выходного (приём не ведётся). OffDate в формате YYYY-MM-DD.
type DayOff struct {
	ID       int64
	BarberID int64
	OffDate  string
}

// ScheduleOverride — особый день: на дату WorkDate своё рабочее время (вместо расписания по умолчанию).
// WorkDate в формате YYYY-MM-DD, время — HH:MM (МСК).
type ScheduleOverride struct {
	BarberID    int64
	WorkDate    string
	StartTime   string
	EndTime     string
	SlotStepMin int
}
