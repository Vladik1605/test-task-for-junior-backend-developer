package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
	taskusecase "example.com/taskservice/internal/usecase/task"
)

type TemplateHandler struct {
	usecase taskusecase.Usecase
}

func NewTemplateHandler(usecase taskusecase.Usecase) *TemplateHandler {
	return &TemplateHandler{usecase: usecase}
}

type templateMutationDTO struct {
	Title            string                    `json:"title"`
	Description      string                    `json:"description"`
	RecurrenceType   taskdomain.RecurrenceType `json:"recurrence_type"`
	RecurrenceParams map[string]interface{}    `json:"recurrence_params"`
	StartDate        string                    `json:"start_date"` // ISO format "YYYY-MM-DD"
	EndDate          *string                   `json:"end_date,omitempty"`
}

type templateDTO struct {
	ID                int64           `json:"id"`
	Title             string          `json:"title"`
	Description       string          `json:"description"`
	RecurrenceType    string          `json:"recurrence_type"`
	RecurrenceParams  json.RawMessage `json:"recurrence_params"`
	StartDate         string          `json:"start_date"`
	EndDate           *string         `json:"end_date,omitempty"`
	LastGeneratedDate *string         `json:"last_generated_date,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type generateTasksRequest struct {
	Date string `json:"date"`
}

type generateTasksResponse struct {
	Generated int `json:"generated"`
}

func newTemplateDTO(template *taskdomain.TaskTemplate) templateDTO {
	startDate := template.StartDate.Format("2006-01-02")

	var endDate *string
	if template.EndDate != nil {
		formatted := template.EndDate.Format("2006-01-02")
		endDate = &formatted
	}

	var lastGenerated *string
	if template.LastGeneratedDate != nil {
		formatted := template.LastGeneratedDate.Format("2006-01-02")
		lastGenerated = &formatted
	}

	return templateDTO{
		ID:                template.ID,
		Title:             template.Title,
		Description:       template.Description,
		RecurrenceType:    string(template.RecurrenceType),
		RecurrenceParams:  template.RecurrenceParams,
		StartDate:         startDate,
		EndDate:           endDate,
		LastGeneratedDate: lastGenerated,
		CreatedAt:         template.CreatedAt,
		UpdatedAt:         template.UpdatedAt,
	}
}

func (h *TemplateHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req templateMutationDTO
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	startDate, err := time.ParseInLocation("2006-01-02", req.StartDate, time.UTC)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var endDate *time.Time
	if req.EndDate != nil {
		parsed, err := time.Parse("2006-01-02", *req.EndDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		endDate = &parsed
	}

	paramsJSON, err := json.Marshal(req.RecurrenceParams)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	created, err := h.usecase.CreateTemplate(r.Context(), taskusecase.CreateTemplateInput{
		Title:            req.Title,
		Description:      req.Description,
		RecurrenceType:   req.RecurrenceType,
		RecurrenceParams: paramsJSON,
		StartDate:        startDate,
		EndDate:          endDate,
	})
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, newTemplateDTO(created))
}

func (h *TemplateHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	template, err := h.usecase.GetTemplateByID(r.Context(), id)
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newTemplateDTO(template))
}

func (h *TemplateHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var req templateMutationDTO
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var endDate *time.Time
	if req.EndDate != nil {
		parsed, err := time.Parse("2006-01-02", *req.EndDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		endDate = &parsed
	}

	paramsJSON, err := json.Marshal(req.RecurrenceParams)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	updated, err := h.usecase.UpdateTemplate(r.Context(), id, taskusecase.UpdateTemplateInput{
		Title:            req.Title,
		Description:      req.Description,
		RecurrenceType:   req.RecurrenceType,
		RecurrenceParams: paramsJSON,
		StartDate:        startDate,
		EndDate:          endDate,
	})
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newTemplateDTO(updated))
}

func (h *TemplateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.usecase.DeleteTemplate(r.Context(), id); err != nil {
		writeUsecaseError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *TemplateHandler) List(w http.ResponseWriter, r *http.Request) {
	templates, err := h.usecase.ListTemplates(r.Context())
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	response := make([]templateDTO, 0, len(templates))
	for i := range templates {
		response = append(response, newTemplateDTO(&templates[i]))
	}

	writeJSON(w, http.StatusOK, response)
}

// Manual trigger endpoint - triggers task generation for a specific date
func (h *TemplateHandler) GenerateForDate(w http.ResponseWriter, r *http.Request) {
	var req generateTasksRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	generated, err := h.usecase.GenerateTasksForDate(r.Context(), date)
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, generateTasksResponse{Generated: generated})
}

// Manual trigger endpoint - generates tasks for next N days for a specific template
func (h *TemplateHandler) GenerateForTemplate(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	type generateTemplateRequest struct {
		DaysAhead int `json:"days_ahead"`
	}

	var req generateTemplateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if req.DaysAhead <= 0 || req.DaysAhead > 365 {
		req.DaysAhead = 30 // Default to 30 days
	}

	generated, err := h.usecase.GenerateTasksForTemplate(r.Context(), id, req.DaysAhead)
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, generateTasksResponse{Generated: generated})
}

// Manual trigger to run full worker logic immediately
// This is exactly the same logic that runs automatically at midnight
func (h *TemplateHandler) RunWorkerNow(w http.ResponseWriter, r *http.Request) {
	processed, err := h.usecase.ProcessRecurringTasks(r.Context())
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{
		"processed_tasks": processed,
	})
}

// Force advance ALL recurring tasks to their next date
// FOR TESTING ONLY! This skips all date checks and always moves tasks forward
func (h *TemplateHandler) ForceAdvanceAll(w http.ResponseWriter, r *http.Request) {
	processed, err := h.usecase.ForceAdvanceAllTasks(r.Context())
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{
		"advanced_tasks": processed,
	})
}
