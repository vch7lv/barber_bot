package domain

// Visit — запись на визит (дата, время начала, длительность, статус).
type Visit struct {
	ID          int64
	ClientID    int64
	BarberID    int64
	StartsAt    int64 // unix timestamp (UTC)
	DurationMin int
	CreatedAt   int64
	Status      string // scheduled, cancelled, completed
}

// VisitService — связь визита и услуги (в одном визите может быть несколько услуг).
type VisitService struct {
	VisitID   int64
	ServiceID int64
}
