package usecase

import (
	"context"
	"time"

	"barber_bot/internal/domain"
	"barber_bot/internal/port"
)

// ReminderItem — визит, которому нужно отправить напоминание (клиенту с TelegramID).
type ReminderItem struct {
	Visit       *domain.Visit
	TelegramID  int64
	StartsAtMSK time.Time
}

// VisitsToRemind возвращает список визитов, по которым нужно отправить напоминание:
// starts_at попадает в окно [now + reminderHours, now + reminderHours + windowMinutes] в локали loc.
func VisitsToRemind(
	ctx context.Context,
	reminderHours int,
	windowMinutes int,
	loc *time.Location,
	visitRepo port.VisitRepository,
	clientRepo port.ClientRepository,
) ([]ReminderItem, error) {
	now := time.Now().In(loc)
	windowStart := now.Add(time.Duration(reminderHours) * time.Hour)
	windowEnd := windowStart.Add(time.Duration(windowMinutes) * time.Minute)
	fromUnix := windowStart.Unix()
	toUnix := windowEnd.Unix()

	visits, err := visitRepo.ListScheduledInRange(ctx, fromUnix, toUnix)
	if err != nil {
		return nil, err
	}

	var result []ReminderItem
	for _, v := range visits {
		client, err := clientRepo.GetByID(ctx, v.ClientID)
		if err != nil || client == nil {
			continue
		}
		result = append(result, ReminderItem{
			Visit:       v,
			TelegramID:  client.TelegramID,
			StartsAtMSK: time.Unix(v.StartsAt, 0).In(loc),
		})
	}
	return result, nil
}

// BarberReminderItem — визит для напоминания барберу (за час до начала и т.п.).
type BarberReminderItem struct {
	Visit         *domain.Visit
	ClientName    string
	ServiceNames  []string
	StartsAtLocal time.Time
}

// VisitsToRemindBarber возвращает визиты, по которым нужно напомнить барберу:
// starts_at в окне [now + hoursBefore, now + hoursBefore + windowMinutes] в локали loc.
func VisitsToRemindBarber(
	ctx context.Context,
	hoursBefore int,
	windowMinutes int,
	loc *time.Location,
	visitRepo port.VisitRepository,
	clientRepo port.ClientRepository,
) ([]BarberReminderItem, error) {
	if hoursBefore <= 0 || windowMinutes <= 0 {
		return nil, nil
	}
	now := time.Now().In(loc)
	windowStart := now.Add(time.Duration(hoursBefore) * time.Hour)
	windowEnd := windowStart.Add(time.Duration(windowMinutes) * time.Minute)
	fromUnix := windowStart.Unix()
	toUnix := windowEnd.Unix()

	visits, err := visitRepo.ListScheduledInRange(ctx, fromUnix, toUnix)
	if err != nil {
		return nil, err
	}

	var result []BarberReminderItem
	for _, v := range visits {
		client, err := clientRepo.GetByID(ctx, v.ClientID)
		if err != nil || client == nil {
			continue
		}
		services, err := visitRepo.GetServicesByVisitID(ctx, v.ID)
		if err != nil {
			continue
		}
		names := make([]string, 0, len(services))
		for _, s := range services {
			names = append(names, s.Name)
		}
		name := client.Name
		if name == "" {
			name = "без имени"
		}
		result = append(result, BarberReminderItem{
			Visit:         v,
			ClientName:    name,
			ServiceNames:  names,
			StartsAtLocal: time.Unix(v.StartsAt, 0).In(loc),
		})
	}
	return result, nil
}
