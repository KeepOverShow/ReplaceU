package workbench

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Demo struct {
	store *Store
	now   func() time.Time
}

func NewDemo(s *Store) *Demo { return &Demo{store: s, now: time.Now} }
func (d *Demo) Ensure(ctx context.Context, t Task) error {
	r, e := d.store.db.ExecContext(ctx, `INSERT OR IGNORE INTO demo_runs(task_id,start_at,scenario) VALUES(?,?,?)`, t.ID, d.now().UnixMilli(), t.Request.Scenario)
	if e != nil {
		return e
	}
	n, e := r.RowsAffected()
	if e != nil {
		return e
	}
	if n == 1 && t.Request.Scenario == "create-timeout" {
		return errors.New("模拟：创建成功后连接超时")
	}
	return nil
}
func (d *Demo) Observe(ctx context.Context, t Task) (Observation, error) {
	var start, cancel int64
	var scenario string
	e := d.store.db.QueryRowContext(ctx, `SELECT start_at,cancel_at,scenario FROM demo_runs WHERE task_id=?`, t.ID).Scan(&start, &cancel, &scenario)
	if errors.Is(e, sql.ErrNoRows) {
		return Observation{State: "Missing"}, nil
	}
	if e != nil {
		return Observation{}, e
	}
	o := Observation{State: "Running", UID: "demo-" + t.ID, Reason: "模拟后端执行中；未执行 GPU 计算"}
	if cancel > 0 {
		if scenario == "cancel-delay" && d.now().UnixMilli()-cancel < 3000 {
			o.State = "Stopping"
		} else {
			o.State = "Cancelled"
		}
		return o, nil
	}
	if scenario == "observe-error" {
		return Observation{}, errors.New("模拟：观察接口不可达")
	}
	if d.now().UnixMilli()-start >= int64(t.Request.DurationSeconds)*1000 {
		o.State = "Succeeded"
		if t.Request.Workload == "fail" {
			o.State = "Failed"
		}
		o.Reason = "模拟执行结束"
	}
	return o, nil
}
func (d *Demo) Cancel(ctx context.Context, t Task) error {
	// A persistent tombstone prevents Ensure from starting a cancelled task.
	_, e := d.store.db.ExecContext(ctx, `INSERT INTO demo_runs(task_id,start_at,cancel_at,scenario) VALUES(?,?,?,?) ON CONFLICT(task_id) DO UPDATE SET cancel_at=CASE WHEN cancel_at=0 THEN excluded.cancel_at ELSE cancel_at END`, t.ID, d.now().UnixMilli(), d.now().UnixMilli(), t.Request.Scenario)
	return e
}
func (d *Demo) Logs(ctx context.Context, t Task) (string, error) {
	o, e := d.Observe(ctx, t)
	return fmt.Sprintf("mode=local-demo simulated=true task=%s state=%s\n无 GPU 性能实测数据\n", t.ID, o.State), e
}
func (d *Demo) Devices(context.Context) ([]Device, error) {
	return []Device{{Node: "demo-node", ID: "SIM-0", Model: "模拟 GPU", MemoryMiB: 8192, Healthy: true, Source: "local-demo (simulated)", ObservedAt: d.now().UnixMilli()}}, nil
}
