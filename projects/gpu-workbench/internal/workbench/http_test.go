package workbench

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPI(t *testing.T) {
	s := newStore(t)
	api := NewHandler(s, NewDemo(s), "local-demo")
	body := `{"name":"first","memory_mib":1024,"cores":50,"duration_seconds":10}`
	call := func(method, path, token, payload, key string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(payload))
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		r.Header.Set("Idempotency-Key", key)
		w := httptest.NewRecorder()
		api.ServeHTTP(w, r)
		return w
	}
	if w := call("GET", "/api/tasks", "", "", ""); w.Code != 401 {
		t.Fatal(w.Code)
	}
	if w := call("POST", "/api/tasks", "test-token-alpha", body, ""); w.Code != 400 {
		t.Fatalf("missing key %d %s", w.Code, w.Body)
	}
	w := call("POST", "/api/tasks", "test-token-alpha", body, "one")
	if w.Code != 201 {
		t.Fatalf("submit %d %s", w.Code, w.Body)
	}
	var a struct {
		Task Task `json:"task"`
	}
	json.Unmarshal(w.Body.Bytes(), &a)
	if a.Task.ID == "" {
		t.Fatal(w.Body)
	}
	w = call("POST", "/api/tasks", "test-token-alpha", body, "one")
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w = call("POST", "/api/tasks", "test-token-alpha", strings.Replace(body, "first", "other", 1), "one"); w.Code != 409 {
		t.Fatal(w.Code)
	}
	if w = call("GET", "/api/tasks/"+a.Task.ID, "test-token-beta", "", ""); w.Code != 404 {
		t.Fatal(w.Code)
	}
	if w = call("POST", "/api/tasks/"+a.Task.ID+"/cancel", "test-token-alpha", "", ""); w.Code != 202 {
		t.Fatal(w.Code)
	}
	if w = call("GET", "/api/quota", "test-token-alpha", "", ""); w.Code != 200 || !strings.Contains(w.Body.String(), `"active_tasks":1`) {
		t.Fatal(w.Code, w.Body)
	}
	if w = call("POST", "/api/tasks", "test-token-alpha", strings.TrimSuffix(body, "}")+`,"team_id":"beta"}`, "bad"); w.Code != 400 {
		t.Fatal(w.Code)
	}
	if w = call("GET", "/api/info", "test-token-alpha", "", ""); !strings.Contains(w.Body.String(), "local-demo") {
		t.Fatal(w.Body)
	}
	if w = call("GET", "/readyz", "", "", ""); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w = call("GET", "/metrics", "", "", ""); w.Code != 401 {
		t.Fatal("metrics need authentication", w.Code)
	}
	if w = call("GET", "/metrics", "test-token-alpha", "", ""); w.Code != 200 || !strings.Contains(w.Body.String(), "workbench_reserved_memory_mib") {
		t.Fatal(w.Code, w.Body)
	}
}

var _ = http.MethodGet
