package workbench

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestControllerUncertainCreateAndRestart(t *testing.T) {
	s := newStore(t)
	r := req()
	r.Scenario = "create-timeout"
	task, _, _ := s.Submit("alpha", "timeout", r)
	d := NewDemo(s)
	now := time.Now()
	d.now = func() time.Time { return now }
	c := Controller{Store: s, Backend: d}
	if e := c.Step(context.Background()); e != nil {
		t.Fatal(e)
	}
	q, _ := s.Quota("alpha")
	if q.ActiveTasks != 1 {
		t.Fatal("released on create timeout")
	}
	c = Controller{Store: s, Backend: NewDemo(s)}
	c.Backend.(*Demo).now = func() time.Time { return now }
	if e := c.Step(context.Background()); e != nil {
		t.Fatal(e)
	}
	got, _ := s.Get("alpha", task.ID)
	if got.State != "Running" && got.State != "Pending" {
		t.Fatal(got)
	}
	var n int
	s.db.QueryRow(`SELECT COUNT(*) FROM demo_runs`).Scan(&n)
	if n != 1 {
		t.Fatal("duplicate execution", n)
	}
	now = now.Add(20 * time.Second)
	c.Step(context.Background())
	q, _ = s.Quota("alpha")
	if q.ActiveTasks != 0 {
		t.Fatal(q)
	}
}
func TestControllerCancellationWaitsForBackend(t *testing.T) {
	s := newStore(t)
	r := req()
	r.Scenario = "cancel-delay"
	task, _, _ := s.Submit("alpha", "cancel", r)
	d := NewDemo(s)
	now := time.Now()
	d.now = func() time.Time { return now }
	c := Controller{Store: s, Backend: d}
	c.Step(context.Background())
	s.Cancel("alpha", task.ID)
	c.Step(context.Background())
	got, _ := s.Get("alpha", task.ID)
	q, _ := s.Quota("alpha")
	if got.State != "Cancelling" || q.ActiveTasks != 1 {
		t.Fatalf("early release %+v %+v", got, q)
	}
	now = now.Add(5 * time.Second)
	c.Step(context.Background())
	got, _ = s.Get("alpha", task.ID)
	q, _ = s.Quota("alpha")
	if got.State != "Cancelled" || q.ActiveTasks != 0 {
		t.Fatalf("not released %+v %+v", got, q)
	}
	c.Step(context.Background())
	var n int
	s.db.QueryRow(`SELECT COUNT(*) FROM demo_runs`).Scan(&n)
	if n != 1 {
		t.Fatal(n)
	}
}
func TestControllerObservationErrorKeepsQuota(t *testing.T) {
	s := newStore(t)
	r := req()
	r.Scenario = "observe-error"
	task, _, _ := s.Submit("alpha", "observe", r)
	d := NewDemo(s)
	c := Controller{Store: s, Backend: d}
	c.Step(context.Background())
	c.Step(context.Background())
	q, _ := s.Quota("alpha")
	if q.ActiveTasks != 1 {
		t.Fatal(q)
	}
	s.Cancel("alpha", task.ID)
	c.Step(context.Background())
	got, _ := s.Get("alpha", task.ID)
	if got.State != "Cancelled" {
		t.Fatal(got)
	}
}

type brokenBackend struct{ Backend }

func (b brokenBackend) Observe(context.Context, Task) (Observation, error) {
	return Observation{}, errors.New("offline")
}
func TestControllerBackendOfflineNeverReleases(t *testing.T) {
	s := newStore(t)
	s.Submit("alpha", "offline", req())
	c := Controller{Store: s, Backend: brokenBackend{}}
	c.Step(context.Background())
	q, _ := s.Quota("alpha")
	if q.ActiveTasks != 1 {
		t.Fatal(q)
	}
}
