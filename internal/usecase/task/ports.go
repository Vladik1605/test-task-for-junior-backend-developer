package task

import (
	"context"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Repository interface {
	Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error)
	GetByID(ctx context.Context, id int64) (*taskdomain.Task, error)
	Update(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]taskdomain.Task, error)
}

type TemplateRepository interface {
	CreateTemplate(ctx context.Context, template *taskdomain.TaskTemplate) (*taskdomain.TaskTemplate, error)
	GetTemplateByID(ctx context.Context, id int64) (*taskdomain.TaskTemplate, error)
	UpdateTemplate(ctx context.Context, template *taskdomain.TaskTemplate) (*taskdomain.TaskTemplate, error)
	DeleteTemplate(ctx context.Context, id int64) error
	ListTemplates(ctx context.Context) ([]taskdomain.TaskTemplate, error)
	ListActiveTemplatesForDate(ctx context.Context, date time.Time) ([]taskdomain.TaskTemplate, error)
}

type Usecase interface {
	Create(ctx context.Context, input CreateInput) (*taskdomain.Task, error)
	GetByID(ctx context.Context, id int64) (*taskdomain.Task, error)
	Update(ctx context.Context, id int64, input UpdateInput) (*taskdomain.Task, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]taskdomain.Task, error)

	// Task Template methods
	CreateTemplate(ctx context.Context, input CreateTemplateInput) (*taskdomain.TaskTemplate, error)
	GetTemplateByID(ctx context.Context, id int64) (*taskdomain.TaskTemplate, error)
	UpdateTemplate(ctx context.Context, id int64, input UpdateTemplateInput) (*taskdomain.TaskTemplate, error)
	DeleteTemplate(ctx context.Context, id int64) error
	ListTemplates(ctx context.Context) ([]taskdomain.TaskTemplate, error)

	// Generation methods
	GenerateTasksForDate(ctx context.Context, date time.Time) (int, error)

	ProcessRecurringTasks(ctx context.Context) (int, error)
	FindNextDateForTemplate(ctx context.Context, template *taskdomain.TaskTemplate, from time.Time) (time.Time, error)
	ForceAdvanceAllTasks(ctx context.Context) (int, error)
}

type CreateInput struct {
	Title       string
	Description string
	Status      taskdomain.Status
}

type UpdateInput struct {
	Title       string
	Description string
	Status      taskdomain.Status
}

type CreateTemplateInput struct {
	Title            string
	Description      string
	RecurrenceType   taskdomain.RecurrenceType
	RecurrenceParams []byte
	StartDate        time.Time
	EndDate          *time.Time
}

type UpdateTemplateInput struct {
	Title            string
	Description      string
	RecurrenceType   taskdomain.RecurrenceType
	RecurrenceParams []byte
	StartDate        time.Time
	EndDate          *time.Time
}
