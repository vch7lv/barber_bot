package usecase

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"barber_bot/internal/domain"
	"barber_bot/internal/port"
)

type mockScheduleRepo struct {
	workingDay map[string]*domain.WorkingDay // dateStr -> working day
}

func (m *mockScheduleRepo) GetWorkingDay(ctx context.Context, barberID int64, dateStr string) (*domain.WorkingDay, error) {
	if m.workingDay == nil {
		return nil, nil
	}
	return m.workingDay[dateStr], nil
}
func (m *mockScheduleRepo) ListWorkingDays(ctx context.Context, barberID int64) ([]*domain.WorkingDay, error) {
	return nil, nil
}
func (m *mockScheduleRepo) SetWorkingDay(ctx context.Context, w *domain.WorkingDay) error {
	return nil
}
func (m *mockScheduleRepo) RemoveWorkingDay(ctx context.Context, barberID int64, dateStr string) error {
	return nil
}

type mockVisitRepoForSlots struct {
	visits []*domain.Visit
}

func (m *mockVisitRepoForSlots) GetByID(ctx context.Context, id int64) (*domain.Visit, error)           { return nil, nil }
func (m *mockVisitRepoForSlots) GetServicesByVisitID(ctx context.Context, visitID int64) ([]*domain.Service, error) {
	return nil, nil
}
func (m *mockVisitRepoForSlots) ListByClient(ctx context.Context, clientID int64, from, to int64) ([]*domain.Visit, error) {
	return nil, nil
}
func (m *mockVisitRepoForSlots) ListByClientTelegramID(ctx context.Context, clientTelegramID int64, from, to int64) ([]*domain.Visit, error) {
	return nil, nil
}
func (m *mockVisitRepoForSlots) ListByBarber(ctx context.Context, barberID int64, from, to int64) ([]*domain.Visit, error) {
	return nil, nil
}
func (m *mockVisitRepoForSlots) VisitsByBarberInRange(ctx context.Context, barberID int64, from, to int64) ([]*domain.Visit, error) {
	return m.visits, nil
}
func (m *mockVisitRepoForSlots) ListScheduledInRange(ctx context.Context, from, to int64) ([]*domain.Visit, error) {
	return nil, nil
}
func (m *mockVisitRepoForSlots) CountByClient(ctx context.Context, clientID int64) (int, error) { return 0, nil }
func (m *mockVisitRepoForSlots) Save(ctx context.Context, v *domain.Visit, serviceIDs []int64) error { return nil }
func (m *mockVisitRepoForSlots) UpdateStatus(ctx context.Context, id int64, status string) error   { return nil }

func TestFreeSlots_PastCalendarDay_ReturnsEmpty(t *testing.T) {
	ctx := context.Background()
	loc := time.UTC
	past := time.Now().In(loc).Add(-48 * time.Hour)
	date := time.Date(past.Year(), past.Month(), past.Day(), 0, 0, 0, 0, loc)
	dateStr := date.Format("2006-01-02")
	sched := &mockScheduleRepo{
		workingDay: map[string]*domain.WorkingDay{
			dateStr: {BarberID: 1, WorkDate: dateStr, StartTime: "09:00", EndTime: "18:00"},
		},
	}
	visitRepo := &mockVisitRepoForSlots{}
	log := slog.Default()
	slots, err := FreeSlots(ctx, 1, date, 60, loc, sched, visitRepo, log)
	if err != nil {
		t.Fatal(err)
	}
	if len(slots) != 0 {
		t.Errorf("expected 0 slots for past calendar day, got %d", len(slots))
	}
}

func TestFreeSlots_NoWorkingDay_ReturnsEmpty(t *testing.T) {
	ctx := context.Background()
	loc := time.UTC
	date := time.Date(2025, 3, 15, 0, 0, 0, 0, loc)
	sched := &mockScheduleRepo{
		workingDay: nil, // no working day for 2025-03-15
	}
	visitRepo := &mockVisitRepoForSlots{}
	log := slog.Default()

	slots, err := FreeSlots(ctx, 1, date, 60, loc, sched, visitRepo, log)
	if err != nil {
		t.Fatal(err)
	}
	if len(slots) != 0 {
		t.Errorf("expected 0 slots when no working day, got %d", len(slots))
	}
}

func TestFreeSlots_WorkingDay_ReturnsSlots(t *testing.T) {
	ctx := context.Background()
	loc := time.UTC
	// Дата в будущем: FreeSlots отбрасывает прошедшие календарные дни и слоты «сегодня» в прошлом.
	future := time.Now().In(loc).Add(72 * time.Hour)
	date := time.Date(future.Year(), future.Month(), future.Day(), 0, 0, 0, 0, loc)
	dateStr := date.Format("2006-01-02")
	sched := &mockScheduleRepo{
		workingDay: map[string]*domain.WorkingDay{
			dateStr: {BarberID: 1, WorkDate: dateStr, StartTime: "11:00", EndTime: "15:00"},
		},
	}
	visitRepo := &mockVisitRepoForSlots{}

	log := slog.Default()
	slots, err := FreeSlots(ctx, 1, date, 60, loc, sched, visitRepo, log)
	if err != nil {
		t.Fatal(err)
	}
	// 11:00 to 15:00 with 1h step: 11:00, 12:00, 13:00, 14:00 (slot+1h <= 15:00) = 4 slots
	if len(slots) != 4 {
		t.Errorf("expected 4 slots (11:00-15:00, 1h step), got %d", len(slots))
	}
	_ = port.ScheduleRepository(sched)
}
