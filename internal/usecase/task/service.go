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

// FindNextDateForTemplate finds the next date starting from `from` when this template should generate a task
func (s *Service) FindNextDateForTemplate(ctx context.Context, template *taskdomain.TaskTemplate, from time.Time) (time.Time, error) {
	from = from.Truncate(24 * time.Hour).UTC()

	for i := 0; i < 365; i++ { // Look ahead maximum 1 year
		date := from.AddDate(0, 0, i)
		shouldGenerate, err := ShouldGenerateForDate(template, date)
		if err != nil {
			return time.Time{}, err
		}
		if shouldGenerate {
			return date, nil
		}
	}

	return time.Time{}, fmt.Errorf("no upcoming date found for template in next year")
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

	// Generate ONLY THE FIRST upcoming task when template is created
	firstDate, err := s.FindNextDateForTemplate(ctx, created, s.now())
	if err == nil && !firstDate.IsZero() {
		task := &taskdomain.Task{
			Title:         created.Title,
			Description:   created.Description,
			Status:        taskdomain.StatusNew,
			TemplateID:    &created.ID,
			ScheduledDate: &firstDate,
			CreatedAt:     s.now(),
			UpdatedAt:     s.now(),
		}
		_, _ = s.repo.Create(ctx, task)
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

	// When template is updated - regenerate only the single upcoming task
	_, err = s.GenerateTasksForTemplate(ctx, updated.ID, 0)
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

// ProcessRecurringTasks runs once per midnight, processes all overdue recurring tasks
// This is the core logic you requested:
// - If task is DONE: generate new task on next suitable date
// - If task is NOT DONE: shift its scheduled date to next suitable date
func (s *Service) ProcessRecurringTasks(ctx context.Context) (int, error) {
	today := s.now().Truncate(24 * time.Hour)
	allTasks, err := s.repo.List(ctx)
	if err != nil {
		return 0, err
	}

	processed := 0

	// Group tasks by template ID
	templateTasks := make(map[int64][]*taskdomain.Task)
	for i := range allTasks {
		task := &allTasks[i]
		if task.TemplateID == nil {
			continue // Skip non-recurring tasks
		}
		templateTasks[*task.TemplateID] = append(templateTasks[*task.TemplateID], task)
	}

	for templateID, tasks := range templateTasks {
		template, err := s.templateRepo.GetTemplateByID(ctx, templateID)
		if err != nil {
			continue
		}

		for _, task := range tasks {
			if task.ScheduledDate == nil {
				continue
			}

			scheduledDate := task.ScheduledDate.Truncate(24 * time.Hour)

			// Only process tasks that are overdue
			if scheduledDate.After(today) {
				continue
			}

			// Find next suitable date starting from tomorrow
			nextDate, err := s.FindNextDateForTemplate(ctx, template, today.AddDate(0, 0, 1))
			if err != nil {
				continue
			}
			if nextDate.IsZero() {
				continue // No more future dates for this template
			}

			if task.Status == taskdomain.StatusDone {
				// Task was completed - create NEW task for next date
				newTask := &taskdomain.Task{
					Title:         template.Title,
					Description:   template.Description,
					Status:        taskdomain.StatusNew,
					TemplateID:    &template.ID,
					ScheduledDate: &nextDate,
					CreatedAt:     s.now(),
					UpdatedAt:     s.now(),
				}
				_, _ = s.repo.Create(ctx, newTask)
			} else {
				// Task was NOT completed - MOVE it to next date instead of creating duplicate
				task.ScheduledDate = &nextDate
				task.UpdatedAt = s.now()
				_, _ = s.repo.Update(ctx, task)
			}

			processed++
		}
	}

	return processed, nil
}

// GenerateTasksForDate - kept for backward compatibility with API, now uses new logic
func (s *Service) GenerateTasksForDate(ctx context.Context, date time.Time) (int, error) {
	return s.ProcessRecurringTasks(ctx)
}

// GenerateTasksForTemplate - kept for backward compatibility
func (s *Service) GenerateTasksForTemplate(ctx context.Context, templateID int64, daysAhead int) (int, error) {
	template, err := s.templateRepo.GetTemplateByID(ctx, templateID)
	if err != nil {
		return 0, err
	}

	// Delete all existing tasks for this template first
	allTasks, err := s.repo.List(ctx)
	if err != nil {
		return 0, err
	}

	for _, task := range allTasks {
		if task.TemplateID != nil && *task.TemplateID == templateID {
			_ = s.repo.Delete(ctx, task.ID)
		}
	}

	// Create ONLY ONE upcoming task
	firstDate, err := s.FindNextDateForTemplate(ctx, template, s.now())
	if err != nil || firstDate.IsZero() {
		return 0, err
	}

	task := &taskdomain.Task{
		Title:         template.Title,
		Description:   template.Description,
		Status:        taskdomain.StatusNew,
		TemplateID:    &template.ID,
		ScheduledDate: &firstDate,
		CreatedAt:     s.now(),
		UpdatedAt:     s.now(),
	}
	_, err = s.repo.Create(ctx, task)
	if err != nil {
		return 0, err
	}

	return 1, nil
}
