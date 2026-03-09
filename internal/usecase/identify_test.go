package usecase

import (
	"context"
	"testing"

	"barber_bot/internal/config"
	"barber_bot/internal/domain"
	"barber_bot/internal/port"
)

type mockClientRepo struct {
	get  map[int64]*domain.Client
	save []*domain.Client
}

func (m *mockClientRepo) GetByID(ctx context.Context, id int64) (*domain.Client, error) {
	for _, c := range m.get {
		if c != nil && c.ID == id {
			return c, nil
		}
	}
	return nil, nil
}

func (m *mockClientRepo) ListAll(ctx context.Context) ([]*domain.Client, error) {
	var list []*domain.Client
	for _, c := range m.get {
		if c != nil {
			list = append(list, c)
		}
	}
	return list, nil
}

func (m *mockClientRepo) GetByTelegramID(ctx context.Context, telegramID int64) (*domain.Client, error) {
	return m.get[telegramID], nil
}

func (m *mockClientRepo) Save(ctx context.Context, c *domain.Client) error {
	m.save = append(m.save, c)
	if c.ID == 0 {
		c.ID = int64(len(m.save))
	}
	return nil
}

type mockBarberRepo struct {
	byTelegram map[int64]*domain.Barber
}

func (m *mockBarberRepo) GetByTelegramID(ctx context.Context, telegramID int64) (*domain.Barber, error) {
	return m.byTelegram[telegramID], nil
}

func (m *mockBarberRepo) EnsureBarbers(ctx context.Context, telegramIDs []int64) error {
	return nil
}

func TestIdentifyUser_Client(t *testing.T) {
	cfg := &config.Config{BarberTelegramIDs: []int64{999}}
	clientRepo := &mockClientRepo{get: map[int64]*domain.Client{}}
	barberRepo := &mockBarberRepo{byTelegram: map[int64]*domain.Barber{}}

	res, err := IdentifyUser(context.Background(), cfg, clientRepo, barberRepo, 111, "Ivan", "ivan_ok")
	if err != nil {
		t.Fatal(err)
	}
	if res.Role != "client" {
		t.Errorf("role = %s, want client", res.Role)
	}
	if res.BarberID != 0 {
		t.Errorf("barberID = %d, want 0", res.BarberID)
	}
	if res.Client == nil || res.Client.TelegramID != 111 || res.Client.Name != "Ivan" {
		t.Errorf("client = %+v", res.Client)
	}
	if len(clientRepo.save) != 1 {
		t.Errorf("Save called %d times, want 1", len(clientRepo.save))
	}
}

func TestIdentifyUser_Barber(t *testing.T) {
	cfg := &config.Config{BarberTelegramIDs: []int64{222}}
	clientRepo := &mockClientRepo{get: map[int64]*domain.Client{}}
	barberRepo := &mockBarberRepo{byTelegram: map[int64]*domain.Barber{222: {ID: 5, TelegramID: 222}}}

	res, err := IdentifyUser(context.Background(), cfg, clientRepo, barberRepo, 222, "Barber", "barber_tg")
	if err != nil {
		t.Fatal(err)
	}
	if res.Role != "barber" {
		t.Errorf("role = %s, want barber", res.Role)
	}
	if res.BarberID != 5 {
		t.Errorf("barberID = %d, want 5", res.BarberID)
	}
	if res.Client == nil || res.Client.TelegramID != 222 {
		t.Errorf("client = %+v", res.Client)
	}
}

func TestIdentifyUser_ExistingClientUpdate(t *testing.T) {
	cfg := &config.Config{BarberTelegramIDs: []int64{}}
	clientRepo := &mockClientRepo{
		get: map[int64]*domain.Client{333: {ID: 10, TelegramID: 333, Name: "Old", Username: "old_u"}},
	}
	barberRepo := &mockBarberRepo{byTelegram: map[int64]*domain.Barber{}}

	res, err := IdentifyUser(context.Background(), cfg, clientRepo, barberRepo, 333, "NewName", "new_username")
	if err != nil {
		t.Fatal(err)
	}
	if res.Client.Name != "NewName" || res.Client.Username != "new_username" {
		t.Errorf("client not updated: %+v", res.Client)
	}
	// ensure port interfaces are satisfied
	_ = port.ClientRepository(clientRepo)
	_ = port.BarberRepository(barberRepo)
}
