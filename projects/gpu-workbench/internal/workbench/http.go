package workbench

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//go:embed web/*
var webFiles embed.FS

func NewHandler(s *Store, b Backend, mode string) http.Handler {
	assets, _ := fs.Sub(webFiles, "web")
	files := http.FileServer(http.FS(assets))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Cache-Control", "no-store")
		if r.URL.Path == "/healthz" {
			w.Write([]byte("ok\n"))
			return
		}
		if r.URL.Path == "/readyz" {
			if e := s.Ping(); e != nil {
				http.Error(w, "database unavailable", 503)
				return
			}
			w.Write([]byte("ok\n"))
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/metrics" {
			files.ServeHTTP(w, r)
			return
		}
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token == r.Header.Get("Authorization") || token == "" {
			http.Error(w, "需要团队令牌", 401)
			return
		}
		team, e := s.Authenticate(token)
		if e != nil {
			http.Error(w, "令牌无效", 401)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
		defer cancel()
		r = r.WithContext(ctx)
		send := func(status int, v any) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(status)
			json.NewEncoder(w).Encode(v)
		}
		fail := func(e error) {
			status := 500
			message := "服务暂时不可用"
			switch {
			case errors.Is(e, ErrNotFound):
				status = 404
				message = "任务不存在"
			case errors.Is(e, ErrQuota):
				status = 409
				message = "团队额度不足"
			case errors.Is(e, ErrConflict):
				status = 409
				message = "幂等键已用于不同请求"
			}
			send(status, map[string]string{"error": message})
		}
		switch r.URL.Path {
		case "/api/info":
			if r.Method != "GET" {
				http.Error(w, "method", 405)
				return
			}
			send(200, map[string]any{"mode": mode, "simulated_gpu": mode != "k8s-koord-hami", "team": team, "device_usage": "unknown", "isolation_verified": false})
			return
		case "/api/quota":
			if r.Method != "GET" {
				http.Error(w, "method", 405)
				return
			}
			q, e := s.Quota(team)
			if e != nil {
				fail(e)
			} else {
				send(200, q)
			}
			return
		case "/api/devices":
			if r.Method != "GET" {
				http.Error(w, "method", 405)
				return
			}
			d, e := b.Devices(ctx)
			if e != nil {
				send(503, map[string]string{"error": "设备信息暂时不可用"})
			} else {
				send(200, map[string]any{"mode": mode, "devices": d, "usage": "unknown"})
			}
			return
		case "/metrics":
			if r.Method != "GET" {
				http.Error(w, "method", 405)
				return
			}
			q, e := s.Quota(team)
			if e != nil {
				fail(e)
				return
			}
			w.Header().Set("Content-Type", "text/plain; version=0.0.4")
			fmt.Fprintf(w, "workbench_info{mode=%q} 1\nworkbench_active_tasks{team=%q} %d\nworkbench_reserved_memory_mib{team=%q} %d\nworkbench_reserved_cores{team=%q} %d\n", mode, team, q.ActiveTasks, team, q.UsedMemoryMiB, team, q.UsedCores)
			return
		case "/api/tasks":
			if r.Method == "GET" {
				limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
				offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
				ts, e := s.Tasks(team, limit, offset)
				if e != nil {
					fail(e)
				} else {
					send(200, map[string]any{"tasks": ts, "mode": mode})
				}
				return
			}
			if r.Method != "POST" {
				http.Error(w, "method", 405)
				return
			}
			key := r.Header.Get("Idempotency-Key")
			if len(key) < 1 || len(key) > 128 {
				send(400, map[string]string{"error": "需要 1–128 字节 Idempotency-Key"})
				return
			}
			dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
			dec.DisallowUnknownFields()
			var req Request
			if e = dec.Decode(&req); e != nil {
				send(400, map[string]string{"error": "请求格式不正确或包含未知字段"})
				return
			}
			var extra any
			if dec.Decode(&extra) != io.EOF {
				send(400, map[string]string{"error": "只接受一个 JSON 对象"})
				return
			}
			req = req.Normalized()
			if e = req.Validate(); e != nil {
				send(400, map[string]string{"error": e.Error()})
				return
			}
			if mode != "local-demo" && req.Scenario != "normal" {
				send(400, map[string]string{"error": "此故障注入仅用于 local-demo；集群故障请使用实验脚本"})
				return
			}
			task, replay, e := s.Submit(team, key, req)
			if e != nil {
				fail(e)
				return
			}
			status := 201
			if replay {
				status = 200
			}
			send(status, map[string]any{"task": task, "replayed": replay, "mode": mode})
			return
		}
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) < 3 || len(parts) > 4 || parts[0] != "api" || parts[1] != "tasks" {
			http.NotFound(w, r)
			return
		}
		task, e := s.Get(team, parts[2])
		if e != nil {
			fail(e)
			return
		}
		action := ""
		if len(parts) == 4 {
			action = parts[3]
		}
		if action == "cancel" && r.Method == "POST" {
			if e = s.Cancel(team, task.ID); e != nil {
				fail(e)
				return
			}
			task, e = s.Get(team, task.ID)
			if e != nil {
				fail(e)
				return
			}
			send(202, map[string]any{"task": task})
			return
		}
		if r.Method != "GET" {
			http.Error(w, "method", 405)
			return
		}
		switch action {
		case "":
			o, e := b.Observe(ctx, task)
			value := map[string]any{"task": task, "mode": mode}
			if e != nil {
				value["observation_error"] = "后端观察失败，保留最后状态"
			} else {
				value["observation"] = o
			}
			send(200, value)
		case "events":
			events, e := s.Events(team, task.ID)
			if e != nil {
				fail(e)
			} else {
				send(200, map[string]any{"events": events})
			}
		case "logs":
			logs, e := b.Logs(ctx, task)
			if e != nil {
				send(503, map[string]string{"error": "日志尚不可用，请等待 Pod 启动或检查集群"})
			} else {
				send(200, map[string]string{"logs": logs})
			}
		default:
			http.NotFound(w, r)
		}
	})
}
