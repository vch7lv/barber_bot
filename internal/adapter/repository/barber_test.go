package repository

import (
	"context"
	"log/slog"
	"testing"

	"barber_bot/internal/domain"
)

// TestBarber_MultipleTelegramIDs_SameBarber проверяет, что несколько TG-аккаунтов
// барбера привязаны к одному barber_id; расписание, заданное с любого аккаунта, видно клиенту.
func TestBarber_MultipleTelegramIDs_SameBarber(t *testing.T) {
	ctx := context.Background()
	log := slog.Default()
	db, err := DB(ctx, "file::memory:?cache=shared", log)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	barberRepo := NewBarberRepo(db)
	scheduleRepo := NewScheduleRepo(db)

	// Два аккаунта барбера — один логический барбер
	if err := barberRepo.EnsureBarbers(ctx, []int64{111, 222}); err != nil {
		t.Fatalf("EnsureBarbers: %v", err)
	}

	b1, err := barberRepo.GetByTelegramID(ctx, 111)
	if err != nil {
		t.Fatalf("GetByTelegramID(111): %v", err)
	}
	if b1 == nil {
		t.Fatal("GetByTelegramID(111): nil")
	}
	b2, err := barberRepo.GetByTelegramID(ctx, 222)
	if err != nil {
		t.Fatalf("GetByTelegramID(222): %v", err)
	}
	if b2 == nil {
		t.Fatal("GetByTelegramID(222): nil")
	}
	if b1.ID != b2.ID {
		t.Errorf("оба аккаунта должны быть одним барбером: id %d vs %d", b1.ID, b2.ID)
	}

	canonicalID := b1.ID
	// «Второй» аккаунт (222) добавляет рабочий день — он сохраняется под общим barber_id
	wd := &domain.WorkingDay{
		BarberID:  canonicalID,
		WorkDate:  "2025-03-20",
		StartTime: "10:00",
		EndTime:   "18:00",
	}
	if err := scheduleRepo.SetWorkingDay(ctx, wd); err != nil {
		t.Fatalf("SetWorkingDay: %v", err)
	}

	// Клиент смотрит расписание по firstBarberID (тот же canonicalID) — день должен быть виден
	got, err := scheduleRepo.GetWorkingDay(ctx, canonicalID, "2025-03-20")
	if err != nil {
		t.Fatalf("GetWorkingDay: %v", err)
	}
	if got == nil {
		t.Fatal("клиент не видит рабочий день, добавленный со второго аккаунта барбера")
	}
	if got.WorkDate != "2025-03-20" || got.StartTime != "10:00" || got.EndTime != "18:00" {
		t.Errorf("GetWorkingDay: got %+v", got)
	}

	// В БД должна быть ровно одна запись барбера
	var barberCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM barbers`).Scan(&barberCount); err != nil {
		t.Fatalf("count barbers: %v", err)
	}
	if barberCount != 1 {
		t.Errorf("expected 1 barber row, got %d", barberCount)
	}
}
