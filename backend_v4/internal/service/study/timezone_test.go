package study

import (
	"testing"
	"time"
)

func TestDayStart(t *testing.T) {
	tests := []struct {
		name     string
		now      time.Time
		tz       string
		wantHour int
		wantDay  int
	}{
		{
			name:     "UTC midnight",
			now:      time.Date(2024, 2, 15, 12, 30, 0, 0, time.UTC),
			tz:       "UTC",
			wantHour: 0,
			wantDay:  15,
		},
		{
			name:     "America/New_York",
			now:      time.Date(2024, 2, 15, 12, 30, 0, 0, time.UTC),
			tz:       "America/New_York",
			wantHour: 5, // EST is UTC-5, so midnight EST = 5:00 UTC
			wantDay:  15,
		},
		{
			name:     "Asia/Tokyo",
			now:      time.Date(2024, 2, 15, 12, 30, 0, 0, time.UTC),
			tz:       "Asia/Tokyo",
			wantHour: 15, // JST is UTC+9, so midnight JST = 15:00 prev day UTC
			wantDay:  14, // Should be previous day
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc := ParseTimezone(tt.tz)
			result := DayStart(tt.now, loc)

			if result.Hour() != tt.wantHour {
				t.Errorf("DayStart() hour = %d, want %d", result.Hour(), tt.wantHour)
			}
			if tt.wantDay > 0 && result.Day() != tt.wantDay {
				t.Errorf("DayStart() day = %d, want %d", result.Day(), tt.wantDay)
			}
			if result.Minute() != 0 || result.Second() != 0 {
				t.Errorf("DayStart() should be at 00:00:00, got %02d:%02d:%02d",
					result.Hour(), result.Minute(), result.Second())
			}
		})
	}
}

func TestNextDayStart(t *testing.T) {
	now := time.Date(2024, 2, 15, 12, 30, 0, 0, time.UTC)
	loc := time.UTC

	next := NextDayStart(now, loc)
	day := DayStart(now, loc)

	diff := next.Sub(day)
	if diff != 24*time.Hour {
		t.Errorf("NextDayStart should be 24h after DayStart, got %v", diff)
	}
}

func TestParseTimezone(t *testing.T) {
	tests := []struct {
		name  string
		tz    string
		valid bool
	}{
		{"valid UTC", "UTC", true},
		{"valid New York", "America/New_York", true},
		{"valid Tokyo", "Asia/Tokyo", true},
		{"invalid", "Invalid/Timezone", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc := ParseTimezone(tt.tz)
			if tt.valid && loc == time.UTC && tt.tz != "UTC" {
				t.Error("Expected non-UTC location for valid timezone")
			}
			if !tt.valid && loc != time.UTC {
				t.Error("Expected UTC fallback for invalid timezone")
			}
		})
	}
}
