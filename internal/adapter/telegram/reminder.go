package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"barber_bot/internal/usecase"
)

const reminderWindowMinutes = 15

type reminderRunner struct {
	bot       *Bot
	hours     int
	loc       *time.Location
	log       *slog.Logger
	lastSent  map[int64]struct{}
}

func newReminderRunner(bot *Bot, hours int, loc *time.Location, log *slog.Logger) *reminderRunner {
	return &reminderRunner{bot: bot, hours: hours, loc: loc, log: log, lastSent: make(map[int64]struct{})}
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
	items, err := usecase.VisitsToRemind(ctx, r.hours, reminderWindowMinutes, r.loc, r.bot.visitRepo, r.bot.clientRepo)
	if err != nil {
		return err
	}
	for _, item := range items {
		key := item.Visit.ID
		if _, ok := r.lastSent[key]; ok {
			continue
		}
		text := fmt.Sprintf("Напоминание: через %d ч у вас запись на %s (МСК).",
			r.hours, item.StartsAtMSK.Format("02.01.2006 15:04"))
		if err := r.bot.SendMessage(item.TelegramID, text); err != nil {
			r.log.Error("send reminder", "err", err, "visit_id", key)
			continue
		}
		r.lastSent[key] = struct{}{}
		r.log.Info("reminder sent", "visit_id", key, "telegram_id", item.TelegramID)
	}
	return nil
}
