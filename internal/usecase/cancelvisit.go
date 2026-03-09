package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"barber_bot/internal/port"
)

var ErrVisitNotFound = errors.New("visit not found")
var ErrNotYourVisit = errors.New("not your visit")
var ErrVisitPast = errors.New("visit already in the past")

// CancelVisit отменяет запись клиента. Проверяет, что визит принадлежит клиенту и ещё не наступил.
func CancelVisit(
	ctx context.Context,
	visitID int64,
	clientID int64,
	visitRepo port.VisitRepository,
	auditRepo port.AuditLogRepository,
) error {
	v, err := visitRepo.GetByID(ctx, visitID)
	if err != nil {
		return err
	}
	if v == nil {
		return ErrVisitNotFound
	}
	if v.ClientID != clientID {
		return ErrNotYourVisit
	}
	if v.Status != "scheduled" {
		return ErrVisitNotFound
	}
	now := time.Now().Unix()
	if v.StartsAt <= now {
		return ErrVisitPast
	}

	if err := visitRepo.UpdateStatus(ctx, visitID, "cancelled"); err != nil {
		return err
	}
	_ = auditRepo.Append(ctx, "visit_cancelled_by_client", formatCancelPayload(visitID, clientID))
	return nil
}

func formatCancelPayload(visitID, clientID int64) string {
	return fmtPayload(visitID, clientID)
}

func fmtPayload(visitID, clientID int64) string {
	return fmt.Sprintf("visit=%d,client=%d", visitID, clientID)
}
