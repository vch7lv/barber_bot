package usecase

import (
	"context"
	"testing"
	"time"

	"barber_bot/internal/domain"
	"barber_bot/internal/port"
)

type mockVisitRepoMyVisits struct {
	visits []*domain.Visit
}

func (m *mockVisitRepoMyVisits) GetByID(ctx context.Context, id int64) (*domain.Visit, error) { return nil, nil }
func (m *mockVisitRepoMyVisits) GetServicesByVisitID(ctx context.Context, visitID int64) ([]*domain.Service, error) {
	return []*domain.Service{{Name: "Стрижка"}}, nil
}
func (m *mockVisitRepoMyVisits) ListByClient(ctx context.Context, clientID int64, from, to int64) ([]*domain.Visit, error) {
	return nil, nil
}
func (m *mockVisitRepoMyVisits) ListByClientTelegramID(ctx context.Context, clientTelegramID int64, from, to int64) ([]*domain.Visit, error) {
	return m.visits, nil
}
func (m *mockVisitRepoMyVisits) ListByBarber(ctx context.Context, barberID int64, from, to int64) ([]*domain.Visit, error) {
	return nil, nil
}
func (m *mockVisitRepoMyVisits) VisitsByBarberInRange(ctx context.Context, barberID int64, from, to int64) ([]*domain.Visit, error) {
	return nil, nil
}
func (m *mockVisitRepoMyVisits) ListScheduledInRange(ctx context.Context, from, to int64) ([]*domain.Visit, error) {
	return nil, nil
}
func (m *mockVisitRepoMyVisits) CountByClient(ctx context.Context, clientID int64) (int, error) { return 0, nil }
func (m *mockVisitRepoMyVisits) Save(ctx context.Context, v *domain.Visit, serviceIDs []int64) error {
	return nil
}
func (m *mockVisitRepoMyVisits) UpdateStatus(ctx context.Context, id int64, status string) error { return nil }

func TestMyVisits_ExcludesPastStartEvenWithLongDuration(t *testing.T) {
	ctx := context.Background()
	past := time.Now().Unix() - 3600
	future := time.Now().Unix() + 3600
	repo := &mockVisitRepoMyVisits{
		visits: []*domain.Visit{
			{ID: 1, StartsAt: past, DurationMin: 9999, Status: "scheduled"},
			{ID: 2, StartsAt: future, DurationMin: 30, Status: "scheduled"},
		},
	}
	list, err := MyVisits(ctx, 100, 0, future+1, repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Visit.ID != 2 {
		t.Fatalf("want only future visit id=2, got %+v", list)
	}
	_ = port.VisitRepository(repo)
}
