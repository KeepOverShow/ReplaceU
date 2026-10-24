package workbench

import (
	"context"
	"errors"
	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

func ktask() Task { return Task{ID: "abc123", TeamID: "alpha", Request: req(), State: "Submitting"} }
func TestKubeTemplate(t *testing.T) {
	k := &Kube{Mode: "k8s-sim", Namespace: "gpu-workbench", Image: "python:3.12.11-slim"}
	j := k.job(ktask(), false)
	p := j.Spec.Template.Spec
	if len(p.Containers) != 1 {
		t.Fatal("missing workload container")
	}
	q := p.Containers[0].Resources.Requests["koordinator.sh/gpu-memory"]
	if p.SchedulerName != "koord-scheduler" || q.String() != "1Gi" {
		t.Fatalf("bad template %+v", p)
	}
	if j.Spec.Template.Labels["workbench.local/simulated"] != "true" {
		t.Fatal("missing simulation label")
	}
	if j.Spec.Template.Labels["koordinator.sh/gpu-isolation-provider"] == "HAMi-core" {
		t.Fatal("simulation must not request real runtime injection")
	}
	k.Mode = "k8s-koord-hami"
	j = k.job(ktask(), false)
	if j.Spec.Template.Labels["koordinator.sh/gpu-isolation-provider"] != "HAMi-core" {
		t.Fatal("missing runtime provider")
	}
}
func TestKubeOwnership(t *testing.T) {
	ctx := context.Background()
	f := fake.NewSimpleClientset(&batch.Job{ObjectMeta: meta.ObjectMeta{Name: "gwb-abc123", Namespace: "gpu-workbench"}})
	k := &Kube{Client: f, Namespace: "gpu-workbench"}
	if _, e := k.Observe(ctx, ktask()); e == nil {
		t.Fatal("accepted foreign job")
	}
	if e := k.Cancel(ctx, ktask()); e == nil {
		t.Fatal("cancelled foreign job")
	}
}
func TestKubeCancelWaitsForPods(t *testing.T) {
	ctx := context.Background()
	task := ktask()
	k := &Kube{Client: fake.NewSimpleClientset(), Mode: "k8s-sim", Namespace: "gpu-workbench", Image: "python:3.12.11-slim"}
	j := k.job(task, false)
	j.UID = types.UID("uid-1")
	task.JobUID = "uid-1"
	k.Client.BatchV1().Jobs(k.Namespace).Create(ctx, j, meta.CreateOptions{})
	p := &core.Pod{ObjectMeta: meta.ObjectMeta{Name: "worker", Namespace: k.Namespace, Labels: j.Spec.Template.Labels, OwnerReferences: []meta.OwnerReference{{UID: j.UID, Kind: "Job", Name: j.Name}}}, Status: core.PodStatus{Phase: core.PodRunning}}
	k.Client.CoreV1().Pods(k.Namespace).Create(ctx, p, meta.CreateOptions{})
	if e := k.Cancel(ctx, task); e != nil {
		t.Fatal(e)
	}
	o, e := k.Observe(ctx, task)
	if e != nil || o.State != "Stopping" {
		t.Fatalf("premature completion %+v %v", o, e)
	}
	k.Client.CoreV1().Pods(k.Namespace).Delete(ctx, "worker", meta.DeleteOptions{})
	o, e = k.Observe(ctx, task)
	if e != nil || o.State != "Stopping" {
		t.Fatalf("released before controller acknowledged suspension %+v %v", o, e)
	}
	ack, err := k.Client.BatchV1().Jobs(k.Namespace).Get(ctx, j.Name, meta.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	ack.Status.Conditions = []batch.JobCondition{{Type: batch.JobSuspended, Status: core.ConditionTrue}}
	if _, err = k.Client.BatchV1().Jobs(k.Namespace).UpdateStatus(ctx, ack, meta.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	// A Pod created by an earlier reconcile must still retain the reservation.
	if _, err = k.Client.CoreV1().Pods(k.Namespace).Create(ctx, p, meta.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	o, e = k.Observe(ctx, task)
	if e != nil || o.State != "Stopping" {
		t.Fatalf("ignored late Pod %+v %v", o, e)
	}
	if err = k.Client.CoreV1().Pods(k.Namespace).Delete(ctx, p.Name, meta.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	o, e = k.Observe(ctx, task)
	if e != nil || o.State != "Cancelled" {
		t.Fatalf("not stopped %+v %v", o, e)
	}
	// A late/retried create cannot replace the suspended tombstone.
	if e = k.Ensure(ctx, task); e != nil {
		t.Fatal(e)
	}
	stored, _ := k.Client.BatchV1().Jobs(k.Namespace).Get(ctx, j.Name, meta.GetOptions{})
	if stored.Spec.Suspend == nil || !*stored.Spec.Suspend {
		t.Fatal("restarted cancelled job")
	}
}
func TestKubeMissingJobWithLivePod(t *testing.T) {
	ctx := context.Background()
	task := ktask()
	task.JobUID = "gone"
	k := &Kube{Client: fake.NewSimpleClientset(), Namespace: "gpu-workbench"}
	p := &core.Pod{ObjectMeta: meta.ObjectMeta{Name: "orphan", Namespace: k.Namespace, Labels: map[string]string{"workbench.local/task": task.ID}, OwnerReferences: []meta.OwnerReference{{UID: "gone", Kind: "Job", Name: "gwb-" + task.ID}}}, Status: core.PodStatus{Phase: core.PodRunning}}
	k.Client.CoreV1().Pods(k.Namespace).Create(ctx, p, meta.CreateOptions{})
	o, e := k.Observe(ctx, task)
	if e != nil || o.State == "Missing" || Terminal(o.State) {
		t.Fatalf("orphan lost %+v %v", o, e)
	}
}

var _ = errors.New

func TestKubeRetainsCompletedAllocation(t *testing.T) {
	ctx := context.Background()
	task := ktask()
	k := &Kube{Client: fake.NewSimpleClientset(), Namespace: "gpu-workbench"}
	j := k.job(task, false)
	j.UID = "finished"
	j.Status.Conditions = []batch.JobCondition{{Type: batch.JobComplete, Status: core.ConditionTrue}}
	k.Client.BatchV1().Jobs(k.Namespace).Create(ctx, j, meta.CreateOptions{})
	p := &core.Pod{ObjectMeta: meta.ObjectMeta{Name: "finished-pod", Namespace: k.Namespace, Labels: j.Spec.Template.Labels, Annotations: map[string]string{allocatedAnnotation: `{"gpu":[{"minor":0}]}`}, OwnerReferences: []meta.OwnerReference{{UID: j.UID, Kind: "Job", Name: j.Name}}}, Spec: core.PodSpec{NodeName: "node"}, Status: core.PodStatus{Phase: core.PodSucceeded}}
	k.Client.CoreV1().Pods(k.Namespace).Create(ctx, p, meta.CreateOptions{})
	o, e := k.Observe(ctx, task)
	if e != nil || o.State != "Succeeded" || o.Allocation == nil || o.Node != "node" {
		t.Fatalf("lost allocation %+v %v", o, e)
	}
}
