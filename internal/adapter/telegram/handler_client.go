package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"barber_bot/internal/domain"
	"barber_bot/internal/usecase"
)

func (b *Bot) cmdPriceList(ctx context.Context, chatID int64) error {
	services, err := usecase.PriceList(ctx, b.serviceRepo)
	if err != nil {
		b.log.Error("pricelist", "err", err)
		_ = b.SendMessage(chatID, "Ошибка загрузки прайса.")
		return b.sendMainMenu(chatID)
	}
	if len(services) == 0 {
		_ = b.SendMessage(chatID, "Прайс-лист пока пуст.")
		return b.sendMainMenu(chatID)
	}
	var lines []string
	for _, s := range services {
		lines = append(lines, fmt.Sprintf("• %s — %d ₽, %d мин", s.Name, s.PriceCents/100, s.DurationMin))
	}
	msg := tgbotapi.NewMessage(chatID, "Прайс-лист:\n\n"+strings.Join(lines, "\n")+"\n\nИспользуйте кнопки ниже для других действий.")
	msg.ReplyMarkup = mainMenuReplyKeyboard()
	_, err = b.api.Send(msg)
	return err
}

func (b *Bot) cmdAddress(ctx context.Context, chatID int64) error {
	addr, err := b.addressRepo.Get(ctx)
	if err != nil {
		b.log.Error("address get", "err", err)
		_ = b.SendMessage(chatID, "Ошибка загрузки адреса.")
		return b.sendMainMenu(chatID)
	}
	displayText := addr.AddressText
	if displayText == "" && addr.AddressPhotoFileID == "" {
		_ = b.SendMessage(chatID, "📍 Адрес пока не указан.")
		return b.sendMainMenu(chatID)
	}
	if displayText == "" {
		displayText = "Адрес салона"
	}
	markup := mainMenuReplyKeyboard()
	if addr.AddressPhotoFileID != "" {
		photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileID(addr.AddressPhotoFileID))
		photo.Caption = "📍 " + displayText
		photo.ReplyMarkup = markup
		_, err = b.api.Send(photo)
		return err
	}
	msg := tgbotapi.NewMessage(chatID, "📍 "+displayText)
	msg.ReplyMarkup = markup
	_, err = b.api.Send(msg)
	return err
}

func (b *Bot) cmdMyVisits(ctx context.Context, chatID int64, client *domain.Client) error {
	now := time.Now()
	// Только визиты с временем начала не раньше текущего момента (в unix); прошедшие не запрашиваем из БД.
	from := now.Unix()
	to := now.Add(90 * 24 * time.Hour).Unix()

	list, err := usecase.MyVisits(ctx, client.TelegramID, from, to, b.visitRepo)
	if err != nil {
		b.log.Error("myvisits", "err", err)
		_ = b.SendMessage(chatID, "Ошибка загрузки записей.")
		return b.sendMainMenu(chatID)
	}
	if len(list) == 0 {
		msg := tgbotapi.NewMessage(chatID, "У вас нет предстоящих записей.")
		msg.ReplyMarkup = mainMenuReplyKeyboard()
		_, _ = b.api.Send(msg)
		return nil
	}

	loc := b.cfg.TZ
	var lines []string
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, vw := range list {
		t := time.Unix(vw.Visit.StartsAt, 0).In(loc)
		svcNames := make([]string, 0, len(vw.Services))
		for _, s := range vw.Services {
			svcNames = append(svcNames, s.Name)
		}
		lines = append(lines, fmt.Sprintf("• %s %s — %s", t.Format("02.01.2006"), t.Format("15:04"), strings.Join(svcNames, ", ")))
		btnLabel := fmt.Sprintf("Отменить запись %s на %s", t.Format("02.01"), t.Format("15:04"))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(btnLabel, "cancel:"+strconv.FormatInt(vw.Visit.ID, 10))))
	}
	msg := tgbotapi.NewMessage(chatID, "Ваши записи:\n\n"+strings.Join(lines, "\n\n")+"\n\nИспользуйте кнопки меню ниже для других действий.")
	if len(rows) > 0 {
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	}
	_, err = b.api.Send(msg)
	return err
}

// handleClientCallback обрабатывает нажатия inline-кнопок.
func (b *Bot) handleClientCallback(ctx context.Context, chatID int64, client *domain.Client, cb *tgbotapi.CallbackQuery) error {
	data := cb.Data
	_, _ = b.api.Request(tgbotapi.NewCallback(cb.ID, ""))

	if strings.HasPrefix(data, "srv:") {
		idStr := strings.TrimPrefix(data, "srv:")
		id, _ := strconv.ParseInt(idStr, 10, 64)
		b.state.ToggleService(chatID, id)
		st := b.state.Get(chatID)
		var ids []int64
		if st != nil {
			ids = st.ServiceIDs
		}
		return b.sendBookingServices(ctx, chatID, "Выберите услуги:", cb.Message.MessageID, ids)
	}
	if data == "book:done" {
		st := b.state.Get(chatID)
		if st == nil || len(st.ServiceIDs) == 0 {
			return b.editOrSend(chatID, cb.Message.MessageID, "Сначала выберите хотя бы одну услугу.")
		}
		return b.sendBookingDates(ctx, chatID, cb.Message.MessageID, st.ServiceIDs)
	}
	if strings.HasPrefix(data, "d:") {
		dateStr := strings.TrimPrefix(data, "d:")
		st := b.state.Get(chatID)
		if st == nil {
			return nil
		}
		return b.sendBookingSlots(ctx, chatID, cb.Message.MessageID, st.ServiceIDs, dateStr)
	}
	if strings.HasPrefix(data, "t:") {
		unixStr := strings.TrimPrefix(data, "t:")
		startUnix, _ := strconv.ParseInt(unixStr, 10, 64)
		st := b.state.Get(chatID)
		if st == nil {
			return nil
		}
		return b.sendBookingConfirm(ctx, chatID, cb.Message.MessageID, client, st.ServiceIDs, startUnix)
	}
	if strings.HasPrefix(data, "book:confirm:") {
		unixStr := strings.TrimPrefix(data, "book:confirm:")
		startUnix, _ := strconv.ParseInt(unixStr, 10, 64)
		st := b.state.Get(chatID)
		if st == nil {
			return b.sendMainMenu(chatID)
		}
		return b.doBookVisit(ctx, chatID, cb.Message.MessageID, client, st.ServiceIDs, startUnix)
	}
	if strings.HasPrefix(data, "book:back_slots:") {
		dateStr := strings.TrimPrefix(data, "book:back_slots:")
		st := b.state.Get(chatID)
		if st == nil {
			return b.sendMainMenu(chatID)
		}
		return b.sendBookingSlots(ctx, chatID, cb.Message.MessageID, st.ServiceIDs, dateStr)
	}
	if data == "book:back_dates" {
		st := b.state.Get(chatID)
		if st == nil {
			return b.sendMainMenu(chatID)
		}
		return b.sendBookingDates(ctx, chatID, cb.Message.MessageID, st.ServiceIDs)
	}
	if data == "book:back_services" {
		st := b.state.Get(chatID)
		if st == nil {
			return b.sendMainMenu(chatID)
		}
		return b.sendBookingServices(ctx, chatID, "Выберите услуги:", cb.Message.MessageID, st.ServiceIDs)
	}
	if data == "book:back_menu" {
		b.state.Set(chatID, nil)
		return b.sendMainMenu(chatID)
	}
	if strings.HasPrefix(data, "cancel:") {
		visitIDStr := strings.TrimPrefix(data, "cancel:")
		visitID, _ := strconv.ParseInt(visitIDStr, 10, 64)
		return b.doCancelVisit(ctx, chatID, client.ID, visitID)
	}

	return b.sendMainMenu(chatID)
}

func (b *Bot) cmdBookStart(ctx context.Context, chatID int64, messageID int) error {
	return b.sendBookingServices(ctx, chatID, "Выберите услуги:", messageID, nil)
}

func (b *Bot) sendBookingServices(ctx context.Context, chatID int64, caption string, messageID int, selectedIDs []int64) error {
	services, err := usecase.PriceList(ctx, b.serviceRepo)
	if err != nil {
		_ = b.SendMessage(chatID, "Ошибка загрузки услуг.")
		return b.sendMainMenu(chatID)
	}
	if len(services) == 0 {
		_ = b.SendMessage(chatID, "Прайс пуст. Запись пока недоступна.")
		return b.sendMainMenu(chatID)
	}

	selectedSet := make(map[int64]bool)
	var selectedNames []string
	for _, id := range selectedIDs {
		selectedSet[id] = true
		for _, s := range services {
			if s.ID == id {
				selectedNames = append(selectedNames, s.Name)
				break
			}
		}
	}
	if len(selectedNames) > 0 {
		caption += "\n\n✓ Выбранные: " + strings.Join(selectedNames, ", ")
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, s := range services {
		label := fmt.Sprintf("%s — %d ₽", s.Name, s.PriceCents/100)
		if selectedSet[s.ID] {
			label = "✓ " + label
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, "srv:"+strconv.FormatInt(s.ID, 10)),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Далее", "book:done")))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("← Назад в меню", "book:back_menu")))

	if messageID != 0 {
		edit := tgbotapi.NewEditMessageText(chatID, messageID, caption)
		edit.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
		_, err := b.api.Send(edit)
		return b.ignoreMessageNotModified(err)
	}
	msg := tgbotapi.NewMessage(chatID, caption)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	_, err = b.api.Send(msg)
	return err
}

func (b *Bot) sendBookingDates(ctx context.Context, chatID int64, messageID int, serviceIDs []int64) error {
	loc := b.cfg.TZ
	now := time.Now().In(loc)
	var rows [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < 14; i++ {
		d := now.AddDate(0, 0, i)
		dateStr := d.Format("2006-01-02")
		wd, err := b.scheduleRepo.GetWorkingDay(ctx, b.firstBarberID, dateStr)
		if err != nil {
			b.log.Error("sendBookingDates GetWorkingDay", "err", err, "date", dateStr)
			continue
		}
		if wd == nil {
			continue
		}
		label := d.Format("02.01")
		if i == 0 {
			label = "Сегодня " + d.Format("02.01")
		} else if i == 1 {
			label = "Завтра " + d.Format("02.01")
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, "d:"+dateStr),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("← Назад — изменить услуги", "book:back_services")))
	text := "Выберите дату:"
	if len(rows) == 0 {
		_ = b.editOrSend(chatID, messageID, "В ближайшие 14 дней приёма нет. Обратитесь в салон.")
		return b.sendMainMenu(chatID)
	}
	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
	_, err := b.api.Send(edit)
	return b.ignoreMessageNotModified(err)
}

func (b *Bot) sendBookingSlots(ctx context.Context, chatID int64, messageID int, serviceIDs []int64, dateStr string) error {
	date, err := time.ParseInLocation("2006-01-02", dateStr, b.cfg.TZ)
	if err != nil {
		_ = b.editOrSend(chatID, messageID, "Неверная дата.")
		return b.sendMainMenu(chatID)
	}

	services, err := b.serviceRepo.List(ctx)
	if err != nil {
		_ = b.editOrSend(chatID, messageID, "Ошибка загрузки.")
		return b.sendMainMenu(chatID)
	}
	var durServices []*domain.Service
	for _, s := range services {
		for _, id := range serviceIDs {
			if s.ID == id {
				durServices = append(durServices, s)
				break
			}
		}
	}
	if len(durServices) == 0 {
		_ = b.editOrSend(chatID, messageID, "Ошибка: услуги не найдены.")
		return b.sendMainMenu(chatID)
	}
	// Разрыв между записями всегда 1 час; длительность услуг не влияет на сетку слотов.
	const slotDurationMin = 60
	slots, err := usecase.FreeSlots(ctx, b.firstBarberID, date, slotDurationMin, b.cfg.TZ, b.scheduleRepo, b.visitRepo, b.log)
	if err != nil {
		b.log.Error("free slots", "err", err)
		_ = b.editOrSend(chatID, messageID, "Запись не удалась. Не удалось загрузить слоты. Попробуйте позже.")
		return b.sendMainMenu(chatID)
	}
	if len(slots) == 0 {
		_ = b.editOrSend(chatID, messageID, "Запись не удалась. На эту дату нет приёма или свободных слотов. Возможно расписание не настроено — обратитесь в салон.")
		return b.sendMainMenu(chatID)
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, t := range slots {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(t.Format("15:04"), "t:"+strconv.FormatInt(t.Unix(), 10)),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("← Назад — другая дата", "book:back_dates")))
	text := "Выберите время (МСК):"
	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
	_, err = b.api.Send(edit)
	return b.ignoreMessageNotModified(err)
}

// sendBookingConfirm показывает сводку записи и кнопки Подтвердить / Назад (другое время).
func (b *Bot) sendBookingConfirm(ctx context.Context, chatID int64, messageID int, client *domain.Client, serviceIDs []int64, startUnix int64) error {
	loc := b.cfg.TZ
	t := time.Unix(startUnix, 0).In(loc)
	dateStr := t.Format("2006-01-02")
	text := fmt.Sprintf("Проверьте запись:\n\n📅 %s в %s (МСК)", t.Format("02.01.2006"), t.Format("15:04"))
	var totalCents int64
	var parts []string
	for _, sid := range serviceIDs {
		s, err := b.serviceRepo.GetByID(ctx, sid)
		if err != nil || s == nil {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s — %d ₽", s.Name, s.PriceCents/100))
		totalCents += int64(s.PriceCents)
	}
	if len(parts) > 0 {
		text += "\n\n" + strings.Join(parts, ", ")
		text += fmt.Sprintf("\nИтого: %d ₽", totalCents/100)
	}
	text += "\n\nПодтвердить запись?"
	rows := [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Подтвердить запись", "book:confirm:"+strconv.FormatInt(startUnix, 10)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("← Назад — другое время", "book:back_slots:"+dateStr),
		),
	}
	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
	_, err := b.api.Send(edit)
	return b.ignoreMessageNotModified(err)
}

func (b *Bot) doBookVisit(ctx context.Context, chatID int64, messageID int, client *domain.Client, serviceIDs []int64, startUnix int64) error {
	// Всегда берём client_id по telegram_id прямо перед записью, чтобы визит точно попал в «Мои записи» (JOIN по telegram_id).
	c, err := b.clientRepo.GetByTelegramID(ctx, client.TelegramID)
	if err != nil || c == nil {
		b.log.Error("doBookVisit get client", "err", err, "telegram_id", client.TelegramID)
		b.state.Set(chatID, nil)
		_ = b.editOrSend(chatID, messageID, "Ошибка. Попробуйте снова.")
		return b.sendMainMenu(chatID)
	}
	v, err := usecase.BookVisit(ctx, c.ID, c.TelegramID, b.firstBarberID, startUnix, serviceIDs,
		b.banRepo, b.visitRepo, b.serviceRepo, b.auditRepo)
	if err != nil {
		b.state.Set(chatID, nil)
		if err == usecase.ErrBanned {
			_ = b.editOrSend(chatID, messageID, "Вы заблокированы.")
		} else if err == usecase.ErrSlotInPast {
			_ = b.editOrSend(chatID, messageID, "Это время уже прошло. Выберите другой слот.")
		} else {
			b.log.Error("book visit", "err", err)
			_ = b.editOrSend(chatID, messageID, "Запись не удалась. Возможно слот уже занят — выберите другое время.")
		}
		return b.sendMainMenu(chatID)
	}
	b.state.Set(chatID, nil)

	b.notifyBarbersNewVisit(ctx, v.StartsAt, c.Name, serviceIDs)

	loc := b.cfg.TZ
	t := time.Unix(v.StartsAt, 0).In(loc)
	text := fmt.Sprintf("Вы записаны на %s в %s (МСК).", t.Format("02.01.2006"), t.Format("15:04"))
	var totalCents int64
	var parts []string
	for _, sid := range serviceIDs {
		s, err := b.serviceRepo.GetByID(ctx, sid)
		if err != nil || s == nil {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s — %d ₽", s.Name, s.PriceCents/100))
		totalCents += int64(s.PriceCents)
	}
	if len(parts) > 0 {
		text += "\n\n" + strings.Join(parts, ", ")
		text += fmt.Sprintf("\nИтого: %d ₽", totalCents/100)
	}
	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ReplyMarkup = nil
	_, _ = b.api.Send(edit) // ошибка "message is not modified" не критична
	return b.sendMainMenu(chatID)
}

// notifyBarbersNewVisit шлёт уведомление на все TG-аккаунты барбера из конфига.
func (b *Bot) notifyBarbersNewVisit(ctx context.Context, startsAt int64, clientName string, serviceIDs []int64) {
	loc := b.cfg.TZ
	t := time.Unix(startsAt, 0).In(loc)
	name := strings.TrimSpace(clientName)
	if name == "" {
		name = "без имени"
	}
	var lines []string
	for _, sid := range serviceIDs {
		s, err := b.serviceRepo.GetByID(ctx, sid)
		if err != nil || s == nil {
			continue
		}
		lines = append(lines, fmt.Sprintf("• %s — %d ₽", s.Name, s.PriceCents/100))
	}
	servicesBlock := strings.Join(lines, "\n")
	if servicesBlock == "" {
		servicesBlock = "• (услуги не указаны)"
	}
	text := fmt.Sprintf(
		"Новая запись\n\n📅 %s в %s (МСК)\n\nКлиент: %s\n\nУслуги:\n%s",
		t.Format("02.01.2006"), t.Format("15:04"), name, servicesBlock,
	)
	for _, tgID := range b.cfg.BarberTelegramIDs {
		if err := b.SendMessage(tgID, text); err != nil {
			b.log.Error("notify barber new visit", "err", err, "barber_telegram_id", tgID)
		}
	}
}

func (b *Bot) doCancelVisit(ctx context.Context, chatID int64, clientID int64, visitID int64) error {
	err := usecase.CancelVisit(ctx, visitID, clientID, b.visitRepo, b.auditRepo)
	if err != nil {
		if err == usecase.ErrNotYourVisit || err == usecase.ErrVisitNotFound {
			_ = b.SendMessage(chatID, "Запись не найдена или уже отменена.")
		} else if err == usecase.ErrVisitPast {
			_ = b.SendMessage(chatID, "Нельзя отменить прошедшую запись.")
		} else {
			_ = b.SendMessage(chatID, "Ошибка отмены.")
		}
		return b.sendMainMenu(chatID)
	}
	_ = b.SendMessage(chatID, "Запись отменена.")
	return b.sendMainMenu(chatID)
}

func (b *Bot) editOrSend(chatID int64, messageID int, text string) error {
	if messageID != 0 {
		edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
		_, err := b.api.Send(edit)
		return b.ignoreMessageNotModified(err)
	}
	return b.SendMessage(chatID, text)
}

// ignoreMessageNotModified возвращает nil для ошибки "message is not modified", иначе err (чтобы не ломать поток и не логировать ERROR).
func (b *Bot) ignoreMessageNotModified(err error) error {
	if err != nil && strings.Contains(err.Error(), "message is not modified") {
		return nil
	}
	return err
}
