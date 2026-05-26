package queue

import "time"

type JobStatus int

const (
	StatusPending JobStatus = iota
	StatusProcessing
	StatusCompleted
	StatusFailed
)

type Job struct {
	ID          string
	ProjectID   string
	ZipFilePath string
	FinalDir    string
	Priority    int
	CreatedAt   time.Time
	RetryCount  int
}

type JobResult struct {
	JobID     string
	ProjectID string
	Status    JobStatus
	Error     error
}
