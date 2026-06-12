package entities

import (
	"errors"
	"time"
)

type ProjectStatus string

var (
	StatusPending  ProjectStatus = "pending"
	StatusBuilding ProjectStatus = "building"
	StatusRunning  ProjectStatus = "running"
	StatusFailed   ProjectStatus = "failed"
	StatusStopped  ProjectStatus = "stopped"
)

type Project struct {
	// ID            int64          // Database ID (BIGSERIAL)
	ID string
	// UserID        string          // Foreign key to users table
	Name           string        // User-given name for the project
	UniqueKey      string        // Short, unique identifier for URLs (e.g., "a1b2c3d4")
	SourceType     string        // e.g., "zip", "git_repo"
	SourceLocation string        // Path on disk, URL, etc.
	Status         ProjectStatus // Current status of the project
	Port           int           // Port the project is running on
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeployedAt     *time.Time
}

type Settings map[string]any

type ProjectWithSetting struct {
	Project  *Project
	Settings Settings
}

// Simple Validation

func (p *Project) Validate() error {
	// if p.UserID <= "" {
	// 	return errors.New("project must belong to a valid user")
	// }
	if p.Name == "" {
		return errors.New("project name cannot be empty")
	}
	if p.UniqueKey == "" {
		return errors.New("project unique key cannot be empty")
	}
	if p.SourceLocation == "" {
		return errors.New("project source location cannot be empty")
	}
	if !isValidStatus(p.Status) {
		return errors.New("invalid project status")
	}
	return nil
}

func isValidStatus(status ProjectStatus) bool {
	switch status {
	case StatusPending, StatusBuilding, StatusRunning, StatusFailed, StatusStopped:
		return true
	default:
		return false
	}
}

func NewProject(name, uniqueKey, sourceType, sourceLocation string) *Project {
	now := time.Now()
	return &Project{
		Name:           name,
		UniqueKey:      uniqueKey,
		SourceType:     sourceType,
		SourceLocation: sourceLocation,
		Status:         StatusPending, // Default status
		Port:           0,             // Default port
		CreatedAt:      now,
		UpdatedAt:      now,
		DeployedAt:     nil,
	}
}
