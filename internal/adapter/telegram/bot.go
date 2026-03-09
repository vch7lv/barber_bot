package telegram

import (
	"context"
	"log/slog"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"barber_bot/internal/config"
	"barber_bot/internal/domain"
	"barber_bot/internal/port"
	"barber_bot/internal/usecase"
)

// Bot — Telegram-бот и обработчик обновлений.
type Bot struct {
	api             *tgbotapi.BotAPI
	cfg             *config.Config
	log             *slog.Logger
	firstBarberID   int64
	clientRepo      port.ClientRepository
	barberRepo      port.BarberRepository
	serviceRepo     port.ServiceRepository
	visitRepo       port.VisitRepository
	scheduleRepo    port.ScheduleRepository
	banRepo         port.BanRepository
	auditRepo       port.AuditLogRepository
	addressRepo     port.ShopAddressRepository
	state           *stateStore
	barberState     *barberStateStore
	barberClientMode barberClientModeStore
	reminderTicker  *reminderRunner
}

// NewBot создаёт бота и заполняет firstBarberID из БД.
func NewBot(
	ctx context.Context,
	cfg *config.Config,
	log *slog.Logger,
	clientRepo port.ClientRepository,
	barberRepo port.BarberRepository,
	serviceRepo port.ServiceRepository,
	visitRepo port.VisitRepository,
	scheduleRepo port.ScheduleRepository,
	banRepo port.BanRepository,
	auditRepo port.AuditLogRepository,
	addressRepo port.ShopAddressRepository,
) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(cfg.TelegramBotToken)
	if err != nil {
		return nil, err
	}
	api.Debug = cfg.LogLevel == "debug"

	var firstBarberID int64
	if len(cfg.BarberTelegramIDs) > 0 {
		barber, err := barberRepo.GetByTelegramID(ctx, cfg.BarberTelegramIDs[0])
		if err != nil {
			return nil, err
		}
		if barber != nil {
			firstBarberID = barber.ID
		}
	}

	bot := &Bot{
		api:           api,
		cfg:           cfg,
		log:           log,
		firstBarberID: firstBarberID,
		clientRepo:     clientRepo,
		barberRepo:     barberRepo,
		serviceRepo:    serviceRepo,
		visitRepo:      visitRepo,
		scheduleRepo:   scheduleRepo,
		banRepo:        banRepo,
		auditRepo:      auditRepo,
		addressRepo:    addressRepo,
		state:          newStateStore(),
		barberState:      newBarberStateStore(),
		barberClientMode: newBarberClientModeStore(),
	}
	bot.reminderTicker = newReminderRunner(bot, cfg.ReminderBeforeHours, cfg.TZ, log)
	return bot, nil
}

// Run запускает long polling и обработку обновлений. Блокируется до отмены ctx.
func (b *Bot) Run(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	upds := b.api.GetUpdatesChan(u)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		b.reminderTicker.Run(ctx)
	}()

	for {
		select {
		case <-ctx.Done():
			b.api.StopReceivingUpdates()
			wg.Wait()
			return
		case upd, ok := <-upds:
			if !ok {
				return
			}
			if err := b.handleUpdate(ctx, upd); err != nil {
				b.log.Error("handle update", "err", err, "update_id", upd.UpdateID)
			}
		}
	}
}

// SendMessage отправляет текстовое сообщение в чат.
func (b *Bot) SendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := b.api.Send(msg)
	return err
}

// handleUpdate маршрутизирует обновление в обработчик клиента или барбера.
func (b *Bot) handleUpdate(ctx context.Context, upd tgbotapi.Update) error {
	var chatID int64
	var from *tgbotapi.User
	var messageText string
	var callback *tgbotapi.CallbackQuery

	if upd.Message != nil {
		chatID = upd.Message.Chat.ID
		from = upd.Message.From
		messageText = upd.Message.Text
	} else if upd.CallbackQuery != nil {
		callback = upd.CallbackQuery
		chatID = callback.Message.Chat.ID
		from = callback.From
	} else {
		return nil
	}

	name := ""
	username := ""
	if from != nil {
		name = from.FirstName
		if from.LastName != "" {
			name += " " + from.LastName
		}
		username = from.UserName
	}

	ident, err := usecase.IdentifyUser(ctx, b.cfg, b.clientRepo, b.barberRepo, from.ID, name, username)
	if err != nil {
		b.log.Error("identify user", "err", err)
		return b.SendMessage(chatID, "Ошибка. Попробуйте позже.")
	}

	if ident.Role == "barber" {
		if messageText == "/client" || messageText == "Режим клиента" {
			b.barberClientMode.Set(chatID, true)
			return b.sendMainMenu(chatID)
		}
		if messageText == "/barber" || messageText == "Режим барбера" {
			b.barberClientMode.Set(chatID, false)
			return b.sendBarberMenu(chatID)
		}
		if b.barberClientMode.Get(chatID) {
			if callback != nil {
				return b.handleClientCallback(ctx, chatID, ident.Client, callback)
			}
			return b.handleClientMessage(ctx, chatID, ident.Client, messageText, upd.Message.MessageID)
		}
		if callback != nil {
			return b.handleBarberCallback(ctx, chatID, ident.BarberID, callback)
		}
		return b.handleBarberMessage(ctx, chatID, ident.BarberID, upd.Message)
	}

	banned, err := b.banRepo.IsBanned(ctx, ident.Client.TelegramID)
	if err != nil {
		return err
	}
	if banned {
		return b.SendMessage(chatID, "Вы заблокированы. Обратитесь к администратору.")
	}

	if callback != nil {
		return b.handleClientCallback(ctx, chatID, ident.Client, callback)
	}
	return b.handleClientMessage(ctx, chatID, ident.Client, messageText, upd.Message.MessageID)
}

// handleClientMessage обрабатывает текстовые сообщения и команды клиента.
func (b *Bot) handleClientMessage(ctx context.Context, chatID int64, client *domain.Client, text string, messageID int) error {
	switch text {
	case "/barber", "Режим барбера":
		if b.cfg.IsBarber(client.TelegramID) {
			b.barberClientMode.Set(chatID, false)
			return b.sendBarberMenu(chatID)
		}
		fallthrough
	case "/start", "Старт", "Назад":
		return b.sendMainMenu(chatID)
	case "Прайс", "/pricelist", "/price":
		return b.cmdPriceList(ctx, chatID)
	case "Мои записи", "/myvisits", "/visits":
		return b.cmdMyVisits(ctx, chatID, client)
	case "Записаться", "/book":
		return b.cmdBookStart(ctx, chatID, 0)
	case "Адрес", "/address":
		return b.cmdAddress(ctx, chatID)
	default:
		return b.sendMainMenu(chatID)
	}
}

// mainMenuReplyKeyboard возвращает reply-клавиатуру главного меню клиента.
func mainMenuReplyKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Прайс"),
			tgbotapi.NewKeyboardButton("Мои записи"),
			tgbotapi.NewKeyboardButton("Записаться"),
		),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Адрес")),
	)
}

func (b *Bot) sendMainMenu(chatID int64) error {
	text := "Главное меню:\n\n• Прайс — услуги и цены\n• Мои записи — предстоящие визиты\n• Записаться — новая запись\n• Адрес — где мы находимся"
	if b.barberClientMode.Get(chatID) {
		text += "\n\nБарберам: /barber — вернуться в панель"
	}
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = mainMenuReplyKeyboard()
	_, err := b.api.Send(msg)
	return err
}
