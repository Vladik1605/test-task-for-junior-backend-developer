package worker

import (
	"context"
	"log"
	"time"

	taskusecase "example.com/taskservice/internal/usecase/task"
)

type RecurrenceWorker struct {
	usecase taskusecase.Usecase
	stop    chan struct{}
}

func NewRecurrenceWorker(usecase taskusecase.Usecase) *RecurrenceWorker {
	return &RecurrenceWorker{
		usecase: usecase,
		stop:    make(chan struct{}),
	}
}

func (w *RecurrenceWorker) Start(ctx context.Context) {
	go w.run(ctx)
}

func (w *RecurrenceWorker) Stop() {
	close(w.stop)
}

func (w *RecurrenceWorker) run(ctx context.Context) {
	log.Println("Recurrence worker started")

	for {
		select {
		case <-w.stop:
			log.Println("Recurrence worker stopped")
			return
		default:
			processed, err := w.usecase.ProcessRecurringTasks(ctx)
			if err != nil {
				log.Printf("Error processing recurring tasks: %v", err)
			} else {
				log.Printf("Processed %d recurring tasks", processed)
			}

			sleepDuration := w.sleepUntilNextMidnight()
			log.Printf("Recurrence worker sleeping until next midnight (%s)", sleepDuration)

			select {
			case <-time.After(sleepDuration):
			case <-w.stop:
				log.Println("Recurrence worker stopped")
				return
			}
		}
	}
}

func (w *RecurrenceWorker) sleepUntilNextMidnight() time.Duration {
	now := time.Now().UTC()
	nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	return time.Until(nextMidnight)
}
