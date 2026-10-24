package workbench

import (
	"context"
	"fmt"
)

// Step is single-threaded. Backend uncertainty never releases a reservation.
func (c *Controller) Step(ctx context.Context) error {
	tasks, e := c.Store.Active()
	if e != nil {
		return e
	}
	for _, snapshot := range tasks {
		if e = ctx.Err(); e != nil {
			return e
		}
		t, e := c.Store.Get(snapshot.TeamID, snapshot.ID)
		if e != nil {
			return e
		}
		if Terminal(t.State) {
			continue
		}
		change := func(state, reason, uid string) error { return c.Store.Transition(t.ID, state, reason, uid) }
		if t.State == "Cancelling" || t.State == "Cleaning" {
			if e = c.Backend.Cancel(ctx, t); e != nil {
				if e = change(t.State, "停止结果未知，保留额度："+e.Error(), ""); e != nil {
					return e
				}
				continue
			}
			obs, e := c.Backend.Observe(ctx, t)
			if e != nil {
				if e = change(t.State, "等待停止确认："+e.Error(), ""); e != nil {
					return e
				}
				continue
			}
			if obs.State == "Missing" || Terminal(obs.State) {
				state := "Cancelled"
				if t.State == "Cleaning" {
					state = "Failed"
				}
				if e = change(state, "执行已停止，释放团队额度", obs.UID); e != nil {
					return e
				}
			}
			continue
		}
		obs, e := c.Backend.Observe(ctx, t)
		if e != nil {
			if e = change(t.State, "观察失败，保留额度："+e.Error(), ""); e != nil {
				return e
			}
			continue
		}
		if obs.State == "Missing" {
			if t.State != "Queued" && t.State != "Submitting" {
				if e = change("Cleaning", "原执行对象消失，清理残留后再释放", ""); e != nil {
					return e
				}
				continue
			}
			if e = change("Submitting", "向后端提交确定性名称的执行对象", ""); e != nil {
				return e
			}
			// Re-read cancellation intent immediately before the external action.
			t, e = c.Store.Get(t.TeamID, t.ID)
			if e != nil {
				return e
			}
			if t.State != "Submitting" {
				continue
			}
			if e = c.Backend.Ensure(ctx, t); e != nil {
				if e = change("Submitting", "创建结果待对账："+e.Error(), ""); e != nil {
					return e
				}
			}
			continue
		}
		switch obs.State {
		case "Pending", "Running", "Succeeded", "Failed", "Cleaning":
			if e = change(obs.State, obs.Reason, obs.UID); e != nil {
				return e
			}
		default:
			if e = change(t.State, fmt.Sprintf("后端状态 %s，保留额度", obs.State), obs.UID); e != nil {
				return e
			}
		}
	}
	return nil
}
