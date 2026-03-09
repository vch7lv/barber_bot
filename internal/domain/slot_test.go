package domain

import (
	"testing"
)

func TestVisitDurationMinutes(t *testing.T) {
	tests := []struct {
		name     string
		services []*Service
		want     int
	}{
		{"empty", nil, 0},
		{"one", []*Service{{DurationMin: 30}}, 30},
		{"two", []*Service{{DurationMin: 30}, {DurationMin: 45}}, 75},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := VisitDurationMinutes(tt.services); got != tt.want {
				t.Errorf("VisitDurationMinutes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSlotsOverlap(t *testing.T) {
	tests := []struct {
		start1, dur1, start2, dur2 int64
		want                       bool
	}{
		{0, 60, 60, 60, false},
		{0, 60, 30, 60, true},
		{30, 30, 0, 60, true},
		{0, 30, 30, 30, false},
		{10, 20, 15, 10, true},
	}
	for i, tt := range tests {
		got := SlotsOverlap(tt.start1, tt.dur1, tt.start2, tt.dur2)
		if got != tt.want {
			t.Errorf("#%d SlotsOverlap(%d,%d,%d,%d) = %v, want %v", i, tt.start1, tt.dur1, tt.start2, tt.dur2, got, tt.want)
		}
	}
}

func TestSlotFits(t *testing.T) {
	// Start/Duration в секундах (unix). 30 мин = 1800 с, 60 мин = 3600 с.
	// Занято: [100, 1900) и [2000, 5600).
	occupied := []struct{ Start, Duration int64 }{
		{100, 30 * 60},
		{2000, 60 * 60},
	}
	tests := []struct {
		slotStart   int64
		durationMin int
		want        bool
	}{
		{0, 1, true},       // [0, 60) не пересекается
		{50, 30, false},    // [50, 1850) пересекается с [100, 1900)
		{1900, 30, false},  // [1900, 3700) пересекается с [2000, 5600)
		{5600, 30, true},   // [5600, 7400) не пересекается
	}
	for _, tt := range tests {
		got := SlotFits(tt.slotStart, tt.durationMin, occupied)
		if got != tt.want {
			t.Errorf("SlotFits(%d, %d, ...) = %v, want %v", tt.slotStart, tt.durationMin, got, tt.want)
		}
	}
}
