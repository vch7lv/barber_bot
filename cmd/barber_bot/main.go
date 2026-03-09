package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"barber_bot/internal/adapter/repository"
	"barber_bot/internal/adapter/telegram"
	"barber_bot/internal/config"
	"barber_bot/internal/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		os.Stderr.WriteString("config: " + err.Error() + "\n")
		os.Exit(1)
	}

	logger := logger.New(cfg.LogLevel)
	logger.Info("starting barber_bot", "bot_mode", cfg.BotMode)

	ctx := context.Background()
	db, err := repository.DB(ctx, cfg.DatabaseDSN, logger)
	if err != nil {
		logger.Error("database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	barberRepo := repository.NewBarberRepo(db)
	if err := barberRepo.EnsureBarbers(ctx, cfg.BarberTelegramIDs); err != nil {
		logger.Error("ensure barbers", "err", err)
		os.Exit(1)
	}

	clientRepo := repository.NewClientRepo(db)
	serviceRepo := repository.NewServiceRepo(db)
	visitRepo := repository.NewVisitRepo(db)
	scheduleRepo := repository.NewScheduleRepo(db)
	banRepo := repository.NewBanRepo(db)
	auditRepo := repository.NewAuditRepo(db)
	addressRepo := repository.NewShopAddressRepo(db)

	bot, err := telegram.NewBot(ctx, cfg, logger, clientRepo, barberRepo, serviceRepo, visitRepo, scheduleRepo, banRepo, auditRepo, addressRepo)
	if err != nil {
		logger.Error("create bot", "err", err)
		os.Exit(1)
	}

	logger.Info("telegram bot started", "mode", cfg.BotMode)

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if cfg.BotMode == "polling" {
		bot.Run(ctx)
	} else {
		logger.Info("webhook mode not implemented yet, use BOT_MODE=polling")
		<-ctx.Done()
	}
	logger.Info("shutdown")
}
