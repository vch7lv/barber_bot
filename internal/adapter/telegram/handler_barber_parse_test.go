package telegram

import "testing"

func TestParseVisitIDFromBarberInput(t *testing.T) {
	tests := []struct {
		in   string
		want int64
		ok   bool
	}{
		{"42", 42, true},
		{"#42", 42, true},
		{"  #7 ", 7, true},
		{"визит #99", 99, true},
		{"• 31.03 15:04 — визит #5 Иван", 5, true},
		{"", 0, false},
		{"abc", 0, false},
	}
	for _, tt := range tests {
		got, err := parseVisitIDFromBarberInput(tt.in)
		if tt.ok {
			if err != nil || got != tt.want {
				t.Errorf("parseVisitIDFromBarberInput(%q) = %d, %v; want %d, nil", tt.in, got, err, tt.want)
			}
		} else {
			if err == nil && got > 0 {
				t.Errorf("parseVisitIDFromBarberInput(%q) = %d, want error", tt.in, got)
			}
		}
	}
}
