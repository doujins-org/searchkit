package tasks

import "time"

type Task struct {
	ID         int64
	EntityType string
	EntityID   int64
	Model      string
	Reason     string
	Attempts   int
	NextRunAt  time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

