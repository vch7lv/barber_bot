package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"barber_bot/internal/domain"
	"barber_bot/internal/port"
)

// FreeSlots возвращает свободные слоты на день date (в локали loc) для барбера barberID.
// Только явно добавленные рабочие дни дают слоты; шаг и занятость — 1 час.
func FreeSlots(
	ctx context.Context,
	barberID int64,
	date time.Time,
	_ int, // durationMin не используется: разрыв между записями всегда 1 час
	loc *time.Location,
	scheduleRepo port.ScheduleRepository,
	visitRepo port.VisitRepository,
	log *slog.Logger,
) ([]time.Time, error) {
	dateStr := date.Format("2006-01-02")
	wd, err := scheduleRepo.GetWorkingDay(ctx, barberID, dateStr)
	if err != nil {
		return nil, err
	}
	if wd == nil {
		return nil, nil
	}

	startMins, err := parseHHMM(wd.StartTime)
	if err != nil {
		return nil, fmt.Errorf("start_time %q: %w", wd.StartTime, err)
	}
	endMins, err := parseHHMM(wd.EndTime)
	if err != nil {
		return nil, fmt.Errorf("end_time %q: %w", wd.EndTime, err)
	}

	// Разрыв между записями всегда 1 час: шаг и занятость слота фиксированы.
	const slotStepMin = 60
	const slotDurationMin = 60

	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, loc)
	slotStart := dayStart.Add(time.Duration(startMins) * time.Minute)
	slotEnd := dayStart.Add(time.Duration(endMins) * time.Minute)
	step := time.Duration(slotStepMin) * time.Minute
	duration := time.Duration(slotDurationMin) * time.Minute

	var candidates []time.Time
	for t := slotStart; t.Add(duration).Before(slotEnd) || t.Add(duration).Equal(slotEnd); t = t.Add(step) {
		candidates = append(candidates, t)
	}

	fromUnix := dayStart.Unix()
	toUnix := dayStart.Add(24 * time.Hour).Unix()
	visits, err := visitRepo.VisitsByBarberInRange(ctx, barberID, fromUnix, toUnix)
	if err != nil {
		return nil, err
	}

	// Каждая запись занимает ровно 1 час при проверке пересечений.
	occupied := make([]struct{ Start, Duration int64 }, 0, len(visits))
	for _, v := range visits {
		occupied = append(occupied, struct{ Start, Duration int64 }{v.StartsAt, 60 * 60})
	}

	var free []time.Time
	for _, t := range candidates {
		unix := t.Unix()
		if domain.SlotFits(unix, slotDurationMin, occupied) {
			free = append(free, t)
		}
	}

	if log != nil && len(free) > 0 {
		log.Debug("free slots", "barber_id", barberID, "date", dateStr, "count", len(free))
	}
	return free, nil
}

func parseHHMM(s string) (int, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("expected HH:MM")
	}
	h, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	m, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, fmt.Errorf("invalid time")
	}
	return h*60 + m, nil
}
