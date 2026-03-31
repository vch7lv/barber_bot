package usecase

import (
	"context"
	"errors"
	"fmt"

	"barber_bot/internal/port"
)

var ErrVisitNotFoundBarber = errors.New("visit not found")

// CancelVisitByBarber отменяет запись барбером. Возвращает Telegram ID клиента для уведомления.
func CancelVisitByBarber(
	ctx context.Context,
	visitID int64,
	reason string,
	visitRepo port.VisitRepository,
	clientRepo port.ClientRepository,
	auditRepo port.AuditLogRepository,
) (clientTelegramID int64, err error) {
	v, err := visitRepo.GetByID(ctx, visitID)
	if err != nil {
		return 0, err
	}
	if v == nil || v.Status != "scheduled" {
		return 0, ErrVisitNotFoundBarber
	}
	client, err := clientRepo.GetByID(ctx, v.ClientID)
	if err != nil {
		return 0, err
	}
	if client == nil {
		return 0, ErrVisitNotFoundBarber
	}
	if err := visitRepo.UpdateStatus(ctx, visitID, "cancelled_by_barber"); err != nil {
		return 0, err
	}
	payload := formatCancelByBarberPayload(visitID, v.ClientID, reason)
	_ = auditRepo.Append(ctx, "visit_cancelled_by_barber", payload)
	return client.TelegramID, nil
}

func formatCancelByBarberPayload(visitID, clientID int64, reason string) string {
	return fmt.Sprintf("visit=%d,client=%d,reason=%s", visitID, clientID, reason)
}
