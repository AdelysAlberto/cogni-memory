package network

import (
	"time"

	"github.com/AdelysAlberto/cogni/internal/core"
)

// SyncPacket represents an export or sync payload transferred between peers.
type SyncPacket struct {
	Version     string        `json:"version"`
	ProjectName string        `json:"project_name"`
	Sender      string        `json:"sender"`
	Timestamp   time.Time     `json:"timestamp"`
	Memories    []core.Memory `json:"memories"`
	Code        string        `json:"code,omitempty"`
}

// SyncResponse is returned by the receiver after processing incoming memories.
type SyncResponse struct {
	Success   bool     `json:"success"`
	Inserted  int      `json:"inserted"`
	Updated   int      `json:"updated"`
	Conflicts int      `json:"conflicts"`
	Details   []string `json:"details,omitempty"`
	Message   string   `json:"message"`
}
