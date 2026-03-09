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

// FreeSlots возвращает свободные слоты на день date (в локали loc) для барбера barberID при длительности durationMin.
// Расписание по датам: используется время по умолчанию (или 11:00–22:00, шаг 30 мин), выходные по датам — слотов нет.
func FreeSlots(
	ctx context.Context,
	barberID int64,
	date time.Time,
	durationMin int,
	loc *time.Location,
	scheduleRepo port.ScheduleRepository,
	visitRepo port.VisitRepository,
	log *slog.Logger,
) ([]time.Time, error) {
	dateStr := date.Format("2006-01-02")
	off, err := scheduleRepo.IsDayOff(ctx, barberID, dateStr)
	if err != nil {
		return nil, err
	}
	if off {
		return nil, nil
	}

	override, err := scheduleRepo.GetScheduleOverride(ctx, barberID, dateStr)
	if err != nil {
		return nil, err
	}
	var startStr, endStr string
	var stepMin int
	if override != nil {
		startStr, endStr = override.StartTime, override.EndTime
		stepMin = override.SlotStepMin
	} else {
		def, err := scheduleRepo.GetDefaultSchedule(ctx, barberID)
		if err != nil {
			return nil, err
		}
		startStr = domain.DefaultScheduleStart
		endStr = domain.DefaultScheduleEnd
		stepMin = domain.DefaultScheduleStepMin
		if def != nil {
			startStr, endStr = def.StartTime, def.EndTime
			stepMin = def.SlotStepMin
		}
	}

	startMins, err := parseHHMM(startStr)
	if err != nil {
		return nil, fmt.Errorf("start_time %q: %w", startStr, err)
	}
	endMins, err := parseHHMM(endStr)
	if err != nil {
		return nil, fmt.Errorf("end_time %q: %w", endStr, err)
	}

	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, loc)
	slotStart := dayStart.Add(time.Duration(startMins) * time.Minute)
	slotEnd := dayStart.Add(time.Duration(endMins) * time.Minute)
	step := time.Duration(stepMin) * time.Minute
	duration := time.Duration(durationMin) * time.Minute

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

	// Длительности в секундах (starts_at — unix), иначе SlotsOverlap даёт неверный результат.
	occupied := make([]struct{ Start, Duration int64 }, 0, len(visits))
	for _, v := range visits {
		occupied = append(occupied, struct{ Start, Duration int64 }{v.StartsAt, int64(v.DurationMin) * 60})
	}

	var free []time.Time
	for _, t := range candidates {
		unix := t.Unix()
		if domain.SlotFits(unix, durationMin, occupied) {
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
