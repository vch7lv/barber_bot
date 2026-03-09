package port

import (
	"context"
)

// AuditLogRepository — лог событий для аудита и диагностики.
type AuditLogRepository interface {
	Append(ctx context.Context, eventType, payload string) error
}
