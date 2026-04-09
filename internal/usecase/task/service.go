package task

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Service struct {
	repo         Repository
	templateRepo TemplateRepository
	now          func() time.Time
}

func NewService(repo Repository, templateRepo TemplateRepository) *Service {
	return &Service{
		repo:         repo,
		templateRepo: templateRepo,
		now:          func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*taskdomain.Task, error) {
	normalized, err := validateCreateInput(input)
	if err != nil {
		return nil, err
	}

	model := &taskdomain.Task{
		Title:       normalized.Title,
		Description: normalized.Description,
		Status:      normalized.Status,
	}
	now := s.now()
	model.CreatedAt = now
	model.UpdatedAt = now

	created, err := s.repo.Create(ctx, model)
	if err != nil {
		return nil, err
	}

	return created, nil
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

	normalized, err := validateUpdateInput(input)
	if err != nil {
		return nil, err
	}

	model := &taskdomain.Task{
		ID:          id,
		Title:       normalized.Title,
		Description: normalized.Description,
		Status:      normalized.Status,
		UpdatedAt:   s.now(),
	}

	updated, err := s.repo.Update(ctx, model)
	if err != nil {
		return nil, err
	}

	return updated, nil
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

func validateCreateInput(input CreateInput) (CreateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return CreateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if input.Status == "" {
		input.Status = taskdomain.StatusNew
	}

	if !input.Status.Valid() {
		return CreateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}

	return input, nil
}

func validateUpdateInput(input UpdateInput) (UpdateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return UpdateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if !input.Status.Valid() {
		return UpdateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}

	return input, nil
}

// ShouldGenerateForDate checks if a task should be generated for a given date based on recurrence rule
func ShouldGenerateForDate(template *taskdomain.TaskTemplate, date time.Time) (bool, error) {
	date = date.Truncate(24 * time.Hour).UTC()
	startDate := template.StartDate.Truncate(24 * time.Hour).UTC()

	// Check if date is before start date
	if date.Before(startDate) {
		return false, nil
	}

	// Check if date is after end date (if set)
	if template.EndDate != nil {
		endDate := template.EndDate.Truncate(24 * time.Hour)
		if date.After(endDate) {
			return false, nil
		}
	}

	switch template.RecurrenceType {
	case taskdomain.RecurrenceDaily:
		var params taskdomain.DailyRecurrence
		if err := json.Unmarshal(template.RecurrenceParams, &params); err != nil {
			return false, err
		}
		if params.Interval <= 0 {
			params.Interval = 1
		}
		daysDiff := int(date.Sub(startDate).Hours() / 24)
		return daysDiff%params.Interval == 0, nil

	case taskdomain.RecurrenceMonthly:
		var params taskdomain.MonthlyRecurrence
		if err := json.Unmarshal(template.RecurrenceParams, &params); err != nil {
			return false, err
		}
		day := date.Day()
		lastDayOfMonth := time.Date(date.Year(), date.Month()+1, 0, 0, 0, 0, 0, date.Location()).Day()

		for _, d := range params.Days {
			if d == day {
				return true, nil
			}
			// Handle cases where requested day > days in month (e.g. 31st in February)
			// We use last day of month in that case
			if d > lastDayOfMonth && day == lastDayOfMonth {
				return true, nil
			}
		}
		return false, nil

	case taskdomain.RecurrenceSpecificDates:
		var params taskdomain.SpecificDatesRecurrence
		if err := json.Unmarshal(template.RecurrenceParams, &params); err != nil {
			return false, err
		}
		for _, d := range params.Dates {
			if d.Truncate(24 * time.Hour).Equal(date) {
				return true, nil
			}
		}
		return false, nil

	case taskdomain.RecurrenceEvenOdd:
		var params taskdomain.EvenOddRecurrence
		if err := json.Unmarshal(template.RecurrenceParams, &params); err != nil {
			return false, err
		}
		day := date.Day()
		if params.Type == "even" {
			return day%2 == 0, nil
		} else if params.Type == "odd" {
			return day%2 == 1, nil
		}
		return false, nil

	default:
		return false, fmt.Errorf("unknown recurrence type: %s", template.RecurrenceType)
	}
}

// CreateTemplate creates a new task template
func (s *Service) CreateTemplate(ctx context.Context, input CreateTemplateInput) (*taskdomain.TaskTemplate, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return nil, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if !input.RecurrenceType.Valid() {
		return nil, fmt.Errorf("%w: invalid recurrence type", ErrInvalidInput)
	}

	now := s.now()
	template := &taskdomain.TaskTemplate{
		Title:            input.Title,
		Description:      input.Description,
		RecurrenceType:   input.RecurrenceType,
		RecurrenceParams: input.RecurrenceParams,
		StartDate:        input.StartDate.Truncate(24 * time.Hour),
		EndDate:          input.EndDate,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	created, err := s.templateRepo.CreateTemplate(ctx, template)
	if err != nil {
		return nil, err
	}

	// Generate tasks for next 30 days when template is created
	_, err = s.GenerateTasksForTemplate(ctx, created.ID, 30)
	if err != nil {
		return nil, err
	}

	return created, nil
}

// GetTemplateByID gets a template by ID
func (s *Service) GetTemplateByID(ctx context.Context, id int64) (*taskdomain.TaskTemplate, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}
	return s.templateRepo.GetTemplateByID(ctx, id)
}

// UpdateTemplate updates an existing template
func (s *Service) UpdateTemplate(ctx context.Context, id int64, input UpdateTemplateInput) (*taskdomain.TaskTemplate, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return nil, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if !input.RecurrenceType.Valid() {
		return nil, fmt.Errorf("%w: invalid recurrence type", ErrInvalidInput)
	}

	template := &taskdomain.TaskTemplate{
		ID:               id,
		Title:            input.Title,
		Description:      input.Description,
		RecurrenceType:   input.RecurrenceType,
		RecurrenceParams: input.RecurrenceParams,
		StartDate:        input.StartDate.Truncate(24 * time.Hour),
		EndDate:          input.EndDate,
		UpdatedAt:        s.now(),
	}

	updated, err := s.templateRepo.UpdateTemplate(ctx, template)
	if err != nil {
		return nil, err
	}

	// Regenerate tasks for next 30 days when template is updated
	_, err = s.GenerateTasksForTemplate(ctx, updated.ID, 30)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

// DeleteTemplate deletes a template
func (s *Service) DeleteTemplate(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}
	return s.templateRepo.DeleteTemplate(ctx, id)
}

// ListTemplates lists all templates
func (s *Service) ListTemplates(ctx context.Context) ([]taskdomain.TaskTemplate, error) {
	return s.templateRepo.ListTemplates(ctx)
}

// GenerateTasksForDate generates tasks for all active templates for the given date
func (s *Service) GenerateTasksForDate(ctx context.Context, date time.Time) (int, error) {
	date = date.Truncate(24 * time.Hour)
	templates, err := s.templateRepo.ListActiveTemplatesForDate(ctx, date)
	if err != nil {
		return 0, err
	}

	generated := 0
	for _, tpl := range templates {
		shouldGenerate, err := ShouldGenerateForDate(&tpl, date)
		if err != nil {
			continue
		}
		if !shouldGenerate {
			continue
		}

		// Create task
		task := &taskdomain.Task{
			Title:         tpl.Title,
			Description:   tpl.Description,
			Status:        taskdomain.StatusNew,
			TemplateID:    &tpl.ID,
			ScheduledDate: &date,
			CreatedAt:     s.now(),
			UpdatedAt:     s.now(),
		}

		_, err = s.repo.Create(ctx, task)
		if err != nil {
			// Ignore unique constraint violations (task already exists)
			if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
				continue
			}
			return generated, err
		}

		generated++
	}

	return generated, nil
}

// GenerateTasksForTemplate generates tasks for a specific template for N days ahead
func (s *Service) GenerateTasksForTemplate(ctx context.Context, templateID int64, daysAhead int) (int, error) {
	template, err := s.templateRepo.GetTemplateByID(ctx, templateID)
	if err != nil {
		return 0, err
	}

	now := s.now().Truncate(24 * time.Hour)
	generated := 0

	for i := 0; i < daysAhead; i++ {
		date := now.AddDate(0, 0, i)
		shouldGenerate, err := ShouldGenerateForDate(template, date)
		if err != nil {
			continue
		}
		if !shouldGenerate {
			continue
		}

		task := &taskdomain.Task{
			Title:         template.Title,
			Description:   template.Description,
			Status:        taskdomain.StatusNew,
			TemplateID:    &template.ID,
			ScheduledDate: &date,
			CreatedAt:     s.now(),
			UpdatedAt:     s.now(),
		}

		_, err = s.repo.Create(ctx, task)
		if err != nil {
			// Ignore unique constraint violations
			if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
				continue
			}
			return generated, err
		}

		generated++
	}

	// Update last generated date
	lastDate := now.AddDate(0, 0, daysAhead-1)
	template.LastGeneratedDate = &lastDate
	template.UpdatedAt = s.now()
	_, err = s.templateRepo.UpdateTemplate(ctx, template)
	if err != nil {
		return generated, err
	}

	return generated, nil
}
