package task

import (
	"encoding/json"
	"testing"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
	"github.com/stretchr/testify/assert"
)

func TestShouldGenerateForDate(t *testing.T) {
	startDate, _ := time.ParseInLocation("2006-01-02", "2026-04-09", time.UTC)

	t.Run("daily recurrence with interval 1", func(t *testing.T) {
		params, _ := json.Marshal(taskdomain.DailyRecurrence{Interval: 1})
		template := &taskdomain.TaskTemplate{
			RecurrenceType:   taskdomain.RecurrenceDaily,
			RecurrenceParams: params,
			StartDate:        startDate,
		}

		result, err := ShouldGenerateForDate(template, startDate)
		assert.NoError(t, err)
		assert.True(t, result)

		result, err = ShouldGenerateForDate(template, startDate.AddDate(0, 0, 1))
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("daily recurrence with interval 2", func(t *testing.T) {
		params, _ := json.Marshal(taskdomain.DailyRecurrence{Interval: 2})
		template := &taskdomain.TaskTemplate{
			RecurrenceType:   taskdomain.RecurrenceDaily,
			RecurrenceParams: params,
			StartDate:        startDate,
		}

		result, err := ShouldGenerateForDate(template, startDate)
		assert.NoError(t, err)
		assert.True(t, result)

		result, err = ShouldGenerateForDate(template, startDate.AddDate(0, 0, 1))
		assert.NoError(t, err)
		assert.False(t, result)

		result, err = ShouldGenerateForDate(template, startDate.AddDate(0, 0, 2))
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("monthly recurrence on 15th", func(t *testing.T) {
		params, _ := json.Marshal(taskdomain.MonthlyRecurrence{Days: []int{15}})
		template := &taskdomain.TaskTemplate{
			RecurrenceType:   taskdomain.RecurrenceMonthly,
			RecurrenceParams: params,
			StartDate:        startDate,
		}

		april15, _ := time.ParseInLocation("2006-01-02", "2026-04-15", time.UTC)
		result, err := ShouldGenerateForDate(template, april15)
		assert.NoError(t, err)
		assert.True(t, result)

		may15, _ := time.ParseInLocation("2006-01-02", "2026-05-15", time.UTC)
		result, err = ShouldGenerateForDate(template, may15)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("monthly recurrence on 31st - handles February", func(t *testing.T) {
		startDate, _ := time.ParseInLocation("2006-01-02", "2026-02-01", time.UTC)
		params, _ := json.Marshal(taskdomain.MonthlyRecurrence{Days: []int{31}})
		template := &taskdomain.TaskTemplate{
			RecurrenceType:   taskdomain.RecurrenceMonthly,
			RecurrenceParams: params,
			StartDate:        startDate,
		}

		// March 31 should work
		march31, _ := time.ParseInLocation("2006-01-02", "2026-03-31", time.UTC)
		result, err := ShouldGenerateForDate(template, march31)
		assert.NoError(t, err)
		assert.True(t, result)

		// February 28 (2026 is not a leap year) should work instead of 31
		feb28, _ := time.ParseInLocation("2006-01-02", "2026-02-28", time.UTC)
		result, err = ShouldGenerateForDate(template, feb28)
		assert.NoError(t, err)
		assert.True(t, result)

		// April 30 should work instead of 31
		april30, _ := time.ParseInLocation("2006-01-02", "2026-04-30", time.UTC)
		result, err = ShouldGenerateForDate(template, april30)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("even days recurrence", func(t *testing.T) {
		params, _ := json.Marshal(taskdomain.EvenOddRecurrence{Type: "even"})
		template := &taskdomain.TaskTemplate{
			RecurrenceType:   taskdomain.RecurrenceEvenOdd,
			RecurrenceParams: params,
			StartDate:        startDate,
		}

		april10, _ := time.ParseInLocation("2006-01-02", "2026-04-10", time.UTC) // even
		result, err := ShouldGenerateForDate(template, april10)
		assert.NoError(t, err)
		assert.True(t, result)

		april11, _ := time.ParseInLocation("2006-01-02", "2026-04-11", time.UTC) // odd
		result, err = ShouldGenerateForDate(template, april11)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("odd days recurrence", func(t *testing.T) {
		params, _ := json.Marshal(taskdomain.EvenOddRecurrence{Type: "odd"})
		template := &taskdomain.TaskTemplate{
			RecurrenceType:   taskdomain.RecurrenceEvenOdd,
			RecurrenceParams: params,
			StartDate:        startDate,
		}

		april9, _ := time.ParseInLocation("2006-01-02", "2026-04-09", time.UTC) // odd
		result, err := ShouldGenerateForDate(template, april9)
		assert.NoError(t, err)
		assert.True(t, result)

		april10, _ := time.ParseInLocation("2006-01-02", "2026-04-10", time.UTC) // even
		result, err = ShouldGenerateForDate(template, april10)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("specific dates recurrence", func(t *testing.T) {
		date1 := "2026-04-15"
		date2 := "2026-05-20"
		params, _ := json.Marshal(taskdomain.SpecificDatesRecurrence{Dates: []string{date1, date2}})

		date1Time, _ := time.ParseInLocation("2006-01-02", date1, time.UTC)
		date2Time, _ := time.ParseInLocation("2006-01-02", date2, time.UTC)

		template := &taskdomain.TaskTemplate{
			RecurrenceType:   taskdomain.RecurrenceSpecificDates,
			RecurrenceParams: params,
			StartDate:        startDate,
		}

		result, err := ShouldGenerateForDate(template, date1Time)
		assert.NoError(t, err)
		assert.True(t, result)

		result, err = ShouldGenerateForDate(template, date2Time)
		assert.NoError(t, err)
		assert.True(t, result)

		result, err = ShouldGenerateForDate(template, startDate)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("date before start date", func(t *testing.T) {
		params, _ := json.Marshal(taskdomain.DailyRecurrence{Interval: 1})
		template := &taskdomain.TaskTemplate{
			RecurrenceType:   taskdomain.RecurrenceDaily,
			RecurrenceParams: params,
			StartDate:        startDate,
		}

		beforeDate, _ := time.ParseInLocation("2006-01-02", "2026-04-08", time.UTC)
		result, err := ShouldGenerateForDate(template, beforeDate)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("date after end date", func(t *testing.T) {
		endDate, _ := time.ParseInLocation("2006-01-02", "2026-04-20", time.UTC)
		params, _ := json.Marshal(taskdomain.DailyRecurrence{Interval: 1})
		template := &taskdomain.TaskTemplate{
			RecurrenceType:   taskdomain.RecurrenceDaily,
			RecurrenceParams: params,
			StartDate:        startDate,
			EndDate:          &endDate,
		}

		afterDate, _ := time.ParseInLocation("2006-01-02", "2026-04-21", time.UTC)
		result, err := ShouldGenerateForDate(template, afterDate)
		assert.NoError(t, err)
		assert.False(t, result)
	})
}
