package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"barber_bot/internal/domain"
	"barber_bot/internal/port"
)

var ErrBanned = errors.New("client is banned")
var ErrNoServices = errors.New("at least one service required")
var ErrSlotInPast = errors.New("slot is in the past")

// BookVisit создаёт визит: проверяет бан, длительность по услугам, сохраняет визит и связи.
func BookVisit(
	ctx context.Context,
	clientID int64,
	clientTelegramID int64,
	barberID int64,
	startUnix int64,
	serviceIDs []int64,
	banRepo port.BanRepository,
	visitRepo port.VisitRepository,
	serviceRepo port.ServiceRepository,
	auditRepo port.AuditLogRepository,
) (*domain.Visit, error) {
	banned, err := banRepo.IsBanned(ctx, clientTelegramID)
	if err != nil {
		return nil, err
	}
	if banned {
		return nil, ErrBanned
	}
	if clientID <= 0 {
		return nil, fmt.Errorf("client_id must be positive, got %d", clientID)
	}
	if len(serviceIDs) == 0 {
		return nil, ErrNoServices
	}

	var services []*domain.Service
	for _, id := range serviceIDs {
		s, err := serviceRepo.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if s == nil {
			continue
		}
		services = append(services, s)
	}
	if len(services) == 0 {
		return nil, ErrNoServices
	}

	durationMin := domain.VisitDurationMinutes(services)
	now := time.Now().Unix()
	if startUnix <= now {
		return nil, ErrSlotInPast
	}

	v := &domain.Visit{
		ClientID:    clientID,
		BarberID:    barberID,
		StartsAt:    startUnix,
		DurationMin: durationMin,
		CreatedAt:   now,
		Status:      "scheduled",
	}
	ids := make([]int64, 0, len(services))
	for _, s := range services {
		ids = append(ids, s.ID)
	}
	if err := visitRepo.Save(ctx, v, ids); err != nil {
		return nil, err
	}

	_ = auditRepo.Append(ctx, "visit_created", formatVisitPayload(v.ID, clientID, barberID, startUnix))
	return v, nil
}

func formatVisitPayload(visitID, clientID, barberID, startUnix int64) string {
	return fmt.Sprintf("visit=%d,client=%d,barber=%d,starts=%d", visitID, clientID, barberID, startUnix)
}
