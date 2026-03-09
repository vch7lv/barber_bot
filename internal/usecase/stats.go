package usecase

import (
	"context"
	"sort"

	"barber_bot/internal/domain"
	"barber_bot/internal/port"
)

// ServiceStat — статистика по услуге (B8).
type ServiceStat struct {
	ServiceName string
	Count       int
	SumCents    int
}

// ClientStat — клиент и количество визитов (топ).
type ClientStat struct {
	Client *domain.Client
	Count  int
}

// StatsResult — результат расчёта статистики за период.
type StatsResult struct {
	RevenueCents int
	VisitCount   int
	ByService    []ServiceStat
	TopClients   []ClientStat
}

// Stats возвращает статистику барбера за период [fromUnix, toUnix].
// Учитываются только визиты, не отменённые (scheduled и completed); отменённые не идут в выручку и счётчики.
func Stats(
	ctx context.Context,
	barberID int64,
	fromUnix, toUnix int64,
	visitRepo port.VisitRepository,
	clientRepo port.ClientRepository,
	serviceRepo port.ServiceRepository,
) (*StatsResult, error) {
	visits, err := visitRepo.ListByBarber(ctx, barberID, fromUnix, toUnix)
	if err != nil {
		return nil, err
	}

	// Только неу отменённые визиты (scheduled + completed)
	var activeVisits []*domain.Visit
	for _, v := range visits {
		if v.Status == "scheduled" || v.Status == "completed" {
			activeVisits = append(activeVisits, v)
		}
	}

	serviceCount := make(map[int64]int)
	serviceSum := make(map[int64]int)
	clientCount := make(map[int64]int)
	var revenueCents int

	for _, v := range activeVisits {
		clientCount[v.ClientID]++
		svcs, err := visitRepo.GetServicesByVisitID(ctx, v.ID)
		if err != nil {
			return nil, err
		}
		for _, s := range svcs {
			serviceCount[s.ID]++
			serviceSum[s.ID] += s.PriceCents
			revenueCents += s.PriceCents
		}
	}

	byService := make([]ServiceStat, 0, len(serviceCount))
	for id, count := range serviceCount {
		name := ""
		if s, _ := serviceRepo.GetByID(ctx, id); s != nil {
			name = s.Name
		}
		byService = append(byService, ServiceStat{ServiceName: name, Count: count, SumCents: serviceSum[id]})
	}
	sort.Slice(byService, func(i, j int) bool { return byService[j].SumCents < byService[i].SumCents })

	var topClients []ClientStat
	for clientID, count := range clientCount {
		client, err := clientRepo.GetByID(ctx, clientID)
		if err != nil || client == nil {
			continue
		}
		topClients = append(topClients, ClientStat{Client: client, Count: count})
	}
	sort.Slice(topClients, func(i, j int) bool { return topClients[j].Count < topClients[i].Count })
	if len(topClients) > 10 {
		topClients = topClients[:10]
	}

	return &StatsResult{
		RevenueCents: revenueCents,
		VisitCount:   len(activeVisits),
		ByService:    byService,
		TopClients:   topClients,
	}, nil
}
