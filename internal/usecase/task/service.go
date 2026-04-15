package task

import (
	"context"
	"fmt"
	"strings"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*taskdomain.Task, error) {
	model, err := validateCreateInput(input)
	if err != nil {
		return nil, err
	}
	now := s.now()
	model.CreatedAt = now
	model.UpdatedAt = now

	if !model.IsPeriodicity || model.RepeatRule.PeriodicityType != taskdomain.PeriodSpecDates {
		return s.repo.Create(ctx, &model)
	}

	var created *taskdomain.Task
	err = s.repo.WithinTransaction(ctx, 
		func(ctx context.Context) error {
			var err error
			created, err = s.repo.Create(ctx, &model)
			if err != nil {
				return err
			}
			return s.createAllSpecDateTasks(ctx, created)
		},
	)

	return created, err
}

func (s *Service) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	model, err := validateUpdateInput(input)
	if err != nil {
		return nil, err
	}

	model.UpdatedAt = s.now()

	updated, err := s.repo.Update(ctx, &model)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *Service) UpdateStatus(ctx context.Context, id int64, input UpdateStatusInput) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	if !input.Status.Valid(){
		return nil, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}

	
	if input.Status == taskdomain.StatusDone || input.Status == taskdomain.StatusCanceled {
		var updated *taskdomain.Task
		err := s.repo.WithinTransaction(ctx, 
			func(ctx context.Context) error {
				model := &taskdomain.Task{ID: id, Status: input.Status, UpdatedAt: s.now()}
				var err error
				updated, err = s.repo.UpdateStatus(ctx, model)
				if err != nil {
					return fmt.Errorf("update status: %w", err)
				}

				if updated.IsPeriodicity {
					if err := s.createNewRepeatableTask(ctx, updated); err != nil {
						return fmt.Errorf("create next repeatable task: %w", err)
					}
				}
				return nil
			},
		)

		return updated, err
	}

	model := &taskdomain.Task{
		ID:        id,
		Status:    input.Status,
		UpdatedAt: s.now(),
	}
	return s.repo.UpdateStatus(ctx, model)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]taskdomain.Task, error) {
	return s.repo.List(ctx)
}

func validateCreateInput(input CreateInput) (taskdomain.Task, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return taskdomain.Task{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	scheduledAt, err := time.Parse(time.RFC3339, input.ScheduledAt)
	if err != nil {
		return taskdomain.Task{}, fmt.Errorf("%w: scheduled_at must be RFC3339 (e.g. 2006-01-02T15:04:05Z)", ErrInvalidInput)
	}

	if input.IsPeriodicity {
		if err := validateRepeatRule(input.RepeatRule); err != nil {
			return taskdomain.Task{}, err
		}
	}

	if input.Status == "" {
		input.Status = taskdomain.StatusNew
	}

	if !input.Status.Valid() {
		return taskdomain.Task{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}

	return taskdomain.Task{
		Title:         input.Title,
		Description:   input.Description,
		IsPeriodicity: input.IsPeriodicity,
		ScheduledAt:   scheduledAt.UTC(),
		RepeatRule:    input.RepeatRule,
		Status:        input.Status,
	}, nil
}

func validateUpdateInput(input UpdateInput) (taskdomain.Task, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return taskdomain.Task{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	scheduledAt, err := time.Parse(time.RFC3339, input.ScheduledAt)
	if err != nil {
		return taskdomain.Task{}, fmt.Errorf("%w: scheduled_at must be RFC3339 (e.g. 2006-01-02T15:04:05Z)", ErrInvalidInput)
	}

	if input.IsPeriodicity {
		if err := validateRepeatRule(input.RepeatRule); err != nil {
			return taskdomain.Task{}, err
		}
	}

	return taskdomain.Task{
		Title:         input.Title,
		Description:   input.Description,
		IsPeriodicity: input.IsPeriodicity,
		ScheduledAt:   scheduledAt.UTC(),
		RepeatRule:    input.RepeatRule,
	}, nil
}

