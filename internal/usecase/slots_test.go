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
	defaultSchedule *domain.DefaultSchedule
	daysOff         map[string]bool
}

func (m *mockScheduleRepo) GetDefaultSchedule(ctx context.Context, barberID int64) (*domain.DefaultSchedule, error) {
	return m.defaultSchedule, nil
}
func (m *mockScheduleRepo) SetDefaultSchedule(ctx context.Context, s *domain.DefaultSchedule) error { return nil }
func (m *mockScheduleRepo) IsDayOff(ctx context.Context, barberID int64, dateStr string) (bool, error) {
	return m.daysOff[dateStr], nil
}
func (m *mockScheduleRepo) AddDayOff(ctx context.Context, barberID int64, offDate string) error {
	return nil
}
func (m *mockScheduleRepo) RemoveDayOff(ctx context.Context, barberID int64, offDate string) error {
	return nil
}
func (m *mockScheduleRepo) ListDaysOff(ctx context.Context, barberID int64) ([]*domain.DayOff, error) {
	return nil, nil
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

func TestFreeSlots_DayOff_ReturnsEmpty(t *testing.T) {
	ctx := context.Background()
	loc := time.UTC
	date := time.Date(2025, 3, 15, 0, 0, 0, 0, loc)
	sched := &mockScheduleRepo{
		daysOff: map[string]bool{"2025-03-15": true},
	}
	visitRepo := &mockVisitRepoForSlots{}
	log := slog.Default()

	slots, err := FreeSlots(ctx, 1, date, 30, loc, sched, visitRepo, log)
	if err != nil {
		t.Fatal(err)
	}
	if len(slots) != 0 {
		t.Errorf("expected 0 slots on day off, got %d", len(slots))
	}
}

func TestFreeSlots_DefaultSchedule_ReturnsSlots(t *testing.T) {
	ctx := context.Background()
	loc := time.UTC
	date := time.Date(2025, 3, 16, 0, 0, 0, 0, loc)
	sched := &mockScheduleRepo{
		defaultSchedule: nil, // use domain default 11:00-22:00, step 30
		daysOff:         nil,
	}
	visitRepo := &mockVisitRepoForSlots{}

	log := slog.Default()
	slots, err := FreeSlots(ctx, 1, date, 30, loc, sched, visitRepo, log)
	if err != nil {
		t.Fatal(err)
	}
	// 11:00 to 22:00 with 30 min step: 11:00, 11:30, ..., 21:30 (slot + 30min must be <= 22:00)
	// 22*30min slots from 11:00
	if len(slots) < 10 {
		t.Errorf("expected many slots with default schedule, got %d", len(slots))
	}
	// ensure port is satisfied
	_ = port.ScheduleRepository(sched)
}
