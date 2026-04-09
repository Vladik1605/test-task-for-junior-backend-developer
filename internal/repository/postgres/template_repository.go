package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type TemplateRepository struct {
	pool *pgxpool.Pool
}

func NewTemplateRepository(pool *pgxpool.Pool) *TemplateRepository {
	return &TemplateRepository{pool: pool}
}

func (r *TemplateRepository) CreateTemplate(ctx context.Context, template *taskdomain.TaskTemplate) (*taskdomain.TaskTemplate, error) {
	const query = `
		INSERT INTO task_templates (title, description, recurrence_type, recurrence_params, start_date, end_date, last_generated_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, title, description, recurrence_type, recurrence_params, start_date, end_date, last_generated_date, created_at, updated_at
	`

	row := r.pool.QueryRow(ctx, query,
		template.Title,
		template.Description,
		template.RecurrenceType,
		template.RecurrenceParams,
		template.StartDate,
		template.EndDate,
		template.LastGeneratedDate,
		template.CreatedAt,
		template.UpdatedAt,
	)

	return scanTemplate(row)
}

func (r *TemplateRepository) GetTemplateByID(ctx context.Context, id int64) (*taskdomain.TaskTemplate, error) {
	const query = `
		SELECT id, title, description, recurrence_type, recurrence_params, start_date, end_date, last_generated_date, created_at, updated_at
		FROM task_templates
		WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, query, id)
	found, err := scanTemplate(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}
		return nil, err
	}

	return found, nil
}

func (r *TemplateRepository) UpdateTemplate(ctx context.Context, template *taskdomain.TaskTemplate) (*taskdomain.TaskTemplate, error) {
	const query = `
		UPDATE task_templates
		SET title = $1,
			description = $2,
			recurrence_type = $3,
			recurrence_params = $4,
			start_date = $5,
			end_date = $6,
			last_generated_date = $7,
			updated_at = $8
		WHERE id = $9
		RETURNING id, title, description, recurrence_type, recurrence_params, start_date, end_date, last_generated_date, created_at, updated_at
	`

	row := r.pool.QueryRow(ctx, query,
		template.Title,
		template.Description,
		template.RecurrenceType,
		template.RecurrenceParams,
		template.StartDate,
		template.EndDate,
		template.LastGeneratedDate,
		template.UpdatedAt,
		template.ID,
	)

	updated, err := scanTemplate(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}
		return nil, err
	}

	return updated, nil
}

func (r *TemplateRepository) DeleteTemplate(ctx context.Context, id int64) error {
	const query = `DELETE FROM task_templates WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return taskdomain.ErrNotFound
	}

	return nil
}

func (r *TemplateRepository) ListTemplates(ctx context.Context) ([]taskdomain.TaskTemplate, error) {
	const query = `
		SELECT id, title, description, recurrence_type, recurrence_params, start_date, end_date, last_generated_date, created_at, updated_at
		FROM task_templates
		ORDER BY id DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	templates := make([]taskdomain.TaskTemplate, 0)
	for rows.Next() {
		template, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		templates = append(templates, *template)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return templates, nil
}

func (r *TemplateRepository) ListActiveTemplatesForDate(ctx context.Context, date time.Time) ([]taskdomain.TaskTemplate, error) {
	const query = `
		SELECT id, title, description, recurrence_type, recurrence_params, start_date, end_date, last_generated_date, created_at, updated_at
		FROM task_templates
		WHERE start_date <= $1
			AND (end_date IS NULL OR end_date >= $1)
			AND (last_generated_date IS NULL OR last_generated_date < $1)
	`

	rows, err := r.pool.Query(ctx, query, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	templates := make([]taskdomain.TaskTemplate, 0)
	for rows.Next() {
		template, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		templates = append(templates, *template)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return templates, nil
}

type templateScanner interface {
	Scan(dest ...any) error
}

func scanTemplate(scanner templateScanner) (*taskdomain.TaskTemplate, error) {
	var (
		template taskdomain.TaskTemplate
		recType  string
	)

	err := scanner.Scan(
		&template.ID,
		&template.Title,
		&template.Description,
		&recType,
		&template.RecurrenceParams,
		&template.StartDate,
		&template.EndDate,
		&template.LastGeneratedDate,
		&template.CreatedAt,
		&template.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	template.RecurrenceType = taskdomain.RecurrenceType(recType)
	return &template, nil
}
