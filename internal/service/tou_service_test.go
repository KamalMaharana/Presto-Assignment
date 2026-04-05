package service

import (
	"testing"
	"time"
)

func TestParseDate_PreservesCalendarDateAcrossTimezones(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		timezone string
		want     time.Time
	}{
		{
			name:     "utc timezone",
			input:    "2026-04-15",
			timezone: "UTC",
			want:     time.Date(2026, time.April, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "positive offset timezone",
			input:    "2026-04-15",
			timezone: "Asia/Kolkata",
			want:     time.Date(2026, time.April, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "far positive offset timezone",
			input:    "2026-01-01",
			timezone: "Pacific/Auckland",
			want:     time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "negative offset timezone",
			input:    "2026-11-05",
			timezone: "America/Los_Angeles",
			want:     time.Date(2026, time.November, 5, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseDate(tt.input, tt.timezone)
			if err != nil {
				t.Fatalf("parseDate returned error: %v", err)
			}
			if !got.Equal(tt.want) {
				t.Fatalf("expected %s, got %s", tt.want.Format(time.RFC3339), got.Format(time.RFC3339))
			}
			if got.Location() != time.UTC {
				t.Fatalf("expected UTC location, got %s", got.Location().String())
			}
		})
	}
}

func TestParseDate_InvalidInputs(t *testing.T) {
	t.Parallel()

	if _, err := parseDate("2026-99-99", "UTC"); err == nil {
		t.Fatal("expected error for invalid date")
	}

	if _, err := parseDate("2026-04-15", "Mars/Olympus_Mons"); err == nil {
		t.Fatal("expected error for invalid timezone")
	}
}
