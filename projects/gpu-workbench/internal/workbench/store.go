package workbench

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

type Store struct{ db *sql.DB }

func Open(path string) (*Store, error) {
	var err error
	path, err = filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	u := url.URL{Scheme: "file", Path: path}
	db, err := sql.Open("sqlite3", u.String()+"?_busy_timeout=5000&_journal_mode=WAL&_foreign_keys=on&_txlock=immediate")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`
 CREATE TABLE IF NOT EXISTS teams(id TEXT PRIMARY KEY, token_hash TEXT UNIQUE NOT NULL, max_tasks INTEGER NOT NULL,max_memory INTEGER NOT NULL,max_cores INTEGER NOT NULL);
 CREATE TABLE IF NOT EXISTS tasks(id TEXT PRIMARY KEY,team_id TEXT NOT NULL REFERENCES teams(id),idem_key TEXT NOT NULL,request TEXT NOT NULL,state TEXT NOT NULL,reason TEXT NOT NULL,created_at INTEGER NOT NULL,updated_at INTEGER NOT NULL,job_uid TEXT NOT NULL DEFAULT '',UNIQUE(team_id,idem_key));
 CREATE TABLE IF NOT EXISTS reservations(task_id TEXT PRIMARY KEY REFERENCES tasks(id),team_id TEXT NOT NULL,memory INTEGER NOT NULL,cores INTEGER NOT NULL,released INTEGER NOT NULL DEFAULT 0);
 CREATE TABLE IF NOT EXISTS events(id INTEGER PRIMARY KEY AUTOINCREMENT,task_id TEXT NOT NULL REFERENCES tasks(id),time INTEGER NOT NULL,state TEXT NOT NULL,message TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS demo_runs(task_id TEXT PRIMARY KEY,start_at INTEGER NOT NULL,cancel_at INTEGER NOT NULL DEFAULT 0,scenario TEXT NOT NULL,create_failed INTEGER NOT NULL DEFAULT 0);
 CREATE INDEX IF NOT EXISTS task_team_time ON tasks(team_id,created_at);
 CREATE INDEX IF NOT EXISTS reservation_team ON reservations(team_id,released);
 `)
	if err != nil {
		db.Close()
		return nil, err
	}
	if err = os.Chmod(path, 0600); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}
func (s *Store) Close() error { return s.db.Close() }
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
func (s *Store) ConfigureTeam(t Team, token string) error {
	if t.ID == "" || token == "" || t.MaxTasks < 1 || t.MaxMemoryMiB < 128 || t.MaxCores < 1 {
		return fmt.Errorf("invalid team configuration")
	}
	_, e := s.db.Exec(`INSERT INTO teams VALUES(?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET token_hash=excluded.token_hash,max_tasks=excluded.max_tasks,max_memory=excluded.max_memory,max_cores=excluded.max_cores`, t.ID, hashToken(token), t.MaxTasks, t.MaxMemoryMiB, t.MaxCores)
	return e
}
func (s *Store) Authenticate(token string) (string, error) {
	var team string
	if token == "" {
		return "", ErrUnauthorized
	}
	e := s.db.QueryRow(`SELECT id FROM teams WHERE token_hash=?`, hashToken(token)).Scan(&team)
	if errors.Is(e, sql.ErrNoRows) {
		return "", ErrUnauthorized
	}
	return team, e
}

const taskColumns = "id,team_id,request,state,reason,created_at,updated_at,job_uid"

type scanner interface{ Scan(...any) error }

func scanTask(row scanner) (Task, error) {
	var t Task
	var raw string
	e := row.Scan(&t.ID, &t.TeamID, &raw, &t.State, &t.Reason, &t.CreatedAt, &t.UpdatedAt, &t.JobUID)
	if errors.Is(e, sql.ErrNoRows) {
		return t, ErrNotFound
	}
	if e != nil {
		return t, e
	}
	e = json.Unmarshal([]byte(raw), &t.Request)
	return t, e
}
func (s *Store) Get(team, id string) (Task, error) {
	return scanTask(s.db.QueryRow(`SELECT `+taskColumns+` FROM tasks WHERE id=? AND team_id=?`, id, team))
}
func (s *Store) Submit(team, key string, r Request) (Task, bool, error) {
	r = r.Normalized()
	if e := r.Validate(); e != nil {
		return Task{}, false, e
	}
	if len(key) < 1 || len(key) > 128 {
		return Task{}, false, fmt.Errorf("Idempotency-Key must contain 1-128 bytes")
	}
	raw, _ := json.Marshal(r)
	tx, e := s.db.Begin()
	if e != nil {
		return Task{}, false, e
	}
	defer tx.Rollback()
	previous, e := scanTask(tx.QueryRow(`SELECT `+taskColumns+` FROM tasks WHERE team_id=? AND idem_key=?`, team, key))
	if e == nil {
		before, _ := json.Marshal(previous.Request)
		if string(before) != string(raw) {
			return Task{}, false, ErrConflict
		}
		return previous, true, nil
	}
	if !errors.Is(e, ErrNotFound) {
		return Task{}, false, e
	}
	var maxTasks, maxMem, maxCores, usedTasks, usedMem, usedCores int
	e = tx.QueryRow(`SELECT max_tasks,max_memory,max_cores FROM teams WHERE id=?`, team).Scan(&maxTasks, &maxMem, &maxCores)
	if e != nil {
		return Task{}, false, e
	}
	e = tx.QueryRow(`SELECT COUNT(*),COALESCE(SUM(memory),0),COALESCE(SUM(cores),0) FROM reservations WHERE team_id=? AND released=0`, team).Scan(&usedTasks, &usedMem, &usedCores)
	if e != nil {
		return Task{}, false, e
	}
	if usedTasks+1 > maxTasks || r.MemoryMiB > maxMem-usedMem || r.Cores > maxCores-usedCores {
		return Task{}, false, ErrQuota
	}
	b := make([]byte, 12)
	if _, e = rand.Read(b); e != nil {
		return Task{}, false, e
	}
	now := time.Now().UnixMilli()
	t := Task{ID: hex.EncodeToString(b), TeamID: team, Request: r, State: "Queued", Reason: "已持久化并预留团队额度", CreatedAt: now, UpdatedAt: now}
	if _, e = tx.Exec(`INSERT INTO tasks(id,team_id,idem_key,request,state,reason,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`, t.ID, team, key, string(raw), t.State, t.Reason, now, now); e != nil {
		return Task{}, false, e
	}
	if _, e = tx.Exec(`INSERT INTO reservations(task_id,team_id,memory,cores) VALUES(?,?,?,?)`, t.ID, team, r.MemoryMiB, r.Cores); e != nil {
		return Task{}, false, e
	}
	if _, e = tx.Exec(`INSERT INTO events(task_id,time,state,message) VALUES(?,?,?,?)`, t.ID, now, t.State, t.Reason); e != nil {
		return Task{}, false, e
	}
	return t, false, tx.Commit()
}
func (s *Store) Quota(team string) (Quota, error) {
	tx, e := s.db.Begin()
	if e != nil {
		return Quota{}, e
	}
	defer tx.Rollback()
	q := Quota{}
	e = tx.QueryRow(`SELECT id,max_tasks,max_memory,max_cores FROM teams WHERE id=?`, team).Scan(&q.ID, &q.MaxTasks, &q.MaxMemoryMiB, &q.MaxCores)
	if e != nil {
		return q, e
	}
	e = tx.QueryRow(`SELECT COUNT(*),COALESCE(SUM(memory),0),COALESCE(SUM(cores),0) FROM reservations WHERE team_id=? AND released=0`, team).Scan(&q.ActiveTasks, &q.UsedMemoryMiB, &q.UsedCores)
	return q, e
}
func (s *Store) Tasks(team string, limit, offset int) ([]Task, error) {
	if limit < 1 || limit > 200 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return s.queryTasks(`SELECT `+taskColumns+` FROM tasks WHERE team_id=? ORDER BY created_at DESC,id LIMIT ? OFFSET ?`, team, limit, offset)
}
func (s *Store) Active() ([]Task, error) {
	return s.queryTasks(`SELECT ` + taskColumns + ` FROM tasks WHERE state NOT IN ('Succeeded','Failed','Cancelled') ORDER BY created_at`)
}
func (s *Store) queryTasks(query string, args ...any) ([]Task, error) {
	rows, e := s.db.Query(query, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Task{}
	for rows.Next() {
		t, e := scanTask(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
func (s *Store) Cancel(team, id string) error {
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	t, e := scanTask(tx.QueryRow(`SELECT `+taskColumns+` FROM tasks WHERE id=? AND team_id=?`, id, team))
	if e != nil {
		return e
	}
	if Terminal(t.State) || t.State == "Cancelling" {
		return nil
	}
	now := time.Now().UnixMilli()
	reason := "已记录取消意图，等待执行后端确认停止"
	if _, e = tx.Exec(`UPDATE tasks SET state='Cancelling',reason=?,updated_at=? WHERE id=?`, reason, now, id); e != nil {
		return e
	}
	if _, e = tx.Exec(`INSERT INTO events(task_id,time,state,message) VALUES(?,?,?,?)`, id, now, "Cancelling", reason); e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Store) Transition(id, state, reason, uid string) error {
	switch state {
	case "Queued", "Submitting", "Pending", "Running", "Cancelling", "Cleaning", "Succeeded", "Failed", "Cancelled":
	default:
		return fmt.Errorf("invalid state %q", state)
	}
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	t, e := scanTask(tx.QueryRow(`SELECT `+taskColumns+` FROM tasks WHERE id=?`, id))
	if e != nil {
		return e
	}
	if Terminal(t.State) {
		return nil
	}
	if t.State == "Cancelling" && state != "Cancelled" && state != "Cancelling" {
		return nil
	}
	if t.State == state && t.Reason == reason && (uid == "" || uid == t.JobUID) {
		return nil
	}
	if uid == "" {
		uid = t.JobUID
	}
	now := time.Now().UnixMilli()
	if _, e = tx.Exec(`UPDATE tasks SET state=?,reason=?,updated_at=?,job_uid=? WHERE id=?`, state, reason, now, uid, id); e != nil {
		return e
	}
	if Terminal(state) {
		if _, e = tx.Exec(`UPDATE reservations SET released=1 WHERE task_id=?`, id); e != nil {
			return e
		}
	}
	if _, e = tx.Exec(`INSERT INTO events(task_id,time,state,message) VALUES(?,?,?,?)`, id, now, state, reason); e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Store) Events(team, id string) ([]Event, error) {
	if _, e := s.Get(team, id); e != nil {
		return nil, e
	}
	rows, e := s.db.Query(`SELECT id,task_id,time,state,message FROM (SELECT * FROM events WHERE task_id=? ORDER BY id DESC LIMIT 200) ORDER BY id`, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Event{}
	for rows.Next() {
		var v Event
		if e = rows.Scan(&v.ID, &v.TaskID, &v.Time, &v.State, &v.Message); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) Ping() error { return s.db.Ping() }
func (s *Store) BindBackend(identity string) error {
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = tx.Exec(`CREATE TABLE IF NOT EXISTS backend_identity (id INTEGER PRIMARY KEY CHECK(id=1),identity TEXT NOT NULL)`); e != nil {
		return e
	}
	if _, e = tx.Exec(`INSERT OR IGNORE INTO backend_identity VALUES(1,?)`, identity); e != nil {
		return e
	}
	var existing string
	if e = tx.QueryRow(`SELECT identity FROM backend_identity WHERE id=1`).Scan(&existing); e != nil {
		return e
	}
	if existing != identity {
		return fmt.Errorf("backend identity changed; use a separate data directory")
	}
	return tx.Commit()
}
