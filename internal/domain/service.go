package domain

// Service — услуга в прайс-листе.
type Service struct {
	ID          int64
	Name        string
	PriceCents  int
	DurationMin int
	SortOrder   int
	CreatedAt   int64
}
