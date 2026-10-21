package settings

import (
	"os"
	"testing"
)

func TestLastRunSaveLoad(t *testing.T) {
	t.Setenv(EnvDataDir, t.TempDir())
	if _, ok := LoadLastRun(); ok {
		t.Fatal("a fresh data dir must not have a remembered selection")
	}
	if err := SaveLastRun(LastRun{Alias: "qwen", Preset: "fast", At: 123}); err != nil {
		t.Fatal(err)
	}
	got, ok := LoadLastRun()
	if !ok || got.Alias != "qwen" || got.Preset != "fast" || got.At != 123 {
		t.Fatalf("last run = %+v ok=%v", got, ok)
	}
	if st, err := os.Stat(LastRunFile()); err != nil || st.Mode().Perm() != 0o600 {
		t.Fatalf("last-run file mode = %v err=%v", st, err)
	}
	// A corrupt file is ignored rather than crashing startup.
	if err := os.WriteFile(LastRunFile(), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok := LoadLastRun(); ok {
		t.Fatal("corrupt last-run.json must be ignored")
	}
}
