package handlers

import (
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type taskMutationDTO struct {
	Title         string                `json:"title"`
	Description   string                `json:"description,omitempty"`
	ScheduledAt   string                `json:"scheduled_at,omitempty"`
	Status        taskdomain.Status     `json:"status,omitempty"`
	IsPeriodicity bool                  `json:"is_periodicity"`
	RepeatRule    taskdomain.RepeatRule `json:"repeat_rule,omitempty"`
}

type taskDTO struct {
	ID            int64                 `json:"id"`
	Title         string                `json:"title"`
	Description   string                `json:"description"`
	ScheduledAt   string                `json:"scheduled_at"`
	Status        taskdomain.Status     `json:"status"`
	IsPeriodicity bool                  `json:"is_periodicity"`
	RepeatRule    taskdomain.RepeatRule `json:"repeat_rule,omitempty"`
	CreatedAt     time.Time             `json:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
}

func newTaskDTO(task *taskdomain.Task) taskDTO {
	return taskDTO{
		ID:            task.ID,
		Title:         task.Title,
		Description:   task.Description,
		ScheduledAt:   task.ScheduledAt.Format(time.RFC3339),
		Status:        task.Status,
		IsPeriodicity: task.IsPeriodicity,
		RepeatRule:    task.RepeatRule,
		CreatedAt:     task.CreatedAt,
		UpdatedAt:     task.UpdatedAt,
	}
}
