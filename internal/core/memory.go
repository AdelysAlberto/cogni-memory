package core

import "time"

// Memory represents a stored synthetic semantic memory record
type Memory struct {
	ID               int64     `json:"id"`
	ProjectName      string    `json:"project_name"`
	Category         string    `json:"category"`
	Title            string    `json:"title"`
	TopicKey         string    `json:"topic_key,omitempty"`
	SummarySignature string    `json:"summary_signature"`
	Tags             string    `json:"tags"`
	Source           string    `json:"source,omitempty"` // "local" or "global"
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Stats represents aggregated memory usage metrics
type Stats struct {
	TotalMemories        int64 `json:"totalMemories"`
	TotalProjects        int64 `json:"totalProjects"`
	EstimatedTokensSaved int64 `json:"estimatedTokensSaved"`
}

// Result represents an explicit Result Pattern envelope for CLI/API outputs
type Result[T any] struct {
	Success bool   `json:"success"`
	Data    T      `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Ok creates a successful Result
func Ok[T any](data T) Result[T] {
	return Result[T]{Success: true, Data: data}
}

// Fail creates an error Result
func Fail[T any](err string) Result[T] {
	return Result[T]{Success: false, Error: err}
}
