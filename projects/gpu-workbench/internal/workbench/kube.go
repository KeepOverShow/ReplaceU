package workbench

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	"strconv"
	"time"
)

const taskLabel = "workbench.local/task"
const allocatedAnnotation = "scheduling.koordinator.sh/device-allocated"

type Kube struct {
	Client                 kubernetes.Interface
	Dynamic                dynamic.Interface
	Namespace, Mode, Image string
}

func ptr[T any](v T) *T { return &v }
func requestHash(t Task) string {
	b, _ := json.Marshal(t.Request)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func (k *Kube) job(t Task, suspend bool) *batch.Job {
	labels := map[string]string{"app.kubernetes.io/managed-by": "gpu-workbench", taskLabel: t.ID, "workbench.local/team": t.TeamID}
	podLabels := map[string]string{}
	for a, b := range labels {
		podLabels[a] = b
	}
	podLabels["koordinator.sh/qosClass"] = "LS"
	if t.Request.Profile == "batch" {
		podLabels["koordinator.sh/qosClass"] = "BE"
	}
	resources := core.ResourceList{core.ResourceCPU: resource.MustParse("100m"), core.ResourceMemory: resource.MustParse("96Mi")}
	limits := core.ResourceList{core.ResourceCPU: resource.MustParse("500m"), core.ResourceMemory: resource.MustParse("256Mi")}
	scheduler := "default-scheduler"
	if k.Mode == "k8s-sim" || k.Mode == "k8s-koord-hami" {
		scheduler = "koord-scheduler"
		resources["koordinator.sh/gpu.shared"] = resource.MustParse("1")
		resources["koordinator.sh/gpu-core"] = resource.MustParse(strconv.Itoa(t.Request.Cores))
		resources["koordinator.sh/gpu-memory"] = resource.MustParse(fmt.Sprintf("%dMi", t.Request.MemoryMiB))
		for name, q := range resources {
			if name != core.ResourceCPU && name != core.ResourceMemory {
				limits[name] = q
			}
		}
	}
	if k.Mode == "k8s-sim" {
		podLabels["workbench.local/simulated"] = "true"
	}
	if k.Mode == "k8s-koord-hami" {
		podLabels["koordinator.sh/gpu-isolation-provider"] = "HAMi-core"
	}
	mode := k.Mode
	env := []core.EnvVar{{Name: "WORKBENCH_MODE", Value: mode}, {Name: "MEMORY_MIB", Value: strconv.Itoa(t.Request.MemoryMiB)}, {Name: "CORES", Value: strconv.Itoa(t.Request.Cores)}, {Name: "DURATION_SECONDS", Value: strconv.Itoa(t.Request.DurationSeconds)}, {Name: "WORKLOAD", Value: t.Request.Workload}, {Name: "DEVICE_ALLOCATION", ValueFrom: &core.EnvVarSource{FieldRef: &core.ObjectFieldSelector{FieldPath: "metadata.annotations['" + allocatedAnnotation + "']"}}}}
	image := k.Image
	if image == "" {
		image = "python:3.12.11-slim-bookworm"
	}
	return &batch.Job{ObjectMeta: meta.ObjectMeta{Name: "gwb-" + t.ID, Namespace: k.Namespace, Labels: labels, Annotations: map[string]string{"workbench.local/request-hash": requestHash(t)}}, Spec: batch.JobSpec{Suspend: ptr(suspend), Parallelism: ptr(int32(1)), Completions: ptr(int32(1)), BackoffLimit: ptr(int32(0)), ActiveDeadlineSeconds: ptr(int64(t.Request.DurationSeconds + 300)), Template: core.PodTemplateSpec{ObjectMeta: meta.ObjectMeta{Labels: podLabels}, Spec: core.PodSpec{SchedulerName: scheduler, RestartPolicy: core.RestartPolicyNever, TerminationGracePeriodSeconds: ptr(int64(5)), AutomountServiceAccountToken: ptr(false), SecurityContext: &core.PodSecurityContext{RunAsNonRoot: ptr(true), RunAsUser: ptr(int64(10001)), RunAsGroup: ptr(int64(10001)), SeccompProfile: &core.SeccompProfile{Type: core.SeccompProfileTypeRuntimeDefault}}, Containers: []core.Container{{Name: "worker", Image: image, ImagePullPolicy: core.PullIfNotPresent, Command: []string{"python3", "-u", "/app/worker.py"}, Env: env, Resources: core.ResourceRequirements{Requests: resources, Limits: limits}, SecurityContext: &core.SecurityContext{AllowPrivilegeEscalation: ptr(false), ReadOnlyRootFilesystem: ptr(true), Capabilities: &core.Capabilities{Drop: []core.Capability{"ALL"}}}, VolumeMounts: []core.VolumeMount{{Name: "code", MountPath: "/app", ReadOnly: true}, {Name: "tmp", MountPath: "/tmp"}}}}, Volumes: []core.Volume{{Name: "code", VolumeSource: core.VolumeSource{ConfigMap: &core.ConfigMapVolumeSource{LocalObjectReference: core.LocalObjectReference{Name: "workbench-worker"}}}}, {Name: "tmp", VolumeSource: core.VolumeSource{EmptyDir: &core.EmptyDirVolumeSource{SizeLimit: ptr(resource.MustParse("32Mi"))}}}}}}}}
}
func (k *Kube) owned(j *batch.Job, t Task) error {
	if j.Labels["app.kubernetes.io/managed-by"] != "gpu-workbench" || j.Labels[taskLabel] != t.ID || j.Labels["workbench.local/team"] != t.TeamID || j.Annotations["workbench.local/request-hash"] != requestHash(t) {
		return errors.New("执行对象归属或请求摘要不匹配")
	}
	if t.JobUID != "" && string(j.UID) != t.JobUID && j.Annotations["workbench.local/cancelled-uid"] != t.JobUID {
		return errors.New("执行对象 UID 已改变")
	}
	return nil
}
func (k *Kube) Ensure(ctx context.Context, t Task) error {
	j, e := k.Client.BatchV1().Jobs(k.Namespace).Get(ctx, "gwb-"+t.ID, meta.GetOptions{})
	if e == nil {
		return k.owned(j, t)
	}
	if !apierrors.IsNotFound(e) {
		return e
	}
	_, e = k.Client.BatchV1().Jobs(k.Namespace).Create(ctx, k.job(t, false), meta.CreateOptions{})
	if apierrors.IsAlreadyExists(e) {
		j, e = k.Client.BatchV1().Jobs(k.Namespace).Get(ctx, "gwb-"+t.ID, meta.GetOptions{})
		if e != nil {
			return e
		}
		return k.owned(j, t)
	}
	return e
}
func (k *Kube) pods(ctx context.Context, t Task) ([]core.Pod, error) {
	list, e := k.Client.CoreV1().Pods(k.Namespace).List(ctx, meta.ListOptions{LabelSelector: taskLabel + "=" + t.ID})
	if e != nil {
		return nil, e
	}
	return list.Items, nil
}
func podOwner(p core.Pod, t Task, j *batch.Job) bool {
	for _, o := range p.OwnerReferences {
		if o.Kind == "Job" && o.Name == "gwb-"+t.ID {
			if j != nil && o.UID == j.UID {
				return true
			}
			if t.JobUID != "" && string(o.UID) == t.JobUID {
				return true
			}
		}
	}
	return false
}
func live(p core.Pod) bool {
	return p.Status.Phase != core.PodSucceeded && p.Status.Phase != core.PodFailed
}
func (k *Kube) Observe(ctx context.Context, t Task) (Observation, error) {
	j, e := k.Client.BatchV1().Jobs(k.Namespace).Get(ctx, "gwb-"+t.ID, meta.GetOptions{})
	if e != nil && !apierrors.IsNotFound(e) {
		return Observation{}, e
	}
	if apierrors.IsNotFound(e) {
		j = nil
	}
	if j != nil {
		if e = k.owned(j, t); e != nil {
			return Observation{}, e
		}
	}
	pods, e := k.pods(ctx, t)
	if e != nil {
		return Observation{}, e
	}
	o := Observation{State: "Missing", Reason: "没有执行对象"}
	if j != nil {
		o.UID = string(j.UID)
		o.State = "Pending"
		o.Reason = "等待 Pod 创建或调度"
	}
	active := 0
	for _, p := range pods {
		if !podOwner(p, t, j) {
			return Observation{}, errors.New("同任务标签下存在归属不明的 Pod，保留额度")
		}
		o.Node = p.Spec.NodeName
		if raw := p.Annotations[allocatedAnnotation]; raw != "" {
			var a any
			if json.Unmarshal([]byte(raw), &a) == nil {
				o.Allocation = a
			}
		}
		if !live(p) {
			continue
		}
		active++
		if p.Status.Phase == core.PodRunning {
			o.State = "Running"
			o.Reason = "工作负载正在运行"
		} else {
			o.State = "Pending"
			for _, c := range p.Status.Conditions {
				if c.Type == core.PodScheduled && c.Status == core.ConditionFalse {
					o.Reason = c.Reason + ": " + c.Message
				}
			}
			for _, c := range p.Status.ContainerStatuses {
				if c.State.Waiting != nil {
					o.Reason = c.State.Waiting.Reason + ": " + c.State.Waiting.Message
				}
			}
		}
	}
	if j == nil {
		if active > 0 {
			o.State = "Cleaning"
			o.Reason = "Job 已消失，但关联 Pod 仍存活"
		}
		return o, nil
	}
	if j.Spec.Suspend != nil && *j.Spec.Suspend {
		// A persisted intent is not an acknowledgement: an earlier reconcile
		// may still create a Pod. Terminal Jobs need not acquire Suspended.
		acknowledged := false
		for _, c := range j.Status.Conditions {
			if c.Status == core.ConditionTrue && (c.Type == batch.JobSuspended || c.Type == batch.JobComplete || c.Type == batch.JobFailed) {
				acknowledged = true
			}
		}
		o.State = "Stopping"
		o.Reason = "等待 Job 控制器确认暂停及 Pod 停止"
		if acknowledged && active == 0 {
			o.State = "Cancelled"
			o.Reason = "控制器已确认停止且无活动 Pod"
		}
		return o, nil
	}
	if active == 0 {
		for _, c := range j.Status.Conditions {
			if c.Status == core.ConditionTrue {
				if c.Type == batch.JobComplete {
					o.State = "Succeeded"
					o.Reason = "Job 完成，已确认无活动 Pod"
				}
				if c.Type == batch.JobFailed {
					o.State = "Failed"
					o.Reason = c.Reason + ": " + c.Message
				}
			}
		}
	}
	return o, nil
}
func (k *Kube) Cancel(ctx context.Context, t Task) error {
	e := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		j, e := k.Client.BatchV1().Jobs(k.Namespace).Get(ctx, "gwb-"+t.ID, meta.GetOptions{})
		if apierrors.IsNotFound(e) {
			j = k.job(t, true)
			j.Annotations["workbench.local/cancelled-uid"] = t.JobUID
			_, e = k.Client.BatchV1().Jobs(k.Namespace).Create(ctx, j, meta.CreateOptions{})
			if apierrors.IsAlreadyExists(e) {
				return apierrors.NewConflict(schema.GroupResource{Resource: "jobs"}, j.Name, e)
			}
			return e
		}
		if e != nil {
			return e
		}
		if e = k.owned(j, t); e != nil {
			return e
		}
		if j.Spec.Suspend != nil && *j.Spec.Suspend {
			return nil
		}
		j.Spec.Suspend = ptr(true)
		_, e = k.Client.BatchV1().Jobs(k.Namespace).Update(ctx, j, meta.UpdateOptions{})
		return e
	})
	if e != nil {
		return e
	}
	j, e := k.Client.BatchV1().Jobs(k.Namespace).Get(ctx, "gwb-"+t.ID, meta.GetOptions{})
	if e != nil {
		return e
	}
	pods, e := k.pods(ctx, t)
	if e != nil {
		return e
	}
	for _, p := range pods {
		if !podOwner(p, t, j) {
			return errors.New("Pod 归属不明，停止清理")
		}
		orphan := false
		for _, o := range p.OwnerReferences {
			if o.Kind == "Job" && o.UID != j.UID {
				orphan = true
			}
		}
		if orphan && live(p) {
			e = k.Client.CoreV1().Pods(k.Namespace).Delete(ctx, p.Name, meta.DeleteOptions{Preconditions: &meta.Preconditions{UID: &p.UID}})
			if e != nil && !apierrors.IsNotFound(e) {
				return e
			}
		}
	}
	return nil
}
func (k *Kube) Logs(ctx context.Context, t Task) (string, error) {
	j, e := k.Client.BatchV1().Jobs(k.Namespace).Get(ctx, "gwb-"+t.ID, meta.GetOptions{})
	if e != nil {
		return "", e
	}
	if e = k.owned(j, t); e != nil {
		return "", e
	}
	pods, e := k.pods(ctx, t)
	if e != nil {
		return "", e
	}
	out := ""
	for _, p := range pods {
		if !podOwner(p, t, j) {
			return "", errors.New("Pod 归属不匹配")
		}
		r, e := k.Client.CoreV1().Pods(k.Namespace).GetLogs(p.Name, &core.PodLogOptions{Container: "worker", LimitBytes: ptr(int64(32000)), TailLines: ptr(int64(200))}).Stream(ctx)
		if e != nil {
			return out, e
		}
		b, e := io.ReadAll(io.LimitReader(r, 32000))
		r.Close()
		if e != nil {
			return out, e
		}
		out += "pod=" + p.Name + "\n" + string(b)
		if len(out) > 64000 {
			break
		}
	}
	return out, nil
}
func (k *Kube) Devices(ctx context.Context) ([]Device, error) {
	out := []Device{}
	if k.Dynamic == nil {
		return out, errors.New("Device CR 客户端不可用")
	}
	list, e := k.Dynamic.Resource(schema.GroupVersionResource{Group: "scheduling.koordinator.sh", Version: "v1alpha1", Resource: "devices"}).List(ctx, meta.ListOptions{})
	if e != nil {
		return out, e
	}
	for _, item := range list.Items {
		spec, _ := item.Object["spec"].(map[string]any)
		devices, _ := spec["devices"].([]any)
		for _, raw := range devices {
			v, ok := raw.(map[string]any)
			if !ok || v["type"] != "gpu" {
				continue
			}
			rs, _ := v["resources"].(map[string]any)
			mem, _ := rs["koordinator.sh/gpu-memory"].(string)
			q, e := resource.ParseQuantity(mem)
			if e != nil {
				continue
			}
			healthy, _ := v["health"].(bool)
			id, _ := v["id"].(string)
			source := "Koordinator Device CR"
			model := "GPU"
			if item.GetLabels()["workbench.local/simulated"] == "true" {
				source += " (simulated)"
				model = "模拟 GPU"
			}
			out = append(out, Device{Node: item.GetName(), ID: id, Model: model, MemoryMiB: q.Value() / (1024 * 1024), Healthy: healthy, Source: source, ObservedAt: time.Now().UnixMilli()})
		}
	}
	return out, nil
}
