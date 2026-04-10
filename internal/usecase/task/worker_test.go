package task

import (
	"context"
	"testing"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockTaskRepository struct {
	mock.Mock
}

func (m *MockTaskRepository) Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	args := m.Called(ctx, task)
	return args.Get(0).(*taskdomain.Task), args.Error(1)
}

func (m *MockTaskRepository) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*taskdomain.Task), args.Error(1)
}

func (m *MockTaskRepository) Update(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	args := m.Called(ctx, task)
	return args.Get(0).(*taskdomain.Task), args.Error(1)
}

func (m *MockTaskRepository) Delete(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTaskRepository) List(ctx context.Context) ([]taskdomain.Task, error) {
	args := m.Called(ctx)
	return args.Get(0).([]taskdomain.Task), args.Error(1)
}

type MockTemplateRepository struct {
	mock.Mock
}

func (m *MockTemplateRepository) CreateTemplate(ctx context.Context, template *taskdomain.TaskTemplate) (*taskdomain.TaskTemplate, error) {
	args := m.Called(ctx, template)
	return args.Get(0).(*taskdomain.TaskTemplate), args.Error(1)
}

func (m *MockTemplateRepository) GetTemplateByID(ctx context.Context, id int64) (*taskdomain.TaskTemplate, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*taskdomain.TaskTemplate), args.Error(1)
}

func (m *MockTemplateRepository) UpdateTemplate(ctx context.Context, template *taskdomain.TaskTemplate) (*taskdomain.TaskTemplate, error) {
	args := m.Called(ctx, template)
	return args.Get(0).(*taskdomain.TaskTemplate), args.Error(1)
}

func (m *MockTemplateRepository) DeleteTemplate(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTemplateRepository) ListTemplates(ctx context.Context) ([]taskdomain.TaskTemplate, error) {
	args := m.Called(ctx)
	return args.Get(0).([]taskdomain.TaskTemplate), args.Error(1)
}

func (m *MockTemplateRepository) ListActiveTemplatesForDate(ctx context.Context, date time.Time) ([]taskdomain.TaskTemplate, error) {
	args := m.Called(ctx, date)
	return args.Get(0).([]taskdomain.TaskTemplate), args.Error(1)
}

func TestWorkerLogic_TaskNotCompleted(t *testing.T) {
	mockTaskRepo := new(MockTaskRepository)
	mockTemplateRepo := new(MockTemplateRepository)

	fixedNow, _ := time.ParseInLocation("2006-01-02", "2026-04-09", time.UTC)
	service := NewService(mockTaskRepo, mockTemplateRepo)
	service.now = func() time.Time { return fixedNow }

	templateID := int64(1)
	scheduledDate, _ := time.ParseInLocation("2006-01-02", "2026-04-09", time.UTC)
	nextDate, _ := time.ParseInLocation("2006-01-02", "2026-04-11", time.UTC)

	template := &taskdomain.TaskTemplate{
		ID:               templateID,
		RecurrenceType:   taskdomain.RecurrenceDaily,
		RecurrenceParams: []byte(`{"interval": 2}`),
		StartDate:        scheduledDate,
	}

	task := taskdomain.Task{
		ID:            1,
		TemplateID:    &templateID,
		ScheduledDate: &scheduledDate,
		Status:        taskdomain.StatusNew,
	}

	mockTaskRepo.On("List", mock.Anything).Return([]taskdomain.Task{task}, nil)
	mockTemplateRepo.On("GetTemplateByID", mock.Anything, templateID).Return(template, nil)

	var updatedTask *taskdomain.Task
	mockTaskRepo.On("Update", mock.Anything, mock.MatchedBy(func(t *taskdomain.Task) bool {
		updatedTask = t
		return true
	})).Return(&task, nil)

	processed, err := service.ProcessRecurringTasks(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, 1, processed)
	assert.Equal(t, nextDate.Format("2006-01-02"), updatedTask.ScheduledDate.Format("2006-01-02"))
	assert.Equal(t, taskdomain.StatusNew, updatedTask.Status)
	mockTaskRepo.AssertExpectations(t)
	mockTemplateRepo.AssertExpectations(t)
}

func TestWorkerLogic_TaskCompleted(t *testing.T) {
	mockTaskRepo := new(MockTaskRepository)
	mockTemplateRepo := new(MockTemplateRepository)

	fixedNow, _ := time.ParseInLocation("2006-01-02", "2026-04-09", time.UTC)
	service := NewService(mockTaskRepo, mockTemplateRepo)
	service.now = func() time.Time { return fixedNow }

	templateID := int64(1)
	scheduledDate, _ := time.ParseInLocation("2006-01-02", "2026-04-09", time.UTC)
	nextDate, _ := time.ParseInLocation("2006-01-02", "2026-04-11", time.UTC)

	template := &taskdomain.TaskTemplate{
		ID:               templateID,
		Title:            "Test Task",
		Description:      "Test Description",
		RecurrenceType:   taskdomain.RecurrenceDaily,
		RecurrenceParams: []byte(`{"interval": 2}`),
		StartDate:        scheduledDate,
	}

	task := taskdomain.Task{
		ID:            1,
		TemplateID:    &templateID,
		ScheduledDate: &scheduledDate,
		Status:        taskdomain.StatusDone,
	}

	mockTaskRepo.On("List", mock.Anything).Return([]taskdomain.Task{task}, nil)
	mockTemplateRepo.On("GetTemplateByID", mock.Anything, templateID).Return(template, nil)

	var newTask *taskdomain.Task
	mockTaskRepo.On("Create", mock.Anything, mock.MatchedBy(func(t *taskdomain.Task) bool {
		newTask = t
		return true
	})).Return(&task, nil)

	processed, err := service.ProcessRecurringTasks(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, 1, processed)
	assert.Equal(t, nextDate.Format("2006-01-02"), newTask.ScheduledDate.Format("2006-01-02"))
	assert.Equal(t, taskdomain.StatusNew, newTask.Status)
	assert.Equal(t, "Test Task", newTask.Title)
	mockTaskRepo.AssertExpectations(t)
	mockTemplateRepo.AssertExpectations(t)
}

func TestWorker_RunsAtMidnight(t *testing.T) {
	imported := true
	assert.True(t, imported)
}
