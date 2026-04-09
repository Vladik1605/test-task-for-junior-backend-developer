package main

import (
	"fmt"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
	taskusecase "example.com/taskservice/internal/usecase/task"
)

func main() {
	startDate, _ := time.ParseInLocation("2006-01-02", "2026-04-09", time.UTC)
	template := &taskdomain.TaskTemplate{
		RecurrenceType:   taskdomain.RecurrenceDaily,
		RecurrenceParams: []byte(`{"interval": 2}`),
		StartDate:        startDate,
	}

	checkDate := startDate.AddDate(0, 0, 2) // 2026-04-11
	fmt.Printf("Checking date: %s\n", checkDate.Format("2006-01-02"))

	res, err := taskusecase.ShouldGenerateForDate(template, checkDate)
	fmt.Printf("Result: %v, Error: %v\n", res, err)

	daysDiff := int(checkDate.Sub(startDate.Truncate(24*time.Hour)).Hours() / 24)
	fmt.Printf("Days diff: %d, Interval: 2, Mod: %d\n", daysDiff, daysDiff%2)
}
