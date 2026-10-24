package workbench

import (
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	s, e := Open(filepath.Join(t.TempDir(), "state.db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	if e = s.ConfigureTeam(Team{ID: "alpha", MaxTasks: 2, MaxMemoryMiB: 2048, MaxCores: 100}, "test-token-alpha"); e != nil {
		t.Fatal(e)
	}
	if e = s.ConfigureTeam(Team{ID: "beta", MaxTasks: 2, MaxMemoryMiB: 2048, MaxCores: 100}, "test-token-beta"); e != nil {
		t.Fatal(e)
	}
	return s
}
func req() Request {
	return Request{Name: "small-job", MemoryMiB: 1024, Cores: 50, DurationSeconds: 10, Workload: "matmul", Profile: "standard", Scenario: "normal"}
}
func TestStoreAtomicQuota(t *testing.T) {
	s := newStore(t)
	var wg sync.WaitGroup
	var accepted atomic.Int32
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _, e := s.Submit("alpha", string(rune('A'+i)), req())
			if e == nil {
				accepted.Add(1)
			} else if !errors.Is(e, ErrQuota) {
				t.Errorf("unexpected: %v", e)
			}
		}(i)
	}
	wg.Wait()
	if accepted.Load() != 2 {
		t.Fatalf("accepted=%d want 2", accepted.Load())
	}
	q, e := s.Quota("alpha")
	if e != nil || q.ActiveTasks != 2 || q.UsedMemoryMiB != 2048 {
		t.Fatalf("quota=%+v err=%v", q, e)
	}
}
func TestStoreIdempotencyAndOwnership(t *testing.T) {
	s := newStore(t)
	a, _, e := s.Submit("alpha", "same", req())
	if e != nil {
		t.Fatal(e)
	}
	b, replay, e := s.Submit("alpha", "same", req())
	if e != nil || !replay || a.ID != b.ID {
		t.Fatalf("replay %+v %v %v", b, replay, e)
	}
	changed := req()
	changed.Cores = 20
	_, _, e = s.Submit("alpha", "same", changed)
	if !errors.Is(e, ErrConflict) {
		t.Fatal(e)
	}
	if _, e = s.Get("beta", a.ID); !errors.Is(e, ErrNotFound) {
		t.Fatal("cross-team read", e)
	}
	if e = s.Cancel("beta", a.ID); !errors.Is(e, ErrNotFound) {
		t.Fatal("cross-team cancellation", e)
	}
	if _, e = s.Authenticate("test-token-alpha"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Authenticate("wrong"); !errors.Is(e, ErrUnauthorized) {
		t.Fatal(e)
	}
}
func TestStoreDurabilityAndRelease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "persist.db")
	s, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	s.ConfigureTeam(Team{ID: "alpha", MaxTasks: 2, MaxMemoryMiB: 2048, MaxCores: 100}, "token")
	a, _, e := s.Submit("alpha", "once", req())
	if e != nil {
		t.Fatal(e)
	}
	s.Close()
	s, e = Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	q, _ := s.Quota("alpha")
	if q.ActiveTasks != 1 {
		t.Fatal(q)
	}
	if e = s.Transition(a.ID, "Running", "executing", ""); e != nil {
		t.Fatal(e)
	}
	if e = s.Transition(a.ID, "Succeeded", "done", ""); e != nil {
		t.Fatal(e)
	}
	s.Transition(a.ID, "Succeeded", "done", "")
	q, _ = s.Quota("alpha")
	if q.ActiveTasks != 0 || q.UsedMemoryMiB != 0 {
		t.Fatal(q)
	}
	s.Transition(a.ID, "Running", "late event", "")
	a, _ = s.Get("alpha", a.ID)
	if a.State != "Succeeded" {
		t.Fatal("terminal resurrected")
	}
}
func TestRequestValidation(t *testing.T) {
	for _, modify := range []func(*Request){func(r *Request) { r.MemoryMiB = 0 }, func(r *Request) { r.Cores = 101 }, func(r *Request) { r.DurationSeconds = 0 }, func(r *Request) { r.DurationSeconds = 3601 }, func(r *Request) { r.Workload = "arbitrary-image" }, func(r *Request) { r.Name = "" }} {
		r := req()
		modify(&r)
		if r.Validate() == nil {
			t.Fatalf("accepted %+v", r)
		}
	}
}
func TestBackendBinding(t *testing.T) {
	s := newStore(t)
	if e := s.BindBackend("k8s-sim|cluster-a|ns"); e != nil {
		t.Fatal(e)
	}
	if e := s.BindBackend("k8s-sim|cluster-a|ns"); e != nil {
		t.Fatal(e)
	}
	if e := s.BindBackend("k8s-sim|cluster-b|ns"); e == nil {
		t.Fatal("allowed existing ledger to move to a different cluster")
	}
}
func TestStoreRelativePath(t *testing.T) {
	t.Chdir(t.TempDir())
	s, e := Open("data/state.db")
	if e != nil {
		t.Fatal(e)
	}
	s.Close()
}
