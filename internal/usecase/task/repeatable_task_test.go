package task

import (
	"errors"
	"testing"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

// dt creates a time.Time at 09:00 UTC — used to verify time-of-day is preserved.
func dt(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 9, 0, 0, 0, time.UTC)
}

// ───────────────────────────────────────────────────────────────────────────────
// nextMonthlyDate
// ───────────────────────────────────────────────────────────────────────────────

func TestNextMonthlyDate(t *testing.T) {
	tests := []struct {
		name    string
		current time.Time
		days    []int
		want    time.Time
	}{
		{
			name:    "day later in same month",
			current: dt(2026, time.January, 10),
			days:    []int{15},
			want:    dt(2026, time.January, 15),
		},
		{
			name:    "multiple days picks nearest after current",
			current: dt(2026, time.January, 10),
			days:    []int{5, 15, 25},
			want:    dt(2026, time.January, 15),
		},
		{
			name:    "current day equals day in list — goes to next month",
			current: dt(2026, time.January, 15),
			days:    []int{15},
			want:    dt(2026, time.February, 15),
		},
		{
			name:    "all days before current — goes to next month first day",
			current: dt(2026, time.January, 25),
			days:    []int{5, 15},
			want:    dt(2026, time.February, 5),
		},
		{
			name:    "december wraps to next year",
			current: dt(2026, time.December, 20),
			days:    []int{15},
			want:    dt(2027, time.January, 15),
		},
		{
			name:    "day 30 skips february (28 days)",
			current: dt(2026, time.January, 30),
			days:    []int{30},
			want:    dt(2026, time.March, 30),
		},
		{
			name:    "day 29 skips february in non-leap year",
			current: dt(2026, time.January, 29),
			days:    []int{29},
			want:    dt(2026, time.March, 29),
		},
		{
			name:    "day 29 lands in february of leap year",
			current: dt(2024, time.January, 29),
			days:    []int{29},
			want:    dt(2024, time.February, 29),
		},
		{
			name:    "day 30 valid in november (30-day month)",
			current: dt(2026, time.October, 31),
			days:    []int{30},
			want:    dt(2026, time.November, 30),
		},
		{
			name:    "multiple days — last one in same month wins when others passed",
			current: dt(2026, time.January, 20),
			days:    []int{5, 15, 25},
			want:    dt(2026, time.January, 25),
		},
		{
			name:    "time of day is preserved",
			current: time.Date(2026, time.January, 10, 14, 30, 45, 0, time.UTC),
			days:    []int{15},
			want:    time.Date(2026, time.January, 15, 14, 30, 45, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nextMonthlyDate(tt.current, tt.days)
			if !got.Equal(tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// ───────────────────────────────────────────────────────────────────────────────
// nextWeeklyDate
// Reference: April 13 2026 = Monday (weekday 1)
// ───────────────────────────────────────────────────────────────────────────────

func TestNextWeeklyDate(t *testing.T) {
	mon := dt(2026, time.April, 13) // weekday 1
	fri := dt(2026, time.April, 17) // weekday 5
	sat := dt(2026, time.April, 18) // weekday 6
	sun := dt(2026, time.April, 19) // weekday 0

	tests := []struct {
		name     string
		current  time.Time
		weekdays []int
		want     time.Time
	}{
		{
			name:     "next day matches",
			current:  mon,
			weekdays: []int{2}, // Tuesday
			want:     dt(2026, time.April, 14),
		},
		{
			name:     "target is several days ahead",
			current:  mon,
			weekdays: []int{5}, // Friday
			want:     fri,
		},
		{
			name:     "same weekday as current — next week",
			current:  mon,
			weekdays: []int{1}, // Monday again
			want:     dt(2026, time.April, 20),
		},
		{
			name:     "saturday, want monday — wraps over sunday",
			current:  sat,
			weekdays: []int{1},
			want:     dt(2026, time.April, 20),
		},
		{
			name:     "sunday, want sunday — full week ahead",
			current:  sun,
			weekdays: []int{0},
			want:     dt(2026, time.April, 26),
		},
		{
			name:     "multiple weekdays — picks nearest",
			current:  mon,
			weekdays: []int{3, 5}, // Wed and Fri
			want:     dt(2026, time.April, 15),
		},
		{
			name:     "saturday, multiple weekdays — picks sunday over monday",
			current:  sat,
			weekdays: []int{0, 1},
			want:     sun,
		},
		{
			name:     "friday, want monday — crosses weekend",
			current:  fri,
			weekdays: []int{1},
			want:     dt(2026, time.April, 20),
		},
		{
			name:     "crosses month boundary",
			current:  dt(2026, time.April, 29), // Wednesday
			weekdays: []int{5},                  // Friday
			want:     dt(2026, time.May, 1),
		},
		{
			name:     "time of day is preserved",
			current:  time.Date(2026, time.April, 13, 14, 30, 45, 0, time.UTC), // Monday
			weekdays: []int{3},                                                   // Wednesday
			want:     time.Date(2026, time.April, 15, 14, 30, 45, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nextWeeklyDate(tt.current, tt.weekdays)
			if !got.Equal(tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// ───────────────────────────────────────────────────────────────────────────────
// nextEvenOddDate
// ───────────────────────────────────────────────────────────────────────────────

func TestNextEvenOddDate(t *testing.T) {
	tests := []struct {
		name    string
		current time.Time
		parity  taskdomain.Parity
		want    time.Time
	}{
		{
			name:    "odd day, want even — next day",
			current: dt(2026, time.April, 1),
			parity:  taskdomain.EvenParity,
			want:    dt(2026, time.April, 2),
		},
		{
			name:    "even day, want odd — next day",
			current: dt(2026, time.April, 2),
			parity:  taskdomain.OddParity,
			want:    dt(2026, time.April, 3),
		},
		{
			name:    "even day, want even — skips odd",
			current: dt(2026, time.April, 2),
			parity:  taskdomain.EvenParity,
			want:    dt(2026, time.April, 4),
		},
		{
			name:    "odd day, want odd — skips even",
			current: dt(2026, time.April, 1),
			parity:  taskdomain.OddParity,
			want:    dt(2026, time.April, 3),
		},
		{
			name:    "day 29 odd, want even — day 30 same month",
			current: dt(2026, time.April, 29),
			parity:  taskdomain.EvenParity,
			want:    dt(2026, time.April, 30),
		},
		{
			name:    "day 30 even (last day april), want odd — may 1",
			current: dt(2026, time.April, 30),
			parity:  taskdomain.OddParity,
			want:    dt(2026, time.May, 1),
		},
		{
			name:    "day 30 even (last day april), want even — may 2",
			current: dt(2026, time.April, 30),
			parity:  taskdomain.EvenParity,
			want:    dt(2026, time.May, 2),
		},
		{
			name:    "december 31 odd, want odd — january 1",
			current: dt(2026, time.December, 31),
			parity:  taskdomain.OddParity,
			want:    dt(2027, time.January, 1),
		},
		{
			name:    "december 31 odd, want even — january 2",
			current: dt(2026, time.December, 31),
			parity:  taskdomain.EvenParity,
			want:    dt(2027, time.January, 2),
		},
		{
			name:    "february 28 non-leap, want odd — march 1",
			current: dt(2026, time.February, 28),
			parity:  taskdomain.OddParity,
			want:    dt(2026, time.March, 1),
		},
		{
			name:    "february 28 non-leap, want even — march 2",
			current: dt(2026, time.February, 28),
			parity:  taskdomain.EvenParity,
			want:    dt(2026, time.March, 2),
		},
		{
			name:    "time of day is preserved",
			current: time.Date(2026, time.April, 1, 14, 30, 45, 0, time.UTC),
			parity:  taskdomain.EvenParity,
			want:    time.Date(2026, time.April, 2, 14, 30, 45, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nextEvenOddDate(tt.current, tt.parity)
			if !got.Equal(tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// ───────────────────────────────────────────────────────────────────────────────
// validateRepeatRule — deduplication
// ───────────────────────────────────────────────────────────────────────────────

func TestValidateRepeatRuleDedup(t *testing.T) {
	tests := []struct {
		name      string
		rule      taskdomain.RepeatRule
		wantDays  []int
		wantDates []string
	}{
		{
			name: "monthly: duplicates removed",
			rule: taskdomain.RepeatRule{
				PeriodicityType: taskdomain.PeriodMonthly,
				Days:            []int{5, 5, 15, 15, 20},
			},
			wantDays: []int{5, 15, 20},
		},
		{
			name: "monthly: unsorted with duplicates — sorted and deduped",
			rule: taskdomain.RepeatRule{
				PeriodicityType: taskdomain.PeriodMonthly,
				Days:            []int{20, 5, 5, 15},
			},
			wantDays: []int{5, 15, 20},
		},
		{
			name: "monthly: no duplicates — unchanged",
			rule: taskdomain.RepeatRule{
				PeriodicityType: taskdomain.PeriodMonthly,
				Days:            []int{1, 15, 30},
			},
			wantDays: []int{1, 15, 30},
		},
		{
			name: "monthly: single element — unchanged",
			rule: taskdomain.RepeatRule{
				PeriodicityType: taskdomain.PeriodMonthly,
				Days:            []int{10},
			},
			wantDays: []int{10},
		},
		{
			name: "weekly: duplicates removed",
			rule: taskdomain.RepeatRule{
				PeriodicityType: taskdomain.PeriodWeekly,
				Weekdays:        []int{1, 1, 3, 3, 5},
			},
			wantDays: []int{1, 3, 5},
		},
		{
			name: "spec_dates: duplicate dates removed",
			rule: taskdomain.RepeatRule{
				PeriodicityType: taskdomain.PeriodSpecDates,
				Dates:           []string{"2026-06-01", "2026-05-01", "2026-05-01"},
			},
			wantDates: []string{"2026-05-01", "2026-06-01"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleaned, err := validateRepeatRule(tt.rule)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantDays != nil {
				got := cleaned.Days
				if len(got) == 0 {
					got = cleaned.Weekdays
				}
				if !equalInts(got, tt.wantDays) {
					t.Errorf("got %v, want %v", got, tt.wantDays)
				}
			}

			if tt.wantDates != nil {
				if !equalStrings(cleaned.Dates, tt.wantDates) {
					t.Errorf("got %v, want %v", cleaned.Dates, tt.wantDates)
				}
			}
		})
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ───────────────────────────────────────────────────────────────────────────────
// computeNextScheduledAt
// ───────────────────────────────────────────────────────────────────────────────

func TestComputeNextScheduledAt(t *testing.T) {
	// April 13 2026 = Monday
	current := dt(2026, time.April, 13)

	tests := []struct {
		name    string
		current time.Time
		rule    taskdomain.RepeatRule
		wantNil bool
		want    time.Time
		wantErr bool
	}{
		{
			name:    "daily dispatches with correct interval",
			current: current,
			rule:    taskdomain.RepeatRule{PeriodicityType: taskdomain.PeriodDaily, Interval: 3},
			want:    dt(2026, time.April, 16),
		},
		{
			name:    "weekly dispatches correctly",
			current: current, // Monday
			rule:    taskdomain.RepeatRule{PeriodicityType: taskdomain.PeriodWeekly, Weekdays: []int{3}},
			want:    dt(2026, time.April, 15), // Wednesday
		},
		{
			name:    "monthly dispatches correctly",
			current: current, // April 13
			rule:    taskdomain.RepeatRule{PeriodicityType: taskdomain.PeriodMonthly, Days: []int{20}},
			want:    dt(2026, time.April, 20),
		},
		{
			name:    "even_odd dispatches correctly",
			current: dt(2026, time.April, 14), // even day
			rule:    taskdomain.RepeatRule{PeriodicityType: taskdomain.PeriodEvenOdd, Parity: taskdomain.OddParity},
			want:    dt(2026, time.April, 15),
		},
		{
			name:    "spec_dates returns nil — no next task",
			current: current,
			rule:    taskdomain.RepeatRule{PeriodicityType: taskdomain.PeriodSpecDates},
			wantNil: true,
		},
		{
			name:    "unknown periodicity type returns error",
			current: current,
			rule:    taskdomain.RepeatRule{PeriodicityType: "unknown"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := computeNextScheduledAt(tt.current, tt.rule)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if !errors.Is(err, ErrInvalidInput) {
					t.Errorf("expected ErrInvalidInput, got %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %v", got)
				}
				return
			}

			if !got.Equal(tt.want) {
				t.Errorf("got %v, want %v", *got, tt.want)
			}
		})
	}
}
