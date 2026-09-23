package core

import (
	"strings"
	"time"
)

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

// SessionSummary represents structured end-of-session or post-compaction context
type SessionSummary struct {
	Goal          string `json:"goal"`
	Instructions  string `json:"instructions,omitempty"`
	Discoveries   string `json:"discoveries,omitempty"`
	Accomplished  string `json:"accomplished"`
	NextSteps     string `json:"next_steps,omitempty"`
	RelevantFiles string `json:"relevant_files,omitempty"`
}

// FormatSessionSummary formats a SessionSummary into a clean, high-density summary signature
func FormatSessionSummary(s SessionSummary) string {
	parts := make([]string, 0, 6)
	if s.Goal != "" {
		parts = append(parts, "Goal: "+s.Goal)
	}
	if s.Accomplished != "" {
		parts = append(parts, "Accomplished: "+s.Accomplished)
	}
	if s.Discoveries != "" {
		parts = append(parts, "Discoveries: "+s.Discoveries)
	}
	if s.NextSteps != "" {
		parts = append(parts, "Next: "+s.NextSteps)
	}
	if s.RelevantFiles != "" {
		parts = append(parts, "Where: "+s.RelevantFiles)
	}
	if s.Instructions != "" {
		parts = append(parts, "Instructions: "+s.Instructions)
	}
	return strings.Join(parts, " | ")
}

// BuildSummarySignature constructs a 4-part structured signature: What | Why | Where | Learned
func BuildSummarySignature(what, why, where, learned string) string {
	parts := make([]string, 0, 4)
	if what != "" {
		parts = append(parts, "What: "+what)
	}
	if why != "" {
		parts = append(parts, "Why: "+why)
	}
	if where != "" {
		parts = append(parts, "Where: "+where)
	}
	if learned != "" {
		parts = append(parts, "Learned: "+learned)
	}
	return strings.Join(parts, " | ")
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
