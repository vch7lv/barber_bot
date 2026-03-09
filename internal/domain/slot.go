package domain

// VisitDurationMinutes возвращает суммарную длительность визита по списку услуг (в минутах).
func VisitDurationMinutes(services []*Service) int {
	var sum int
	for _, s := range services {
		sum += s.DurationMin
	}
	return sum
}

// SlotsOverlap возвращает true, если два интервала [start1, start1+dur1) и [start2, start2+dur2) пересекаются.
// Все start и duration в одних единицах (секунды для unix).
func SlotsOverlap(start1, dur1, start2, dur2 int64) bool {
	end1 := start1 + dur1
	end2 := start2 + dur2
	return start1 < end2 && start2 < end1
}

// SlotFits проверяет, что интервал [slotStart, slotStart+durationMin) не пересекается ни с одним из занятых.
// slotStart — unix в секундах; occupied — пары (Start в секундах, Duration в секундах).
func SlotFits(slotStart int64, durationMin int, occupied []struct{ Start, Duration int64 }) bool {
	slotDurSec := int64(durationMin) * 60
	for _, o := range occupied {
		if SlotsOverlap(slotStart, slotDurSec, o.Start, o.Duration) {
			return false
		}
	}
	return true
}
