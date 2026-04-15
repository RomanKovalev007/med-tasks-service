package task

import "time"

type Status string

type PeriodicityType string

type Parity string

const (
	StatusNew        Status = "new"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
	StatusCanceled   Status = "canceled"

	PeriodDaily     PeriodicityType = "daily"
	PeriodMothly    PeriodicityType = "monthly"
	PeriodSpecDates PeriodicityType = "spec_dates"
	PeriodEvenOdd   PeriodicityType = "even_odd"

	EvenParity Parity = "even"
	OddParity  Parity = "odd"
)

type RepeatRule struct {
	PeriodicityType PeriodicityType `json:"periodicity_type"`
	Interval        int             `json:"interval,omitempty"` // for daily
	Days            []int           `json:"days,omitempty"`     // for monthly (month dates 1-30)
	Dates           []string        `json:"dates,omitempty"`    // for spec_dates (YYYY-MM-DD)
	Parity          Parity          `json:"parity,omitempty"`   // "even" / "odd"
}

type Task struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`

	ScheduledAt   time.Time  `json:"scheduled_at,omitempty"`
	IsPeriodicity bool       `json:"is_periodicity"`
	RepeatRule    RepeatRule `json:"repeat_rule,omitempty"`

	Status    Status    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (s Status) Valid() bool {
	switch s {
	case StatusNew, StatusInProgress, StatusDone, StatusCanceled:
		return true
	default:
		return false
	}
}

func (t PeriodicityType) Valid() bool {
	switch t {
	case PeriodDaily, PeriodMothly, PeriodEvenOdd, PeriodSpecDates:
		return true
	default:
		return false
	}
}

func (t Parity) Valid() bool {
	switch t {
	case OddParity, EvenParity:
		return true
	default:
		return false
	}
}
