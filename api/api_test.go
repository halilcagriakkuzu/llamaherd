package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
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
func (f *fake) CatalogJSON() ([]byte, error)  { return []byte(`{"entries":[]}`), nil }
func (f *fake) StartDownload(ref, file, vision string) error {
	f.rec("download:" + ref + "|" + file + "|" + vision)
	return nil
}
func (f *fake) CancelDownload() error            { f.rec("cancel"); return nil }
func (f *fake) SetHFToken(t string) error        { f.rec("token"); return nil }
func (f *fake) Models() []schema.Model           { return []schema.Model{{Display: "m", Alias: "a"}} }
func (f *fake) Presets(a string) []schema.Preset { return []schema.Preset{{Name: "default"}} }
func (f *fake) SelectModelByAlias(string) error  { return nil }
func (f *fake) SelectPreset(string)              {}
func (f *fake) Open(alias, preset string) error {
	f.rec("open:" + alias + "/" + preset)
	return f.openErr
}
func (f *fake) Stop() error { f.rec("stop"); return nil }
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
	} {
		code, body := do(t, h, "GET", path, "", nil)
		if code != 200 || !strings.Contains(body, want) {
			t.Errorf("GET %s = %d %s, want %q", path, code, body, want)
		}
	}
	if code, _ := do(t, h, "GET", "/presets", "", nil); code != http.StatusBadRequest {
		t.Errorf("/presets without alias = %d, want 400", code)
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
	if got := strings.Join(f.calls, ","); got != "add:a/n,update:a/n,remove:a/n" {
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
	do(t, h, "POST", "/download", `{"repo":"o/r","file":"f.gguf","vision":"v.gguf"}`, nil)
	do(t, h, "POST", "/download/cancel", "", nil)
	if got := strings.Join(f.calls, ","); got != "download:o/r|f.gguf|v.gguf,cancel" {
		t.Errorf("calls = %s", got)
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
