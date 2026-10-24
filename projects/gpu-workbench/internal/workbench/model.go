package workbench

import (
	"errors"
	"fmt"
	"strings"
)

var ErrQuota = errors.New("team quota exceeded")
var ErrConflict = errors.New("idempotency key conflicts with request")
var ErrNotFound = errors.New("task not found")
var ErrUnauthorized = errors.New("unauthorized")

type Request struct {
	Name            string `json:"name"`
	MemoryMiB       int    `json:"memory_mib"`
	Cores           int    `json:"cores"`
	DurationSeconds int    `json:"duration_seconds"`
	Workload        string `json:"workload"`
	Profile         string `json:"profile"`
	Scenario        string `json:"scenario"`
}

func (r Request) Validate() error {
	if strings.TrimSpace(r.Name) == "" || len(r.Name) > 80 {
		return fmt.Errorf("name must contain 1-80 characters")
	}
	if r.MemoryMiB < 128 || r.MemoryMiB > 1048576 {
		return fmt.Errorf("memory_mib must be 128..1048576")
	}
	if r.Cores < 1 || r.Cores > 100 {
		return fmt.Errorf("cores must be 1..100")
	}
	if r.DurationSeconds < 1 || r.DurationSeconds > 3600 {
		return fmt.Errorf("duration_seconds must be 1..3600")
	}
	if r.Workload != "matmul" && r.Workload != "memory" && r.Workload != "fail" {
		return fmt.Errorf("unsupported workload")
	}
	if r.Profile != "standard" && r.Profile != "batch" {
		return fmt.Errorf("unsupported profile")
	}
	switch r.Scenario {
	case "normal", "create-timeout", "observe-error", "cancel-delay":
	default:
		return fmt.Errorf("unsupported scenario")
	}
	return nil
}
func (r Request) Normalized() Request {
	r.Name = strings.TrimSpace(r.Name)
	if r.Workload == "" {
		r.Workload = "matmul"
	}
	if r.Profile == "" {
		r.Profile = "standard"
	}
	if r.Scenario == "" {
		r.Scenario = "normal"
	}
	return r
}
func Terminal(s string) bool { return s == "Succeeded" || s == "Failed" || s == "Cancelled" }

type Team struct {
	ID           string `json:"id"`
	MaxTasks     int    `json:"max_tasks"`
	MaxMemoryMiB int    `json:"max_memory_mib"`
	MaxCores     int    `json:"max_cores"`
}
type Task struct {
	ID        string  `json:"id"`
	TeamID    string  `json:"team_id"`
	Request   Request `json:"request"`
	State     string  `json:"state"`
	Reason    string  `json:"reason"`
	CreatedAt int64   `json:"created_at"`
	UpdatedAt int64   `json:"updated_at"`
	JobUID    string  `json:"job_uid"`
}
type Quota struct {
	Team
	ActiveTasks   int `json:"active_tasks"`
	UsedMemoryMiB int `json:"used_memory_mib"`
	UsedCores     int `json:"used_cores"`
}
type Event struct {
	ID      int64  `json:"id"`
	TaskID  string `json:"task_id"`
	Time    int64  `json:"time"`
	State   string `json:"state"`
	Message string `json:"message"`
}
