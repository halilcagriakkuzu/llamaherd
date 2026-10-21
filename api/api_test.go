package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"llamaherd/schema"
	"llamaherd/settings"
)

// fake implements Control and records calls.
type fake struct {
	calls   []string
	openErr error
	added   schema.Config
}

func (f *fake) rec(s string) { f.calls = append(f.calls, s) }

func (f *fake) SnapshotJSON() ([]byte, error) { return []byte(`{"status":"stopped"}`), nil }
func (f *fake) DoctorJSON() ([]byte, error)   { return []byte(`{"ok":true}`), nil }
func (f *fake) DownloadJSON() ([]byte, error) { return []byte(`{"active":false}`), nil }
func (f *fake) LogsJSON(tail int, after int64) ([]byte, error) {
	f.rec("logs:" + strconv.Itoa(tail) + ":" + strconv.FormatInt(after, 10))
	return []byte(`{"lines":["hello"],"next":7,"enabled":true}`), nil
}
func (f *fake) CatalogJSON(all bool) ([]byte, error) { return []byte(`{"entries":[]}`), nil }
func (f *fake) StartDownload(ref, file, vision, mtp string) error {
	f.rec("download:" + ref + "|" + file + "|" + vision + "|" + mtp)
	return nil
}
func (f *fake) RepoFilesJSON(ref string) ([]byte, error) {
	return []byte(`{"repo":"` + ref + `","models":[]}`), nil
}
func (f *fake) CancelDownload() error { f.rec("cancel"); return nil }
func (f *fake) PauseDownload() error  { f.rec("pause"); return nil }
func (f *fake) ResumeDownload() error { f.rec("resume"); return nil }
func (f *fake) RetryDownload() error  { f.rec("retry"); return nil }
func (f *fake) RemoveQueuedDownload(id string) error {
	f.rec("queue-remove:" + id)
	return nil
}
func (f *fake) SetHFToken(t string) error { f.rec("token"); return nil }
func (f *fake) NotificationSettings() settings.NotificationSettings {
	s := settings.NotifySettings()
	s.ServerReady, s.Download = true, true
	return s
}
func (f *fake) SetNotificationSettings(s settings.NotificationSettings) error {
	f.rec(fmt.Sprintf("notifications:%v:%v", s.Enabled, s.Download))
	return nil
}
func (f *fake) Models() []schema.Model           { return []schema.Model{{Display: "m", Alias: "a"}} }
func (f *fake) Presets(a string) []schema.Preset { return []schema.Preset{{Name: "default"}} }
func (f *fake) SelectModelByAlias(string) error  { return nil }
func (f *fake) SelectPreset(string)              {}
func (f *fake) Open(alias, preset string) error {
	f.rec("open:" + alias + "/" + preset)
	return f.openErr
}
func (f *fake) Stop() error          { f.rec("stop"); return nil }
func (f *fake) RestartServer() error { f.rec("restart"); return nil }
func (f *fake) RescanModels() error  { f.rec("rescan"); return nil }
func (f *fake) ResetPreset(name string) error {
	f.rec("reset:" + name)
	return nil
}
func (f *fake) ExportPresets() (string, error) { return `{"alias":"a","presets":[]}`, nil }
func (f *fake) AutoFitConfig() (string, error) { f.rec("autofit"); return "fitted", nil }
func (f *fake) DeleteModel(ref string) error   { f.rec("delete:" + ref); return nil }
func (f *fake) ImportModels(paths []string, mmproj string, move, allowDuplicate bool) (schema.ImportReport, error) {
	f.rec(fmt.Sprintf("import:%s|%s|%v|%v", strings.Join(paths, ","), mmproj, move, allowDuplicate))
	return schema.ImportReport{Items: []schema.ImportItem{{Source: paths[0], Status: "imported"}}}, nil
}
func (f *fake) CheckImport(paths []string) []schema.ImportCheck {
	f.rec("check:" + strings.Join(paths, ","))
	return []schema.ImportCheck{{Path: paths[0], Duplicate: true}}
}
func (f *fake) RunBenchmark() (schema.Benchmark, error) {
	f.rec("bench")
	return schema.Benchmark{Preset: "default", GenTokS: 42.5}, nil
}
func (f *fake) Benchmarks() []schema.Benchmark { return []schema.Benchmark{{Preset: "default"}} }
func (f *fake) ExportBenchmarks(format string) ([]byte, error) {
	f.rec("export:" + format)
	return []byte("model,preset\r\nm,default\r\n"), nil
}
func (f *fake) ClearBenchmarks() error { f.rec("benchclear"); return nil }
func (f *fake) SetModelFavorite(ref string, fav bool) error {
	f.rec("favorite:" + ref)
	return nil
}
func (f *fake) SaveConfig() { f.rec("save") }
func (f *fake) LoadConfig() { f.rec("load") }
func (f *fake) AddPreset(a, n string, c schema.Config) error {
	f.rec("add:" + a + "/" + n)
	f.added = c
	return nil
}
func (f *fake) UpdatePreset(a, n string, c schema.Config) error {
	f.rec("update:" + a + "/" + n)
	return nil
}
func (f *fake) RemovePreset(a, n string) error { f.rec("remove:" + a + "/" + n); return nil }
func (f *fake) RenamePreset(a, oldName, newName string) error {
	f.rec("rename:" + a + "/" + oldName + ">" + newName)
	return nil
}
func (f *fake) RenameModel(ref, name, alias string) error {
	f.rec("rename:" + ref + "|" + name + "|" + alias)
	return f.openErr
}
func (f *fake) SetModelCategory(ref string, c []string) error {
	f.rec("setcat:" + ref + "|" + strings.Join(c, ">"))
	return nil
}
func (f *fake) CreateCategory(p []string) error {
	f.rec("catcreate:" + strings.Join(p, ">"))
	return nil
}
func (f *fake) RenameCategory(p []string, n string) error {
	f.rec("catrename:" + strings.Join(p, ">") + "|" + n)
	return nil
}
func (f *fake) DeleteCategory(p []string) error {
	f.rec("catdelete:" + strings.Join(p, ">"))
	return nil
}

func do(t *testing.T, h http.Handler, method, url, body string, hdr map[string]string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(method, url, strings.NewReader(body))
	// The production handler only accepts loopback Host headers; httptest's
	// default "example.com" is deliberately rejected by the new guard.
	req.Host = "127.0.0.1:8517"
	if method != http.MethodGet && method != http.MethodHead {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	b, _ := io.ReadAll(rec.Result().Body)
	return rec.Code, strings.TrimSpace(string(b))
}

func isolate(t *testing.T) {
	t.Helper()
	t.Cleanup(settings.Reload)
	t.Setenv(settings.EnvConfig, t.TempDir()+"/absent.json")
	t.Setenv(settings.EnvAPIToken, "")
	settings.Reload()
}

func TestReadEndpoints(t *testing.T) {
	isolate(t)
	h := Handler(&fake{})
	for path, want := range map[string]string{
		"/status":          `"status":"stopped"`,
		"/doctor":          `"ok":true`,
		"/download":        `"active":false`,
		"/catalog":         `"entries"`,
		"/models":          `"alias":"a"`,
		"/presets?alias=a": `"name":"default"`,
		"/logs?tail=5":     `"lines":["hello"]`,
	} {
		code, body := do(t, h, "GET", path, "", nil)
		if code != 200 || !strings.Contains(body, want) {
			t.Errorf("GET %s = %d %s, want %q", path, code, body, want)
		}
	}
	if code, _ := do(t, h, "GET", "/presets", "", nil); code != http.StatusBadRequest {
		t.Errorf("/presets without alias = %d, want 400", code)
	}
	if code, _ := do(t, h, "GET", "/logs?tail=-1", "", nil); code != http.StatusBadRequest {
		t.Errorf("/logs with negative tail = %d, want 400", code)
	}
	if code, _ := do(t, h, "GET", "/logs?after=x", "", nil); code != http.StatusBadRequest {
		t.Errorf("/logs with bad after = %d, want 400", code)
	}
}

func TestOpenStopAndErrors(t *testing.T) {
	isolate(t)
	f := &fake{}
	h := Handler(f)
	code, body := do(t, h, "POST", "/open", `{"alias":"a","preset":"p"}`, nil)
	if code != 200 || body != `{"ok":true}` {
		t.Fatalf("open = %d %s", code, body)
	}
	f.openErr = errors.New("start blocked: too big")
	_, body = do(t, h, "POST", "/open", `{"alias":"a"}`, nil)
	var r struct {
		OK    bool
		Error string
	}
	json.Unmarshal([]byte(body), &r)
	if r.OK || r.Error != "start blocked: too big" {
		t.Errorf("open error not surfaced: %s", body)
	}
	if code, _ := do(t, h, "POST", "/open", `{bad json`, nil); code != http.StatusBadRequest {
		t.Errorf("malformed body = %d, want 400", code)
	}
	do(t, h, "POST", "/stop", "", nil)
	if got := strings.Join(f.calls, ","); got != "open:a/p,open:a/,stop" {
		t.Errorf("calls = %s", got)
	}
}

func TestPresetEndpoints(t *testing.T) {
	isolate(t)
	f := &fake{}
	h := Handler(f)
	body := `{"alias":"a","name":"n","config":{"basic":{"model":"m.gguf","port":9000},"advanced":{}}}`
	do(t, h, "POST", "/config/add", body, nil)
	do(t, h, "POST", "/config/update", body, nil)
	do(t, h, "POST", "/config/remove", `{"alias":"a","name":"n"}`, nil)
	do(t, h, "POST", "/config/rename", `{"alias":"a","name":"n","newName":"m"}`, nil)
	if got := strings.Join(f.calls, ","); got != "add:a/n,update:a/n,remove:a/n,rename:a/n>m" {
		t.Errorf("calls = %s", got)
	}
	if f.added.Basic.Model != "m.gguf" || f.added.Basic.Port != 9000 {
		t.Errorf("config not decoded: %+v", f.added)
	}
}

func TestDownloadEndpoints(t *testing.T) {
	isolate(t)
	f := &fake{}
	h := Handler(f)
	do(t, h, "POST", "/download", `{"repo":"o/r","file":"f.gguf","vision":"v.gguf","mtp":"none"}`, nil)
	if code, body := do(t, h, "GET", "/download/files?ref=o/r", "", nil); code != 200 || !strings.Contains(body, `"repo":"o/r"`) {
		t.Errorf("GET /download/files = %d %s", code, body)
	}
	if code, _ := do(t, h, "GET", "/download/files", "", nil); code != http.StatusBadRequest {
		t.Errorf("missing ref = %d, want 400", code)
	}
	do(t, h, "POST", "/download/cancel", "", nil)
	do(t, h, "POST", "/download/pause", "", nil)
	do(t, h, "POST", "/download/resume", "", nil)
	do(t, h, "POST", "/download/retry", "", nil)
	do(t, h, "POST", "/download/queue", `{"id":"dl-2"}`, nil)
	want := "download:o/r|f.gguf|v.gguf|none,cancel,pause,resume,retry,queue-remove:dl-2"
	if got := strings.Join(f.calls, ","); got != want {
		t.Errorf("calls = %s, want %s", got, want)
	}
}

func TestBearerTokenAuth(t *testing.T) {
	isolate(t)
	t.Setenv(settings.EnvAPIToken, "s3cret")
	h := Handler(&fake{})
	if code, _ := do(t, h, "GET", "/status", "", nil); code != http.StatusUnauthorized {
		t.Errorf("no token = %d, want 401", code)
	}
	if code, _ := do(t, h, "GET", "/status", "", map[string]string{"Authorization": "Bearer wrong"}); code != http.StatusUnauthorized {
		t.Errorf("wrong token = %d, want 401", code)
	}
	if code, _ := do(t, h, "GET", "/status", "", map[string]string{"Authorization": "Bearer s3cret"}); code != 200 {
		t.Errorf("header token = %d, want 200", code)
	}
	if code, _ := do(t, h, "GET", "/status?token=s3cret", "", nil); code != 200 {
		t.Errorf("query token = %d, want 200", code)
	}
}

func TestPingBeaconNeedsNoToken(t *testing.T) {
	isolate(t)
	t.Setenv(settings.EnvAPIToken, "s3cret")
	h := Handler(&fake{})
	code, body := do(t, h, "GET", "/ping", "", nil)
	if code != 200 || !strings.Contains(body, `"app":"llamaherd"`) {
		t.Fatalf("GET /ping = %d %s, want the identity beacon", code, body)
	}
	// The loopback Host guard still applies to the unauthenticated beacon.
	req := httptest.NewRequest("GET", "/ping", nil)
	req.Host = "evil.example"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("non-loopback host = %d, want 403", rec.Code)
	}
}

func TestLoopbackGuards(t *testing.T) {
	isolate(t)
	h := Handler(&fake{})

	// A DNS-rebinding style Host is rejected.
	req := httptest.NewRequest("GET", "/status", nil)
	req.Host = "evil.example"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("non-loopback host = %d, want 403", rec.Code)
	}

	// A cross-origin browser request is rejected even without a token.
	if code, _ := do(t, h, "GET", "/status", "", map[string]string{"Origin": "https://evil.example"}); code != http.StatusForbidden {
		t.Errorf("cross-origin = %d, want 403", code)
	}
	if code, _ := do(t, h, "GET", "/status", "", map[string]string{"Origin": "http://127.0.0.1:5173"}); code != 200 {
		t.Errorf("loopback origin = %d, want 200", code)
	}

	// text/plain POSTs are the classic CSRF shape; require JSON.
	req = httptest.NewRequest("POST", "/stop", strings.NewReader("{}"))
	req.Host = "127.0.0.1:8517"
	req.Header.Set("Content-Type", "text/plain")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Errorf("text/plain POST = %d, want 415", rec.Code)
	}
}

func TestUXEndpoints(t *testing.T) {
	isolate(t)
	f := &fake{}
	h := Handler(f)
	do(t, h, "POST", "/restart", "", nil)
	do(t, h, "POST", "/rescan", "", nil)
	do(t, h, "POST", "/model/favorite", `{"model":"a","favorite":true}`, nil)
	do(t, h, "POST", "/model/delete", `{"model":"a"}`, nil)
	do(t, h, "POST", "/config/reset", `{"name":"default"}`, nil)
	do(t, h, "POST", "/config/autofit", "", nil)
	if code, body := do(t, h, "GET", "/config/export", "", nil); code != 200 || !strings.Contains(body, `"alias":"a"`) {
		t.Errorf("GET /config/export = %d %s", code, body)
	}
	want := "restart,rescan,favorite:a,delete:a,reset:default,autofit"
	if got := strings.Join(f.calls, ","); got != want {
		t.Errorf("calls = %s\nwant    %s", got, want)
	}
}

func TestBenchmarkAndImportEndpoints(t *testing.T) {
	isolate(t)
	f := &fake{}
	h := Handler(f)
	do(t, h, "POST", "/model/import", `{"src":"/tmp/m.gguf","mmproj":"/tmp/mmproj.gguf","move":true}`, nil)
	if code, body := do(t, h, "POST", "/benchmark", "", nil); code != 200 || !strings.Contains(body, `"genTokPerSec":42.5`) {
		t.Errorf("POST /benchmark = %d %s", code, body)
	}
	if code, body := do(t, h, "GET", "/benchmarks", "", nil); code != 200 || !strings.Contains(body, `"preset":"default"`) {
		t.Errorf("GET /benchmarks = %d %s", code, body)
	}
	do(t, h, "POST", "/benchmarks/clear", "", nil)
	want := "import:/tmp/m.gguf|/tmp/mmproj.gguf|true|false,bench,benchclear"
	if got := strings.Join(f.calls, ","); got != want {
		t.Errorf("calls = %s\nwant    %s", got, want)
	}
}

func TestLibraryEndpoints(t *testing.T) {
	isolate(t)
	f := &fake{}
	h := Handler(f)
	do(t, h, "POST", "/model/rename", `{"model":"a","name":"Nice name","alias":"nice"}`, nil)
	do(t, h, "POST", "/model/category", `{"model":"a","category":["Qwen","3.8"]}`, nil)
	do(t, h, "POST", "/category/create", `{"path":["Qwen","new"]}`, nil)
	do(t, h, "POST", "/category/rename", `{"path":["Qwen","new"],"name":"old"}`, nil)
	do(t, h, "POST", "/category/delete", `{"path":["Qwen","old"]}`, nil)
	want := "rename:a|Nice name|nice,setcat:a|Qwen>3.8,catcreate:Qwen>new,catrename:Qwen>new|old,catdelete:Qwen>old"
	if got := strings.Join(f.calls, ","); got != want {
		t.Errorf("calls = %s\nwant    %s", got, want)
	}
	f.openErr = errors.New("alias taken")
	_, body := do(t, h, "POST", "/model/rename", `{"model":"a","alias":"x"}`, nil)
	if !strings.Contains(body, `"ok":false`) || !strings.Contains(body, "alias taken") {
		t.Errorf("error not surfaced: %s", body)
	}
	if code, _ := do(t, h, "POST", "/category/create", `{bad`, nil); code != http.StatusBadRequest {
		t.Errorf("malformed body = %d", code)
	}
}

func TestNotificationEndpoints(t *testing.T) {
	isolate(t)
	f := &fake{}
	h := Handler(f)
	if code, body := do(t, h, "GET", "/notifications", "", nil); code != 200 || !strings.Contains(body, `"enabled":true`) {
		t.Errorf("GET /notifications = %d %s", code, body)
	}
	if code, body := do(t, h, "POST", "/notifications", `{"enabled":true,"download":false}`, nil); code != 200 || !strings.Contains(body, `"ok":true`) {
		t.Errorf("POST /notifications = %d %s", code, body)
	}
	if got := strings.Join(f.calls, ","); !strings.Contains(got, "notifications:true:false") {
		t.Errorf("calls = %s", got)
	}
}

func TestBenchmarkExportEndpoint(t *testing.T) {
	isolate(t)
	f := &fake{}
	h := Handler(f)
	if code, body := do(t, h, "GET", "/benchmarks/export?format=csv", "", nil); code != 200 || !strings.Contains(body, "model,preset") {
		t.Errorf("CSV export = %d %s", code, body)
	}
	if code, _ := do(t, h, "GET", "/benchmarks/export?format=json", "", nil); code != 200 {
		t.Errorf("JSON export = %d", code)
	}
	if got := strings.Join(f.calls, ","); got != "export:csv,export:json" {
		t.Errorf("calls = %s", got)
	}
}
