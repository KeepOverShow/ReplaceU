package workbench

import "context"

type Observation struct {
	State      string `json:"state"`
	Reason     string `json:"reason"`
	UID        string `json:"uid"`
	Node       string `json:"node,omitempty"`
	Allocation any    `json:"allocation,omitempty"`
}
type Device struct {
	Node       string `json:"node"`
	ID         string `json:"id"`
	Model      string `json:"model"`
	MemoryMiB  int64  `json:"memory_mib"`
	Healthy    bool   `json:"healthy"`
	Source     string `json:"source"`
	ObservedAt int64  `json:"observed_at"`
}
type Backend interface {
	Ensure(context.Context, Task) error
	Observe(context.Context, Task) (Observation, error)
	Cancel(context.Context, Task) error
	Logs(context.Context, Task) (string, error)
	Devices(context.Context) ([]Device, error)
}
type Controller struct {
	Store   *Store
	Backend Backend
}
