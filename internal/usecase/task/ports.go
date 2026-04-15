package task

import (
	"context"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Repository interface {
	Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error)
	CreateBatch(ctx context.Context, tasks []*taskdomain.Task) error
	GetByID(ctx context.Context, id int64) (*taskdomain.Task, error)
	Update(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error)
	UpdateStatus(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]taskdomain.Task, error)
}

type Usecase interface {
	Create(ctx context.Context, input CreateInput) (*taskdomain.Task, error)
	GetByID(ctx context.Context, id int64) (*taskdomain.Task, error)
	Update(ctx context.Context, id int64, input UpdateInput) (*taskdomain.Task, error)
	UpdateStatus(ctx context.Context, id int64, input UpdateStatusInput) (*taskdomain.Task, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]taskdomain.Task, error)
}

type CreateInput struct {
	Title         string
	Description   string
	Status        taskdomain.Status
	ScheduledAt   string
	IsPeriodicity bool
	RepeatRule    taskdomain.RepeatRule
}

type UpdateInput struct {
	Title         string
	Description   string
	ScheduledAt   string
	IsPeriodicity bool
	RepeatRule    taskdomain.RepeatRule
}

type UpdateStatusInput struct {
	Status      taskdomain.Status
}
