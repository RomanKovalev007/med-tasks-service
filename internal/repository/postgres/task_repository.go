package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
}

type txKey struct{}

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) getQuerier(ctx context.Context) querier {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return r.pool
}

func (r *Repository) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `
		INSERT INTO tasks (title, description, status, scheduled_at, is_periodicity, repeat_rule, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, title, description, status, created_at, updated_at, scheduled_at, is_periodicity, repeat_rule
	`

	repeatRuleJSON, err := marshalRepeatRule(task.IsPeriodicity, task.RepeatRule)
	if err != nil {
		return nil, err
	}

	row := r.getQuerier(ctx).QueryRow(ctx, query,
		task.Title,
		task.Description,
		task.Status,
		task.ScheduledAt,
		task.IsPeriodicity,
		repeatRuleJSON,
		task.CreatedAt,
		task.UpdatedAt,
	)

	return scanTask(row)
}

func (r *Repository) CreateBatch(ctx context.Context, tasks []*taskdomain.Task) error {
	const query = `
		INSERT INTO tasks (title, description, status, scheduled_at, is_periodicity, repeat_rule, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	batch := &pgx.Batch{}
	for _, task := range tasks {
		repeatRuleJSON, err := marshalRepeatRule(task.IsPeriodicity, task.RepeatRule)
		if err != nil {
			return err
		}
		batch.Queue(query,
			task.Title,
			task.Description,
			task.Status,
			task.ScheduledAt,
			task.IsPeriodicity,
			repeatRuleJSON,
			task.CreatedAt,
			task.UpdatedAt,
		)
	}

	results := r.getQuerier(ctx).SendBatch(ctx, batch)
	defer results.Close()

	for range tasks {
		if _, err := results.Exec(); err != nil {
			return err
		}
	}

	return nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	const query = `
		SELECT id, title, description, status, created_at, updated_at,
		       scheduled_at, is_periodicity, repeat_rule
		FROM tasks
		WHERE id = $1
	`

	row := r.getQuerier(ctx).QueryRow(ctx, query, id)
	found, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}

		return nil, err
	}

	return found, nil
}

func (r *Repository) Update(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `
		UPDATE tasks
		SET title          = $1,
		    description    = $2,
		    updated_at     = $3,
		    scheduled_at   = $4,
		    is_periodicity = $5,
		    repeat_rule    = $6
		WHERE id = $7
		RETURNING id, title, description, status, created_at, updated_at,
		          scheduled_at, is_periodicity, repeat_rule
	`

	repeatRuleJSON, err := marshalRepeatRule(task.IsPeriodicity, task.RepeatRule)
	if err != nil {
		return nil, err
	}

	row := r.getQuerier(ctx).QueryRow(ctx, query,
		task.Title,
		task.Description,
		task.UpdatedAt,
		task.ScheduledAt,
		task.IsPeriodicity,
		repeatRuleJSON,
		task.ID,
	)

	updated, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}
		return nil, err
	}

	return updated, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `
		UPDATE tasks
		SET status = $1,
		    updated_at = $2
		WHERE id = $3
		RETURNING id, title, description, status, created_at, updated_at,
		          scheduled_at, is_periodicity, repeat_rule
	`

	row := r.getQuerier(ctx).QueryRow(ctx, query, task.Status, task.UpdatedAt, task.ID)
	updated, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}
		return nil, err
	}

	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM tasks WHERE id = $1`

	result, err := r.getQuerier(ctx).Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return taskdomain.ErrNotFound
	}

	return nil
}

func (r *Repository) List(ctx context.Context) ([]taskdomain.Task, error) {
	const query = `
		SELECT id, title, description, status, created_at, updated_at,
		       scheduled_at, is_periodicity, repeat_rule
		FROM tasks
		ORDER BY id DESC
	`

	rows, err := r.getQuerier(ctx).Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]taskdomain.Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, *task)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

type taskScanner interface {
	Scan(dest ...any) error
}

func scanTask(scanner taskScanner) (*taskdomain.Task, error) {
	var (
		task           taskdomain.Task
		status         string
		repeatRuleJSON []byte
	)

	if err := scanner.Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&status,
		&task.CreatedAt,
		&task.UpdatedAt,
		&task.ScheduledAt,
		&task.IsPeriodicity,
		&repeatRuleJSON,
	); err != nil {
		return nil, err
	}

	task.Status = taskdomain.Status(status)

	if repeatRuleJSON != nil {
		if err := json.Unmarshal(repeatRuleJSON, &task.RepeatRule); err != nil {
			return nil, err
		}
	}

	return &task, nil
}

// marshalRepeatRule serialize RepeatRule to JSON for store JSONB.
// return nil (NULL) if task is not periodicity.
func marshalRepeatRule(isPeriodic bool, rule taskdomain.RepeatRule) ([]byte, error) {
	if !isPeriodic {
		return nil, nil
	}
	return json.Marshal(rule)
}
