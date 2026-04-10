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

func ShouldGenerateForDate(template *taskdomain.TaskTemplate, date time.Time) (bool, error) {
	date = date.Truncate(24 * time.Hour).UTC()
	startDate := template.StartDate.UTC().Truncate(24 * time.Hour)

	if date.Before(startDate) {
		return false, nil
	}

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
		// Calculate days difference by YYYY-MM-DD only, no timezone issues
		y1, m1, d1 := startDate.Date()
		y2, m2, d2 := date.Date()
		date1 := time.Date(y1, m1, d1, 0, 0, 0, 0, time.UTC)
		date2 := time.Date(y2, m2, d2, 0, 0, 0, 0, time.UTC)
		daysDiff := int(date2.Sub(date1).Hours() / 24)

		// For daily recurrence: if date is before start date - never match
		if daysDiff < 0 {
			return false, nil
		}

		// If we start at day 0, next date is +interval days
		return (daysDiff)%params.Interval == 0, nil

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

			if d > lastDayOfMonth && day == lastDayOfMonth {
				return true, nil
			}
		}
		return false, nil

	case taskdomain.RecurrenceSpecificDates:
		var params struct {
			Dates []string `json:"dates"`
		}
		if err := json.Unmarshal(template.RecurrenceParams, &params); err != nil {
			return false, err
		}

		for _, dateStr := range params.Dates {
			d, err := time.ParseInLocation("2006-01-02", dateStr, time.UTC)
			if err != nil {
				continue
			}
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

func (s *Service) FindNextDateForTemplate(ctx context.Context, template *taskdomain.TaskTemplate, from time.Time) (time.Time, error) {
	from = from.Truncate(24 * time.Hour).UTC()

	for i := 0; i < 365; i++ { // Look ahead maximum 1 year
		date := from.AddDate(0, 0, i)

		// Never return date after template end date
		if template.EndDate != nil && date.After(*template.EndDate) {
			return time.Time{}, nil
		}

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

	// Always search from TODAY, never return dates in the past
	firstDate, err := s.FindNextDateForTemplate(ctx, created, s.now())
	if err == nil && !firstDate.IsZero() {
		// Save last generated date to template - THIS WAS MISSING!
		created.LastGeneratedDate = &firstDate
		_, _ = s.templateRepo.UpdateTemplate(ctx, created)

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

func (s *Service) GetTemplateByID(ctx context.Context, id int64) (*taskdomain.TaskTemplate, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}
	return s.templateRepo.GetTemplateByID(ctx, id)
}

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

	_, err = s.GenerateTasksForTemplate(ctx, updated.ID, 0)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *Service) DeleteTemplate(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}
	return s.templateRepo.DeleteTemplate(ctx, id)
}

func (s *Service) ListTemplates(ctx context.Context) ([]taskdomain.TaskTemplate, error) {
	return s.templateRepo.ListTemplates(ctx)
}

func (s *Service) ProcessRecurringTasks(ctx context.Context) (int, error) {
	today := s.now().Truncate(24 * time.Hour)
	allTasks, err := s.repo.List(ctx)
	if err != nil {
		return 0, err
	}

	processed := 0

	templateTasks := make(map[int64][]*taskdomain.Task)
	for i := range allTasks {
		task := &allTasks[i]
		if task.TemplateID == nil {
			continue
		}
		templateTasks[*task.TemplateID] = append(templateTasks[*task.TemplateID], task)
	}

	for templateID, tasks := range templateTasks {
		template, err := s.templateRepo.GetTemplateByID(ctx, templateID)
		if err != nil {
			continue
		}

		for _, task := range tasks {
			var scheduledDate time.Time
			// For DONE tasks even without scheduled_date we should still create next task
			if task.ScheduledDate != nil {
				scheduledDate = task.ScheduledDate.Truncate(24 * time.Hour)
			} else if template.LastGeneratedDate != nil {
				scheduledDate = template.LastGeneratedDate.Truncate(24 * time.Hour)
			} else {
				continue
			}

			// Only process tasks that are STRICTLY OVERDUE (before today)
			// Tasks scheduled exactly for today are still active, don't process them
			if !scheduledDate.Before(today) {
				continue
			}

			nextDate, err := s.FindNextDateForTemplate(ctx, template, today.AddDate(0, 0, 1))
			if err != nil {
				continue
			}
			if nextDate.IsZero() {
				continue
			}

			if task.Status == taskdomain.StatusDone {
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
				task.ScheduledDate = &nextDate
				task.UpdatedAt = s.now()
				_, _ = s.repo.Update(ctx, task)
			}

			processed++
		}
	}

	return processed, nil
}

func (s *Service) ForceAdvanceAllTasks(ctx context.Context) (int, error) {
	allTasks, err := s.repo.List(ctx)
	if err != nil {
		return 0, err
	}

	processed := 0

	templateTasks := make(map[int64][]*taskdomain.Task)
	for i := range allTasks {
		task := &allTasks[i]
		if task.TemplateID == nil {
			continue
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

			currentDate := task.ScheduledDate.Truncate(24 * time.Hour)

			var nextDate time.Time
			for i := 1; i < 365; i++ {
				checkDate := currentDate.AddDate(0, 0, i)
				shouldGenerate, err := ShouldGenerateForDate(template, checkDate)
				if err == nil && shouldGenerate {
					nextDate = checkDate
					break
				}
			}

			if nextDate.IsZero() {
				continue
			}
			if err != nil {
				continue
			}
			if nextDate.IsZero() {
				continue
			}

			if task.Status == taskdomain.StatusDone {
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
				task.ScheduledDate = &nextDate
				task.UpdatedAt = s.now()
				_, _ = s.repo.Update(ctx, task)
			}

			processed++
		}
	}

	return processed, nil
}

func (s *Service) GenerateTasksForTemplate(ctx context.Context, templateID int64, daysAhead int) (int, error) {
	template, err := s.templateRepo.GetTemplateByID(ctx, templateID)
	if err != nil {
		return 0, err
	}

	allTasks, err := s.repo.List(ctx)
	if err != nil {
		return 0, err
	}

	for _, task := range allTasks {
		if task.TemplateID != nil && *task.TemplateID == templateID {
			_ = s.repo.Delete(ctx, task.ID)
		}
	}

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

// GenerateTasksForDate processes tasks as if it was the specified date
// SKIPS ALL INTERMEDIATE STEPS and jumps directly up to requested date
func (s *Service) GenerateTasksForDate(ctx context.Context, date time.Time) (int, error) {
	date = date.Truncate(24 * time.Hour).UTC()
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
			var currentDate time.Time
			// For DONE tasks even without scheduled_date we should still create next task
			if task.ScheduledDate != nil {
				currentDate = task.ScheduledDate.Truncate(24 * time.Hour)
			} else if template.LastGeneratedDate != nil {
				currentDate = template.LastGeneratedDate.Truncate(24 * time.Hour)
			} else {
				continue
			}

			// Skip tasks that are already after requested date
			if !currentDate.Before(date) {
				continue
			}

			// JUMP FORWARD UNTIL WE HAVE DATE >= requested date
			// Always find the FIRST date that is NOT BEFORE requested date
			nextDate := currentDate
			// Always step at least once forward
			for {
				candidate, err := s.FindNextDateForTemplate(ctx, template, nextDate.AddDate(0, 0, 1))
				if err != nil || candidate.IsZero() {
					break
				}
				// Respect end date boundary
				if template.EndDate != nil && candidate.After(*template.EndDate) {
					break
				}

				nextDate = candidate

				// Stop when we reach or pass requested date
				if !candidate.Before(date) {
					break
				}
			}

			// If we found no valid dates at all - skip
			if nextDate.Equal(currentDate) {
				continue
			}

			if task.Status == taskdomain.StatusDone {
				// Task was completed - create NEW task for final calculated date
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
				// Task was NOT completed - MOVE it directly to final date
				task.ScheduledDate = &nextDate
				task.UpdatedAt = s.now()
				_, _ = s.repo.Update(ctx, task)
			}

			// ALWAYS update template last generated date
			template.LastGeneratedDate = &nextDate
			template.UpdatedAt = s.now()
			_, _ = s.templateRepo.UpdateTemplate(ctx, template)

			processed++
		}
	}

	return processed, nil
}
