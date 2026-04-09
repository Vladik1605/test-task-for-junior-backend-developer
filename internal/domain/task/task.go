package task

import (
	"encoding/json"
	"time"
)

type Status string

const (
	StatusNew        Status = "new"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

type RecurrenceType string

const (
	RecurrenceDaily         RecurrenceType = "daily"
	RecurrenceMonthly       RecurrenceType = "monthly"
	RecurrenceSpecificDates RecurrenceType = "specific_dates"
	RecurrenceEvenOdd       RecurrenceType = "even_odd"
)

type Task struct {
	ID            int64      `json:"id"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	Status        Status     `json:"status"`
	TemplateID    *int64     `json:"template_id,omitempty"`
	ScheduledDate *time.Time `json:"scheduled_date,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type TaskTemplate struct {
	ID                int64           `json:"id"`
	Title             string          `json:"title"`
	Description       string          `json:"description"`
	RecurrenceType    RecurrenceType  `json:"recurrence_type"`
	RecurrenceParams  json.RawMessage `json:"recurrence_params"`
	StartDate         time.Time       `json:"start_date"`
	EndDate           *time.Time      `json:"end_date,omitempty"`
	LastGeneratedDate *time.Time      `json:"last_generated_date,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// Recurrence parameter types
type DailyRecurrence struct {
	Interval int `json:"interval"`
}

type MonthlyRecurrence struct {
	Days []int `json:"days"`
}

type SpecificDatesRecurrence struct {
	Dates []time.Time `json:"dates"`
}

type EvenOddRecurrence struct {
	Type string `json:"type"` // "even" or "odd"
}

func (s Status) Valid() bool {
	switch s {
	case StatusNew, StatusInProgress, StatusDone:
		return true
	default:
		return false
	}
}

func (rt RecurrenceType) Valid() bool {
	switch rt {
	case RecurrenceDaily, RecurrenceMonthly, RecurrenceSpecificDates, RecurrenceEvenOdd:
		return true
	default:
		return false
	}
}
