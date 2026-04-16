package task

import (
	"context"
	"fmt"
	"sort"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

// createNewRepeatableTask calculate next date by RepeatRule and create new task by rule
// return nil if not has next task
func (s *Service) createNewRepeatableTask(ctx context.Context, task *taskdomain.Task) error {
	nextAt, err := computeNextScheduledAt(task.ScheduledAt, task.RepeatRule)
	if err != nil {
		return err
	}
	if nextAt == nil {
		return nil
	}

	now := s.now()
	newTask := &taskdomain.Task{
		Title:         task.Title,
		Description:   task.Description,
		IsPeriodicity: task.IsPeriodicity,
		ScheduledAt:   *nextAt,
		RepeatRule:    task.RepeatRule,
		Status:        taskdomain.StatusNew,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	_, err = s.repo.Create(ctx, newTask)
	return err
}

// computeNextScheduledAt calculates the next start time according to the rule.
func computeNextScheduledAt(current time.Time, rule taskdomain.RepeatRule) (*time.Time, error) {
	switch rule.PeriodicityType {
	case taskdomain.PeriodDaily:
		next := current.AddDate(0, 0, rule.Interval)
		return &next, nil

	case taskdomain.PeriodMonthly:
		next := nextMonthlyDate(current, rule.Days)
		return &next, nil

	case taskdomain.PeriodWeekly:
		next := nextWeeklyDate(current, rule.Weekdays)
		return &next, nil

	case taskdomain.PeriodSpecDates:
		return nil, nil

	case taskdomain.PeriodEvenOdd:
		next := nextEvenOddDate(current, rule.Parity)
		return &next, nil

	default:
		return nil, fmt.Errorf("%w: unknown periodicity type: %s", ErrInvalidInput, rule.PeriodicityType)
	}
}

// nextMonthlyDate finds the next month's date from the days list.
// if there are no suitable dates in the current month, it moves to the following months.
// dates that do not exist in a particular month (such as February 31) are skipped.
func nextMonthlyDate(current time.Time, days []int) time.Time {
	h, m, sec := current.Clock()
	y, mo := current.Year(), current.Month()
	loc := current.Location()

	// looking for a suitable day in the current month
	for _, d := range days {
		if d > current.Day() {
			candidate := time.Date(current.Year(), current.Month(), d, h, m, sec, 0, loc)
			if candidate.Day() == d && candidate.Month() == current.Month() {
				return candidate
			}
		}
	}

	// looking in the next months
	for i := 0; i < 3; i++ {
		mo++
		if mo > 12 {
			mo = 1
			y++
		}
		for _, d := range days {
			candidate := time.Date(y, mo, d, h, m, sec, 0, loc)
			if candidate.Day() == d && candidate.Month() == mo {
				return candidate
			}
		}
	}

	// should not be execute
	return current.AddDate(0, 1, 0)
}

// nextWeeklyDate finds the nearest next day matching one of the given weekdays.
// weekdays must be sorted and contain values 0 (Sunday) through 6 (Saturday).
func nextWeeklyDate(current time.Time, weekdays []int) time.Time {
	h, m, sec := current.Clock()
	loc := current.Location()

	next := current.AddDate(0, 0, 1)
	for range 7 {
		wd := int(next.Weekday())
		for _, w := range weekdays {
			if w == wd {
				return time.Date(next.Year(), next.Month(), next.Day(), h, m, sec, 0, loc)
			}
		}
		next = next.AddDate(0, 0, 1)
	}

	// unreachable with valid input (weekdays is non-empty, 7 days cover all weekdays)
	return current.AddDate(0, 0, 7)
}

// nextEvenOddDate find nearest next day with needed parity.
func nextEvenOddDate(current time.Time, parity taskdomain.Parity) time.Time {
	h, m, sec := current.Clock()
	loc := current.Location()

	next := current.AddDate(0, 0, 1)
	for {
		isEven := next.Day()%2 == 0
		matches := (parity == taskdomain.EvenParity && isEven) || (parity == taskdomain.OddParity && !isEven)
		if matches {
			return time.Date(next.Year(), next.Month(), next.Day(), h, m, sec, 0, loc)
		}
		next = next.AddDate(0, 0, 1)
	}
}

// createAllSpecDateTasks creates tasks for all dates in spec_dates,
// except for the first one (which is already created as the main task).
func (s *Service) createAllSpecDateTasks(ctx context.Context, task *taskdomain.Task) error {
	firstDateStr := task.ScheduledAt.Format("2006-01-02")
	h, m, sec := task.ScheduledAt.Clock()
	loc := task.ScheduledAt.Location()
	now := s.now()

	var tasks []*taskdomain.Task
	for _, dateStr := range task.RepeatRule.Dates {
		if dateStr <= firstDateStr {
			continue
		}

		d, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return fmt.Errorf("invalid spec_date %q: %w", dateStr, err)
		}

		tasks = append(tasks, &taskdomain.Task{
			Title:         task.Title,
			Description:   task.Description,
			IsPeriodicity: task.IsPeriodicity,
			ScheduledAt:   time.Date(d.Year(), d.Month(), d.Day(), h, m, sec, 0, loc),
			RepeatRule:    task.RepeatRule,
			Status:        taskdomain.StatusNew,
			CreatedAt:     now,
			UpdatedAt:     now,
		})
	}

	if len(tasks) == 0 {
		return nil
	}

	return s.repo.CreateBatch(ctx, tasks)
}

func validateRepeatRule(rule taskdomain.RepeatRule) error {
	if !rule.PeriodicityType.Valid() {
		return fmt.Errorf("%w: invalid periodicity type", ErrInvalidInput)
	}

	switch rule.PeriodicityType {
	case taskdomain.PeriodDaily:
		if rule.Interval <= 0 {
			return fmt.Errorf("%w: daily interval must be positive", ErrInvalidInput)
		}
	case taskdomain.PeriodMonthly:
		if len(rule.Days) == 0 {
			return fmt.Errorf("%w: monthly rule requires at least one day", ErrInvalidInput)
		}
		for _, d := range rule.Days {
			if d < 1 || d > 30 {
				return fmt.Errorf("%w: monthly day must be between 1 and 30", ErrInvalidInput)
			}
		}
		sort.Ints(rule.Days)
	case taskdomain.PeriodSpecDates:
		if len(rule.Dates) == 0 {
			return fmt.Errorf("%w: spec_dates rule requires at least one date", ErrInvalidInput)
		}
		for _, d := range rule.Dates {
			if _, err := time.Parse("2006-01-02", d); err != nil {
				return fmt.Errorf("%w: spec_dates date must be YYYY-MM-DD, got %q", ErrInvalidInput, d)
			}
		}
		sort.Strings(rule.Dates)
	case taskdomain.PeriodEvenOdd:
		if !rule.Parity.Valid() {
			return fmt.Errorf("%w: even_odd rule requires valid parity (even|odd)", ErrInvalidInput)
		}
	
	case taskdomain.PeriodWeekly:
		if len(rule.Weekdays) == 0 {
			return fmt.Errorf("%w: weekly rule requires at least one weekday (0=Sun, 6=Sat)", ErrInvalidInput)
		}
		for _, d := range rule.Weekdays {
			if d < 0 || d > 6 {
				return fmt.Errorf("%w: weekday must be between 0 and 6", ErrInvalidInput)
			}
		}
		sort.Ints(rule.Weekdays)

	}

	return nil
}
