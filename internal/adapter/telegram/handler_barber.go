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

// visitStatusLabel возвращает русскую подпись статуса визита для отображения барберу; для scheduled возвращает пустую строку.
func visitStatusLabel(status string) string {
	switch status {
	case "scheduled":
		return ""
	case "cancelled":
		return "(отменена)"
	case "completed":
		return "(выполнена)"
	default:
		return ""
	}
}

// barberMenuReplyKeyboard возвращает reply-клавиатуру панели барбера.
func barberMenuReplyKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Прайс"),
			tgbotapi.NewKeyboardButton("График"),
			tgbotapi.NewKeyboardButton("Адрес"),
			tgbotapi.NewKeyboardButton("Записи"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Клиенты"),
			tgbotapi.NewKeyboardButton("Бан"),
			tgbotapi.NewKeyboardButton("Разбан"),
		),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Рассылка")),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Статистика"),
			tgbotapi.NewKeyboardButton("Режим клиента"),
		),
	)
}

func (b *Bot) sendBarberMenu(chatID int64) error {
	text := "Панель барбера:\n\n• Прайс — услуги\n• График — расписание\n• Адрес — текст и фото для клиентов\n• Записи — просмотр/отмена\n• Клиенты — список\n• Бан / Разбан\n• Рассылка — сообщение всем\n• Статистика — выручка и визиты\n\nДля теста: «Режим клиента». Назад — в любой ввод «Назад» или «Меню»."
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = barberMenuReplyKeyboard()
	_, err := b.api.Send(msg)
	return err
}

func (b *Bot) handleBarberMessage(ctx context.Context, chatID int64, barberID int64, msg *tgbotapi.Message) error {
	if barberID == 0 {
		_ = b.SendMessage(chatID, "Ошибка: барбер не найден в БД.")
		return b.sendBarberMenu(chatID)
	}

	text := ""
	if msg != nil {
		text = msg.Text
	}

	st := b.barberState.Get(chatID)
	if st != nil {
		if st.Step == "address_edit_photo" && msg != nil && len(msg.Photo) > 0 {
			fileID := msg.Photo[len(msg.Photo)-1].FileID
			b.barberState.Clear(chatID)
			addr, err := b.addressRepo.Get(ctx)
			if err != nil {
				_ = b.SendMessage(chatID, "Ошибка загрузки адреса.")
				return b.sendBarberMenu(chatID)
			}
			addr.AddressPhotoFileID = fileID
			if err := b.addressRepo.Set(ctx, addr); err != nil {
				_ = b.SendMessage(chatID, "Ошибка сохранения фото.")
				return b.sendBarberMenu(chatID)
			}
			return b.barberAddress(ctx, chatID, barberID)
		}
		return b.handleBarberStateStep(ctx, chatID, barberID, text, st)
	}

	switch text {
	case "/start", "Назад", "Меню":
		return b.sendBarberMenu(chatID)
	case "Прайс":
		return b.barberPriceList(ctx, chatID, barberID)
	case "График":
		return b.barberSchedule(ctx, chatID, barberID)
	case "Адрес":
		return b.barberAddress(ctx, chatID, barberID)
	case "Записи":
		return b.barberVisitsPeriod(ctx, chatID, barberID)
	case "Клиенты":
		return b.barberClients(ctx, chatID)
	case "Бан":
		b.barberState.Set(chatID, "ban_tgid", "")
		return b.SendMessage(chatID, "Введите Telegram ID клиента для бана (число). Назад — отмена.")
	case "Разбан":
		b.barberState.Set(chatID, "unban_tgid", "")
		return b.SendMessage(chatID, "Введите Telegram ID клиента для разблокировки. Назад — отмена.")
	case "Рассылка":
		b.barberState.Set(chatID, "broadcast", "")
		return b.SendMessage(chatID, "Отправьте текст сообщения для рассылки всем клиентам. Назад — отмена.")
	case "Статистика":
		return b.barberStatsPeriod(ctx, chatID, barberID)
	default:
		return b.sendBarberMenu(chatID)
	}
}

func (b *Bot) handleBarberStateStep(ctx context.Context, chatID int64, barberID int64, text string, st *barberState) error {
	if text == "Назад" || text == "Меню" {
		b.barberState.Clear(chatID)
		return b.sendBarberMenu(chatID)
	}
	switch st.Step {
	case "broadcast":
		b.barberState.Clear(chatID)
		return b.barberDoBroadcast(ctx, chatID, text)
	case "ban_tgid":
		b.barberState.Clear(chatID)
		tgID, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
		if err != nil {
			b.barberState.Clear(chatID)
			_ = b.SendMessage(chatID, "Неверный формат. Введите число (Telegram ID). Назад — отмена.")
			return b.sendBarberMenu(chatID)
		}
		return b.barberBan(ctx, chatID, tgID)
	case "service_name":
		b.barberState.Set(chatID, "service_price", text)
		return b.SendMessage(chatID, "Введите цену услуги в рублях (число):")
	case "service_price":
		price, err := strconv.Atoi(strings.TrimSpace(text))
		if err != nil || price < 0 {
			b.barberState.Clear(chatID)
			_ = b.SendMessage(chatID, "Неверный формат. Начните заново из Прайс → Добавить услугу.")
			return b.sendBarberMenu(chatID)
		}
		b.barberState.Set(chatID, "service_duration", st.Data+"|"+strconv.Itoa(price*100))
		return b.SendMessage(chatID, "Введите длительность услуги в минутах (число):")
	case "service_duration":
		b.barberState.Clear(chatID)
		dur, err := strconv.Atoi(strings.TrimSpace(text))
		if err != nil || dur <= 0 {
			b.barberState.Clear(chatID)
			_ = b.SendMessage(chatID, "Неверный формат. Начните заново из Прайс → Добавить услугу.")
			return b.sendBarberMenu(chatID)
		}
		parts := strings.SplitN(st.Data, "|", 2)
		name := parts[0]
		priceCents := 0
		if len(parts) == 2 {
			priceCents, _ = strconv.Atoi(parts[1])
		}
		return b.barberAddService(ctx, chatID, name, priceCents, dur)
	case "schedule_work_add":
		b.barberState.Clear(chatID)
		dateStr := strings.TrimSpace(text)
		if _, err := time.Parse("2006-01-02", dateStr); err != nil {
			_ = b.SendMessage(chatID, "Неверный формат даты. Введите ГГГГ-ММ-ДД. Начните заново из График.")
			return b.sendBarberMenu(chatID)
		}
		b.barberState.Set(chatID, "schedule_work_time", dateStr)
		return b.SendMessage(chatID, "Введите время начала и окончания (МСК). Например: 11:00 22:00. Между слотами 1 час. Назад — отмена.")
	case "schedule_work_time":
		b.barberState.Clear(chatID)
		parts := strings.SplitN(strings.TrimSpace(text), " ", 2)
		if len(parts) < 2 {
			_ = b.SendMessage(chatID, "Неверный формат. Пример: 11:00 22:00. Начните заново из График.")
			return b.sendBarberMenu(chatID)
		}
		dateStr := st.Data
		w := &domain.WorkingDay{BarberID: barberID, WorkDate: dateStr, StartTime: strings.TrimSpace(parts[0]), EndTime: strings.TrimSpace(parts[1])}
		if err := b.scheduleRepo.SetWorkingDay(ctx, w); err != nil {
			_ = b.SendMessage(chatID, "Ошибка сохранения.")
			return b.sendBarberMenu(chatID)
		}
		return b.barberSchedule(ctx, chatID, barberID)
	case "schedule_work_edit":
		b.barberState.Clear(chatID)
		parts := strings.SplitN(strings.TrimSpace(text), " ", 2)
		if len(parts) < 2 {
			_ = b.SendMessage(chatID, "Неверный формат. Пример: 11:00 22:00. Начните заново из График.")
			return b.sendBarberMenu(chatID)
		}
		dateStr := st.Data
		w := &domain.WorkingDay{BarberID: barberID, WorkDate: dateStr, StartTime: strings.TrimSpace(parts[0]), EndTime: strings.TrimSpace(parts[1])}
		if err := b.scheduleRepo.SetWorkingDay(ctx, w); err != nil {
			_ = b.SendMessage(chatID, "Ошибка сохранения.")
			return b.sendBarberMenu(chatID)
		}
		return b.barberSchedule(ctx, chatID, barberID)
	case "address_edit_text":
		b.barberState.Clear(chatID)
		addr, err := b.addressRepo.Get(ctx)
		if err != nil {
			_ = b.SendMessage(chatID, "Ошибка загрузки адреса.")
			return b.sendBarberMenu(chatID)
		}
		addr.AddressText = strings.TrimSpace(text)
		if err := b.addressRepo.Set(ctx, addr); err != nil {
			_ = b.SendMessage(chatID, "Ошибка сохранения адреса.")
			return b.sendBarberMenu(chatID)
		}
		return b.barberAddress(ctx, chatID, barberID)
	case "address_edit_photo":
		_ = b.SendMessage(chatID, "Ожидается фото. Отправьте изображение или «Назад».")
		return nil
	case "cancel_visit_id":
		b.barberState.Clear(chatID)
		visitID, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
		if err != nil || visitID <= 0 {
			_ = b.SendMessage(chatID, "Неверный формат. Введите номер записи (число). Назад — отмена.")
			return b.sendBarberMenu(chatID)
		}
		return b.barberCancelVisit(ctx, chatID, barberID, visitID)
	case "unban_tgid":
		b.barberState.Clear(chatID)
		tgID, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
		if err != nil {
			b.barberState.Clear(chatID)
			_ = b.SendMessage(chatID, "Неверный формат. Введите число (Telegram ID).")
			return b.sendBarberMenu(chatID)
		}
		return b.barberUnban(ctx, chatID, tgID)
	case "edit_svc_name":
		b.barberState.Set(chatID, "edit_svc_price", st.Data+"|"+text)
		return b.SendMessage(chatID, "Введите новую цену в рублях (число). Назад — отмена.")
	case "edit_svc_price":
		price, err := strconv.Atoi(strings.TrimSpace(text))
		if err != nil || price < 0 {
			b.barberState.Clear(chatID)
			_ = b.SendMessage(chatID, "Неверный формат. Начните изменение заново из Прайс.")
			return b.sendBarberMenu(chatID)
		}
		b.barberState.Set(chatID, "edit_svc_duration", st.Data+"|"+strconv.Itoa(price*100))
		return b.SendMessage(chatID, "Введите длительность в минутах (число):")
	case "edit_svc_duration":
		b.barberState.Clear(chatID)
		dur, err := strconv.Atoi(strings.TrimSpace(text))
		if err != nil || dur <= 0 {
			b.barberState.Clear(chatID)
			_ = b.SendMessage(chatID, "Неверный формат. Начните изменение заново из Прайс.")
			return b.sendBarberMenu(chatID)
		}
		parts := strings.SplitN(st.Data, "|", 3)
		if len(parts) < 3 {
			b.barberState.Clear(chatID)
			_ = b.SendMessage(chatID, "Ошибка. Начните изменение заново из прайса.")
			return b.sendBarberMenu(chatID)
		}
		id, _ := strconv.ParseInt(parts[0], 10, 64)
		priceCents, _ := strconv.Atoi(parts[2])
		return b.barberUpdateService(ctx, chatID, id, parts[1], priceCents, dur)
	}
	return b.sendBarberMenu(chatID)
}

func (b *Bot) barberAddService(ctx context.Context, chatID int64, name string, priceCents int, durationMin int) error {
	svc := &domain.Service{Name: name, PriceCents: priceCents, DurationMin: durationMin, SortOrder: 0, CreatedAt: time.Now().Unix()}
	if err := b.serviceRepo.Save(ctx, svc); err != nil {
		b.log.Error("barber add service", "err", err)
		_ = b.SendMessage(chatID, "Ошибка сохранения.")
		return b.sendBarberMenu(chatID)
	}
	_ = b.SendMessage(chatID, fmt.Sprintf("Услуга «%s» добавлена. ID: %d", name, svc.ID))
	return b.sendBarberMenu(chatID)
}

func (b *Bot) barberUpdateService(ctx context.Context, chatID int64, id int64, name string, priceCents int, durationMin int) error {
	svc, err := b.serviceRepo.GetByID(ctx, id)
	if err != nil || svc == nil {
		_ = b.SendMessage(chatID, "Услуга не найдена.")
		return b.sendBarberMenu(chatID)
	}
	if name != "" && name != "-" {
		svc.Name = name
	}
	svc.PriceCents = priceCents
	svc.DurationMin = durationMin
	if err := b.serviceRepo.Save(ctx, svc); err != nil {
		_ = b.SendMessage(chatID, "Ошибка сохранения.")
		return b.sendBarberMenu(chatID)
	}
	_ = b.SendMessage(chatID, fmt.Sprintf("Услуга обновлена: %s — %d ₽, %d мин", svc.Name, svc.PriceCents/100, svc.DurationMin))
	return b.sendBarberMenu(chatID)
}

func (b *Bot) barberPriceList(ctx context.Context, chatID int64, barberID int64) error {
	services, err := usecase.PriceList(ctx, b.serviceRepo)
	if err != nil {
		_ = b.SendMessage(chatID, "Ошибка загрузки прайса.")
		return b.sendBarberMenu(chatID)
	}
	if len(services) == 0 {
		msg := tgbotapi.NewMessage(chatID, "Прайс пуст. Нажмите кнопку ниже, чтобы добавить первую услугу.")
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("➕ Добавить услугу", "b_addsvc")),
		)
		_, err = b.api.Send(msg)
		return err
	}
	var lines []string
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, s := range services {
		lines = append(lines, fmt.Sprintf("• %s — %d ₽, %d мин", s.Name, s.PriceCents/100, s.DurationMin))
		// Кнопки с названием услуги, чтобы было понятно, какая к какой услуге относится
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✏️ "+s.Name, "b_editsvc:"+strconv.FormatInt(s.ID, 10)),
			tgbotapi.NewInlineKeyboardButtonData("🗑 "+s.Name, "b_delsvc:"+strconv.FormatInt(s.ID, 10)),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("➕ Добавить услугу", "b_addsvc")))
	msg := tgbotapi.NewMessage(chatID, "Прайс-лист:\n\n"+strings.Join(lines, "\n"))
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	_, err = b.api.Send(msg)
	return err
}

// barberScheduleContent собирает текст и клавиатуру экрана «График» (только рабочие дни, между слотами 1 час).
func (b *Bot) barberScheduleContent(ctx context.Context, barberID int64) (text string, markup tgbotapi.InlineKeyboardMarkup, err error) {
	list, err := b.scheduleRepo.ListWorkingDays(ctx, barberID)
	if err != nil {
		return "", tgbotapi.InlineKeyboardMarkup{}, err
	}
	loc := b.cfg.TZ
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	var visible []*domain.WorkingDay
	for _, w := range list {
		d, err := time.ParseInLocation("2006-01-02", w.WorkDate, loc)
		if err != nil {
			continue
		}
		dayStart := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc)
		if dayStart.Before(today) {
			continue
		}
		visible = append(visible, w)
	}
	list = visible

	var lines []string
	lines = append(lines, "Рабочие дни (между слотами 1 час, МСК):")
	if len(list) == 0 {
		lines = append(lines, "  Нет добавленных дней.")
	} else {
		for _, w := range list {
			lines = append(lines, fmt.Sprintf("  • %s: %s–%s", w.WorkDate, w.StartTime, w.EndTime))
		}
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Добавить рабочий день", "b_sched_work_add")))
	for _, w := range list {
		d, _ := time.Parse("2006-01-02", w.WorkDate)
		short := d.Format("02.01")
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Изм. "+short, "b_sched_work_edit:"+w.WorkDate),
			tgbotapi.NewInlineKeyboardButtonData("Удал. "+short, "b_sched_work_del:"+w.WorkDate),
		))
	}
	text = "График (рабочие дни):\n\n" + strings.Join(lines, "\n")
	markup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	return text, markup, nil
}

func (b *Bot) barberSchedule(ctx context.Context, chatID int64, barberID int64) error {
	text, markup, err := b.barberScheduleContent(ctx, barberID)
	if err != nil {
		_ = b.SendMessage(chatID, "Ошибка загрузки расписания.")
		return b.sendBarberMenu(chatID)
	}
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = markup
	_, err = b.api.Send(msg)
	return err
}

// barberScheduleEdit обновляет существующее сообщение с меню графика (без нового сообщения в чат).
func (b *Bot) barberScheduleEdit(ctx context.Context, chatID int64, messageID int, barberID int64) error {
	text, markup, err := b.barberScheduleContent(ctx, barberID)
	if err != nil {
		_ = b.SendMessage(chatID, "Ошибка загрузки расписания.")
		return b.sendBarberMenu(chatID)
	}
	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ReplyMarkup = &markup
	_, err = b.api.Send(edit)
	return err
}

func (b *Bot) barberAddress(ctx context.Context, chatID int64, barberID int64) error {
	addr, err := b.addressRepo.Get(ctx)
	if err != nil {
		_ = b.SendMessage(chatID, "Ошибка загрузки адреса.")
		return b.sendBarberMenu(chatID)
	}
	displayText := addr.AddressText
	if displayText == "" {
		displayText = "Адрес не задан."
	}
	rows := [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Изменить текст", "b_address_edit_text")),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Изменить фото", "b_address_edit_photo")),
	}
	if addr.AddressPhotoFileID != "" {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Удалить фото", "b_address_del_photo")))
	}
	markup := tgbotapi.NewInlineKeyboardMarkup(rows...)
	if addr.AddressPhotoFileID != "" {
		photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileID(addr.AddressPhotoFileID))
		photo.Caption = "📍 Адрес:\n\n" + displayText
		photo.ReplyMarkup = markup
		_, err = b.api.Send(photo)
		return err
	}
	msg := tgbotapi.NewMessage(chatID, "📍 Адрес:\n\n"+displayText)
	msg.ReplyMarkup = markup
	_, err = b.api.Send(msg)
	return err
}

func (b *Bot) barberVisitsPeriod(ctx context.Context, chatID int64, barberID int64) error {
	workingDays, err := b.scheduleRepo.ListWorkingDays(ctx, barberID)
	if err != nil {
		_ = b.SendMessage(chatID, "Ошибка загрузки графика.")
		return b.sendBarberMenu(chatID)
	}
	loc := b.cfg.TZ
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	fromCut := today.AddDate(0, 0, -7)
	toCut := today.AddDate(0, 0, 30)
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, w := range workingDays {
		d, err := time.ParseInLocation("2006-01-02", w.WorkDate, loc)
		if err != nil {
			continue
		}
		dayStart := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc)
		if dayStart.Before(fromCut) || !dayStart.Before(toCut) {
			continue
		}
		label := d.Format("02.01")
		if dayStart.Equal(today) {
			label = "Сегодня " + label
		} else if dayStart.Equal(today.AddDate(0, 0, 1)) {
			label = "Завтра " + label
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, "b_visits:"+w.WorkDate),
		))
	}
	if len(rows) == 0 {
		_ = b.SendMessage(chatID, "Нет рабочих дней в ближайшие 30 дней. Добавьте дни в График.")
		return b.sendBarberMenu(chatID)
	}
	msg := tgbotapi.NewMessage(chatID, "Выберите рабочий день:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	_, err = b.api.Send(msg)
	return err
}

func (b *Bot) barberClients(ctx context.Context, chatID int64) error {
	list, err := usecase.ListClientsWithVisitCount(ctx, b.clientRepo, b.visitRepo)
	if err != nil {
		_ = b.SendMessage(chatID, "Ошибка загрузки списка клиентов.")
		return b.sendBarberMenu(chatID)
	}
	if len(list) == 0 {
		_ = b.SendMessage(chatID, "Нет зарегистрированных клиентов.")
		return b.sendBarberMenu(chatID)
	}
	var lines []string
	for _, c := range list {
		lines = append(lines, fmt.Sprintf("• %s (@%s) — ID: %d, записей: %d", c.Client.Name, c.Client.Username, c.Client.TelegramID, c.Count))
	}
	msg := tgbotapi.NewMessage(chatID, "Клиенты:\n\n"+strings.Join(lines, "\n")+"\n\nИспользуйте кнопки ниже для других действий.")
	msg.ReplyMarkup = barberMenuReplyKeyboard()
	_, err = b.api.Send(msg)
	return err
}

func (b *Bot) barberBan(ctx context.Context, chatID int64, telegramID int64) error {
	ban := &domain.Ban{ClientTelegramID: telegramID, BannedAt: time.Now().Unix(), Reason: "барбер"}
	if err := b.banRepo.Ban(ctx, ban); err != nil {
		_ = b.SendMessage(chatID, "Ошибка (возможно уже забанен).")
		return b.sendBarberMenu(chatID)
	}
	_ = b.SendMessage(chatID, fmt.Sprintf("Клиент %d заблокирован.", telegramID))
	return b.sendBarberMenu(chatID)
}

func (b *Bot) barberUnban(ctx context.Context, chatID int64, telegramID int64) error {
	if err := b.banRepo.Unban(ctx, telegramID); err != nil {
		_ = b.SendMessage(chatID, "Ошибка снятия бана.")
		return b.sendBarberMenu(chatID)
	}
	_ = b.SendMessage(chatID, fmt.Sprintf("Клиент %d разблокирован.", telegramID))
	return b.sendBarberMenu(chatID)
}

func (b *Bot) barberDoBroadcast(ctx context.Context, chatID int64, text string) error {
	ids, err := usecase.BroadcastRecipients(ctx, b.clientRepo, b.banRepo)
	if err != nil {
		_ = b.SendMessage(chatID, "Ошибка получения списка получателей.")
		return b.sendBarberMenu(chatID)
	}
	sent := 0
	for _, id := range ids {
		if err := b.SendMessage(id, text); err != nil {
			b.log.Error("broadcast send", "err", err, "to", id)
		} else {
			sent++
		}
	}
	_ = b.auditRepo.Append(ctx, "broadcast", fmt.Sprintf("sent=%d,text_len=%d", sent, len(text)))
	_ = b.SendMessage(chatID, fmt.Sprintf("Рассылка отправлена %d клиентам.", sent))
	return b.sendBarberMenu(chatID)
}

func (b *Bot) barberStatsPeriod(ctx context.Context, chatID int64, barberID int64) error {
	msg := tgbotapi.NewMessage(chatID, "Статистика за период:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("День", "b_stats:day"),
			tgbotapi.NewInlineKeyboardButtonData("Неделя", "b_stats:week"),
			tgbotapi.NewInlineKeyboardButtonData("Месяц", "b_stats:month"),
		),
	)
	_, err := b.api.Send(msg)
	return err
}

func (b *Bot) handleBarberCallback(ctx context.Context, chatID int64, barberID int64, cb *tgbotapi.CallbackQuery) error {
	_, _ = b.api.Request(tgbotapi.NewCallback(cb.ID, ""))
	data := cb.Data

	if strings.HasPrefix(data, "b_delsvc:") {
		idStr := strings.TrimPrefix(data, "b_delsvc:")
		id, _ := strconv.ParseInt(idStr, 10, 64)
		if err := b.serviceRepo.Delete(ctx, id); err != nil {
			_ = b.SendMessage(chatID, "Ошибка удаления.")
			return b.sendBarberMenu(chatID)
		}
		_ = b.SendMessage(chatID, "Услуга удалена.")
		return b.sendBarberMenu(chatID)
	}
	if data == "b_addsvc" {
		b.barberState.Set(chatID, "service_name", "")
		return b.SendMessage(chatID, "Введите название услуги. Назад — отмена.")
	}
	if strings.HasPrefix(data, "b_editsvc:") {
		idStr := strings.TrimPrefix(data, "b_editsvc:")
		id, _ := strconv.ParseInt(idStr, 10, 64)
		svc, err := b.serviceRepo.GetByID(ctx, id)
		if err != nil || svc == nil {
			_ = b.SendMessage(chatID, "Услуга не найдена.")
			return b.sendBarberMenu(chatID)
		}
		b.barberState.Set(chatID, "edit_svc_name", idStr)
		return b.SendMessage(chatID, fmt.Sprintf("Текущее название: %s. Введите новое название или «-» чтобы оставить:", svc.Name))
	}
	if data == "b_sched_work_add" {
		b.barberState.Set(chatID, "schedule_work_add", "")
		return b.SendMessage(chatID, "Введите дату рабочего дня (ГГГГ-ММ-ДД, например 2025-03-15). Назад — отмена.")
	}
	if strings.HasPrefix(data, "b_sched_work_del:") {
		workDate := strings.TrimPrefix(data, "b_sched_work_del:")
		if err := b.scheduleRepo.RemoveWorkingDay(ctx, barberID, workDate); err != nil {
			_ = b.SendMessage(chatID, "Ошибка удаления.")
			return b.sendBarberMenu(chatID)
		}
		return b.barberScheduleEdit(ctx, chatID, cb.Message.MessageID, barberID)
	}
	if strings.HasPrefix(data, "b_sched_work_edit:") {
		workDate := strings.TrimPrefix(data, "b_sched_work_edit:")
		b.barberState.Set(chatID, "schedule_work_edit", workDate)
		return b.SendMessage(chatID, "Введите новое время начала и окончания (МСК). Например: 11:00 22:00. Назад — отмена.")
	}
	if data == "b_address_edit_text" {
		b.barberState.Set(chatID, "address_edit_text", "")
		return b.SendMessage(chatID, "Отправьте новый текст адреса (одним сообщением). Назад — отмена.")
	}
	if data == "b_address_edit_photo" {
		b.barberState.Set(chatID, "address_edit_photo", "")
		return b.SendMessage(chatID, "Отправьте новое фото (одним сообщением). Назад — отмена.")
	}
	if data == "b_address_del_photo" {
		addr, err := b.addressRepo.Get(ctx)
		if err != nil {
			_ = b.SendMessage(chatID, "Ошибка загрузки адреса.")
			return b.sendBarberMenu(chatID)
		}
		addr.AddressPhotoFileID = ""
		if err := b.addressRepo.Set(ctx, addr); err != nil {
			_ = b.SendMessage(chatID, "Ошибка сохранения.")
			return b.sendBarberMenu(chatID)
		}
		return b.barberAddress(ctx, chatID, barberID)
	}
	if strings.HasPrefix(data, "b_visits:") {
		period := strings.TrimPrefix(data, "b_visits:")
		return b.barberShowVisits(ctx, chatID, barberID, period)
	}
	if strings.HasPrefix(data, "b_stats:") {
		period := strings.TrimPrefix(data, "b_stats:")
		return b.barberShowStats(ctx, chatID, barberID, period)
	}
	if data == "b_cancel_ask" {
		b.barberState.Set(chatID, "cancel_visit_id", "")
		return b.SendMessage(chatID, "Введите номер записи для отмены (визит #N из списка выше). Назад — отмена.")
	}
	return b.sendBarberMenu(chatID)
}

func (b *Bot) barberShowVisits(ctx context.Context, chatID int64, barberID int64, dateStr string) error {
	loc := b.cfg.TZ
	date, err := time.ParseInLocation("2006-01-02", dateStr, loc)
	if err != nil {
		_ = b.SendMessage(chatID, "Неверная дата.")
		return b.sendBarberMenu(chatID)
	}
	from := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, loc)
	to := from.Add(24 * time.Hour)
	visits, err := b.visitRepo.ListByBarber(ctx, barberID, from.Unix(), to.Unix())
	if err != nil {
		_ = b.SendMessage(chatID, "Ошибка загрузки записей.")
		return b.sendBarberMenu(chatID)
	}
	if len(visits) == 0 {
		_ = b.SendMessage(chatID, "Нет записей на "+dateStr+".")
		return b.sendBarberMenu(chatID)
	}
	var lines []string
	for _, v := range visits {
		t := time.Unix(v.StartsAt, 0).In(loc)
		clientLabel := fmt.Sprintf("клиент %d", v.ClientID)
		if c, _ := b.clientRepo.GetByID(ctx, v.ClientID); c != nil && c.Username != "" {
			clientLabel = "@" + c.Username
		} else if c != nil && c.Name != "" {
			clientLabel = c.Name
		}
		svcNames, _ := b.visitRepo.GetServicesByVisitID(ctx, v.ID)
		svcStr := ""
		for i, s := range svcNames {
			if i > 0 {
				svcStr += ", "
			}
			svcStr += s.Name
		}
		if svcStr == "" {
			svcStr = "—"
		}
		line := fmt.Sprintf("• %s %s — визит #%d %s (%s)", t.Format("02.01"), t.Format("15:04"), v.ID, clientLabel, svcStr)
		if lbl := visitStatusLabel(v.Status); lbl != "" {
			line += " " + lbl
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		_ = b.SendMessage(chatID, "Нет записей на "+dateStr+".")
		return b.sendBarberMenu(chatID)
	}
	// Одна кнопка «Отменить запись» — затем барбер вводит номер визита.
	hasScheduled := false
	for _, v := range visits {
		if v.Status == "scheduled" {
			hasScheduled = true
			break
		}
	}
	msg := tgbotapi.NewMessage(chatID, "Записи на "+dateStr+":\n\n"+strings.Join(lines, "\n")+"\n\nИспользуйте кнопки меню ниже для других действий.")
	if hasScheduled {
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Отменить запись", "b_cancel_ask")),
		)
	} else {
		msg.ReplyMarkup = barberMenuReplyKeyboard()
	}
	_, err = b.api.Send(msg)
	if err != nil {
		return err
	}
	return b.sendBarberMenu(chatID)
}

func (b *Bot) barberCancelVisit(ctx context.Context, chatID int64, barberID int64, visitID int64) error {
	v, err := b.visitRepo.GetByID(ctx, visitID)
	if err != nil || v == nil || v.Status != "scheduled" {
		_ = b.SendMessage(chatID, "Запись не найдена или уже отменена.")
		return b.sendBarberMenu(chatID)
	}
	svcs, _ := b.visitRepo.GetServicesByVisitID(ctx, visitID)
	loc := b.cfg.TZ
	t := time.Unix(v.StartsAt, 0).In(loc)
	dateTime := t.Format("02.01.2006") + " в " + t.Format("15:04")
	svcNames := make([]string, 0, len(svcs))
	for _, s := range svcs {
		svcNames = append(svcNames, s.Name)
	}
	svcStr := strings.Join(svcNames, ", ")
	if svcStr == "" {
		svcStr = "—"
	}
	clientMsg := fmt.Sprintf("Ваша запись %s (%s) отменена.", dateTime, svcStr)

	clientTGID, err := usecase.CancelVisitByBarber(ctx, visitID, "барбер", b.visitRepo, b.clientRepo, b.auditRepo)
	if err != nil {
		_ = b.SendMessage(chatID, "Запись не найдена или уже отменена.")
		return b.sendBarberMenu(chatID)
	}
	_ = b.SendMessage(clientTGID, clientMsg)
	_ = b.SendMessage(chatID, "Запись отменена, клиент уведомлён.")
	return b.sendBarberMenu(chatID)
}

func (b *Bot) barberShowStats(ctx context.Context, chatID int64, barberID int64, period string) error {
	loc := b.cfg.TZ
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	// Статистика за уже прошедшие периоды (только неу отменённые визиты учитываются в usecase.Stats).
	var from, to time.Time
	switch period {
	case "day":
		from = today.AddDate(0, 0, -1)
		to = today
	case "week":
		from = today.AddDate(0, 0, -7)
		to = today
	case "month":
		from = today.AddDate(0, 0, -30)
		to = today
	default:
		from = today.AddDate(0, 0, -1)
		to = today
	}
	res, err := usecase.Stats(ctx, barberID, from.Unix(), to.Unix(), b.visitRepo, b.clientRepo, b.serviceRepo)
	if err != nil {
		_ = b.SendMessage(chatID, "Ошибка расчёта статистики.")
		return b.sendBarberMenu(chatID)
	}
	var periodLabel string
	switch period {
	case "day":
		periodLabel = "за вчера"
	case "week":
		periodLabel = "за последние 7 дней"
	case "month":
		periodLabel = "за последние 30 дней"
	default:
		periodLabel = "за период"
	}
	var lines []string
	lines = append(lines, "Статистика "+periodLabel)
	lines = append(lines, fmt.Sprintf("Выручка: %d ₽", res.RevenueCents/100))
	lines = append(lines, fmt.Sprintf("Визитов: %d", res.VisitCount))
	lines = append(lines, "")
	lines = append(lines, "По услугам:")
	for _, s := range res.ByService {
		lines = append(lines, fmt.Sprintf("  • %s — %d раз, %d ₽", s.ServiceName, s.Count, s.SumCents/100))
	}
	lines = append(lines, "")
	lines = append(lines, "Топ клиентов:")
	for i, c := range res.TopClients {
		if i >= 5 {
			break
		}
		lines = append(lines, fmt.Sprintf("  • %s — %d визитов", c.Client.Name, c.Count))
	}
	msg := tgbotapi.NewMessage(chatID, strings.Join(lines, "\n")+"\n\nИспользуйте кнопки ниже для других действий.")
	msg.ReplyMarkup = barberMenuReplyKeyboard()
	_, err = b.api.Send(msg)
	return err
}

