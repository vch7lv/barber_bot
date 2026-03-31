package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"barber_bot/internal/usecase"
)

const reminderWindowMinutes = 15

type reminderRunner struct {
	bot            *Bot
	clientHours    int
	barberHours    int
	loc            *time.Location
	log            *slog.Logger
	lastSentClient map[int64]struct{}
	lastSentBarber map[int64]struct{}
}

func newReminderRunner(bot *Bot, clientHours, barberHours int, loc *time.Location, log *slog.Logger) *reminderRunner {
	return &reminderRunner{
		bot:            bot,
		clientHours:    clientHours,
		barberHours:    barberHours,
		loc:            loc,
		log:            log,
		lastSentClient: make(map[int64]struct{}),
		lastSentBarber: make(map[int64]struct{}),
	}
}

func (r *reminderRunner) Run(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.runOnce(ctx); err != nil {
				r.log.Error("reminder run", "err", err)
			}
		}
	}
}

func (r *reminderRunner) runOnce(ctx context.Context) error {
	if err := r.runClientReminders(ctx); err != nil {
		return err
	}
	return r.runBarberReminders(ctx)
}

func (r *reminderRunner) runClientReminders(ctx context.Context) error {
	items, err := usecase.VisitsToRemind(ctx, r.clientHours, reminderWindowMinutes, r.loc, r.bot.visitRepo, r.bot.clientRepo)
	if err != nil {
		return err
	}
	for _, item := range items {
		key := item.Visit.ID
		if _, ok := r.lastSentClient[key]; ok {
			continue
		}
		text := fmt.Sprintf("Напоминание: через %d ч у вас запись на %s (МСК).",
			r.clientHours, item.StartsAtMSK.Format("02.01.2006 15:04"))
		if err := r.bot.SendMessage(item.TelegramID, text); err != nil {
			r.log.Error("send reminder", "err", err, "visit_id", key)
			continue
		}
		r.lastSentClient[key] = struct{}{}
		r.log.Info("reminder sent", "visit_id", key, "telegram_id", item.TelegramID)
	}
	return nil
}

func (r *reminderRunner) runBarberReminders(ctx context.Context) error {
	if r.barberHours <= 0 {
		return nil
	}
	items, err := usecase.VisitsToRemindBarber(ctx, r.barberHours, reminderWindowMinutes, r.loc, r.bot.visitRepo, r.bot.clientRepo)
	if err != nil {
		return err
	}
	for _, item := range items {
		key := item.Visit.ID
		if _, ok := r.lastSentBarber[key]; ok {
			continue
		}
		svcText := strings.Join(item.ServiceNames, ", ")
		if svcText == "" {
			svcText = "—"
		}
		text := fmt.Sprintf(
			"Через %d ч запись: %s, %s (МСК).\nУслуги: %s",
			r.barberHours, item.ClientName, item.StartsAtLocal.Format("02.01.2006 15:04"), svcText,
		)
		allOK := true
		for _, tgID := range r.bot.cfg.BarberTelegramIDs {
			if err := r.bot.SendMessage(tgID, text); err != nil {
				r.log.Error("send barber reminder", "err", err, "visit_id", key, "barber_telegram_id", tgID)
				allOK = false
			}
		}
		if !allOK {
			continue
		}
		r.lastSentBarber[key] = struct{}{}
		r.log.Info("barber reminder sent", "visit_id", key)
	}
	return nil
}
