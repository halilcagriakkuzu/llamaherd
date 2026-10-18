package settings

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// isolate points LLAMAHERD_CONFIG at a non-existent temp file so a real user
// config on the developer's machine cannot leak into these tests.
func isolate(t *testing.T) {
	t.Helper()
	t.Setenv(EnvConfig, filepath.Join(t.TempDir(), "absent-config.json"))
	Reload()
}

func TestServerBinEnvOverrideWins(t *testing.T) {
	isolate(t)
	want := filepath.Join(t.TempDir(), "somewhere", "llama-server")
	t.Setenv(EnvServerBin, want)
	if got := ServerBin(); got != want {
		t.Fatalf("ServerBin() = %q, want %q", got, want)
	}
}

func TestDeriveFromBinary(t *testing.T) {
	j := filepath.Join
	tests := []struct {
		name      string
		bin       string
		wantDir   string
		wantBuild string
	}{
		{"build-latest", j("home", "u", "llama.cpp", "build-latest", "bin", "llama-server"), j("home", "u", "llama.cpp"), "build-latest"},
		{"plain build", j("opt", "llama.cpp", "build", "bin", "llama-server"), j("opt", "llama.cpp"), "build"},
		{"path install", j("usr", "bin", "llama-server"), "", ""},
		{"wrong name", j("home", "u", "llama.cpp", "build", "bin", "other"), "", ""},
		{"empty", "", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir, build := deriveFromBinary(tc.bin)
			if dir != tc.wantDir || build != tc.wantBuild {
				t.Fatalf("deriveFromBinary(%q) = (%q,%q), want (%q,%q)", tc.bin, dir, build, tc.wantDir, tc.wantBuild)
			}
		})
	}
}

func TestDeriveLlamaDirAndBuildDirFromBinary(t *testing.T) {
	isolate(t)
	repo := filepath.Join(t.TempDir(), "repo")
	binDir := filepath.Join(repo, "build-latest", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(binDir, "llama-server")
	if err := os.WriteFile(bin, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvServerBin, bin)
	t.Setenv(EnvLlamaDir, "")
	t.Setenv(EnvBuildDir, "")
	if got := LlamaDir(); got != repo {
		t.Fatalf("LlamaDir() = %q, want %q", got, repo)
	}
	if got := LlamaBuildDir(); got != "build-latest" {
		t.Fatalf("LlamaBuildDir() = %q, want %q", got, "build-latest")
	}
}

func TestEnvBuildDirWins(t *testing.T) {
	isolate(t)
	t.Setenv(EnvBuildDir, "build-custom")
	if got := LlamaBuildDir(); got != "build-custom" {
		t.Fatalf("LlamaBuildDir() = %q, want %q", got, "build-custom")
	}
}

func TestModelsDirEnvOverride(t *testing.T) {
	isolate(t)
	want := filepath.Join(t.TempDir(), "my-models")
	t.Setenv(EnvModelsDir, want)
	if got := ModelsDir(); got != want {
		t.Fatalf("ModelsDir() = %q, want %q", got, want)
	}
}

func TestModelsDirDefaultIsHomeLlamaHerdModels(t *testing.T) {
	isolate(t)
	t.Setenv(EnvModelsDir, "")
	h := homeDir()
	if h == "" {
		t.Skip("no home directory")
	}
	if got, want := ModelsDir(), filepath.Join(h, "LlamaHerd", "models"); got != want {
		t.Fatalf("ModelsDir() = %q, want %q", got, want)
	}
}

func TestAPIPort(t *testing.T) {
	isolate(t)
	t.Setenv(EnvAPIPort, "9001")
	if got := APIPort(); got != "9001" {
		t.Fatalf("APIPort() = %q, want 9001", got)
	}
	if got := APIAddr(); got != "127.0.0.1:9001" {
		t.Fatalf("APIAddr() = %q", got)
	}
}

func TestAPIPortDefault(t *testing.T) {
	isolate(t)
	t.Setenv(EnvAPIPort, "")
	if got := APIPort(); got != DefaultAPIPort {
		t.Fatalf("APIPort() = %q, want %q", got, DefaultAPIPort)
	}
}

func TestConfigFileLoadsValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv(EnvConfig, path)
	t.Setenv(EnvServerBin, "")
	t.Setenv(EnvModelsDir, "")
	t.Setenv(EnvAPIPort, "")
	t.Setenv(EnvLlamaDir, "")
	want := File{
		ServerBin: "/opt/llama.cpp/build/bin/llama-server",
		LlamaDir:  "/opt/llama.cpp",
		ModelsDir: "/data/models",
		APIPort:   "9000",
	}
	if err := Save(want); err != nil {
		t.Fatal(err)
	}
	Reload()
	if got := ServerBin(); got != want.ServerBin {
		t.Fatalf("ServerBin() = %q, want %q", got, want.ServerBin)
	}
	if got := LlamaDir(); got != want.LlamaDir {
		t.Fatalf("LlamaDir() = %q, want %q", got, want.LlamaDir)
	}
	if got := ModelsDir(); got != want.ModelsDir {
		t.Fatalf("ModelsDir() = %q, want %q", got, want.ModelsDir)
	}
	if got := APIPort(); got != "9000" {
		t.Fatalf("APIPort() = %q, want 9000", got)
	}
}

func TestEnvOverridesConfigFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv(EnvConfig, path)
	if err := Save(File{ModelsDir: "/from-file"}); err != nil {
		t.Fatal(err)
	}
	Reload()
	t.Setenv(EnvModelsDir, "/from-env")
	if got := ModelsDir(); got != "/from-env" {
		t.Fatalf("ModelsDir() = %q, want /from-env", got)
	}
}

func TestDataDirEnvOverride(t *testing.T) {
	isolate(t)
	want := filepath.Join(t.TempDir(), "data")
	t.Setenv(EnvDataDir, want)
	if got := DataDir(); got != want {
		t.Fatalf("DataDir() = %q, want %q", got, want)
	}
}

func TestConfigsDirAndLibraryDir(t *testing.T) {
	isolate(t)
	data := filepath.Join(t.TempDir(), "data")
	t.Setenv(EnvDataDir, data)
	t.Setenv(EnvConfigsDir, "")
	t.Setenv(EnvLibraryDir, "")
	// The settings package directory has no ./configs or ./library, so both
	// fall through to the per-user data dir.
	if got, want := ConfigsDir(), filepath.Join(data, "configs"); got != want {
		t.Fatalf("ConfigsDir() = %q, want %q", got, want)
	}
	if got, want := LibraryDir(), filepath.Join(data, "library"); got != want {
		t.Fatalf("LibraryDir() = %q, want %q", got, want)
	}
	// Explicit overrides win over the cwd check.
	cdir := filepath.Join(t.TempDir(), "myconfigs")
	t.Setenv(EnvConfigsDir, cdir)
	if got := ConfigsDir(); got != cdir {
		t.Fatalf("ConfigsDir() = %q, want %q", got, cdir)
	}
}

func TestResolveModelPath(t *testing.T) {
	isolate(t)
	md := t.TempDir()
	t.Setenv(EnvModelsDir, md)
	abs := "/abs/a.gguf"
	if runtime.GOOS == "windows" {
		abs = `C:\abs\a.gguf`
	}
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"sub/a.gguf", filepath.Join(md, "sub", "a.gguf")},
		{abs, abs},
		{"${MODELS_DIR}/x.gguf", filepath.Join(md, "x.gguf")},
		{"$LLAMAHERD_MODELS_DIR/y.gguf", filepath.Join(md, "y.gguf")},
	}
	for _, tc := range cases {
		if got := ResolveModelPath(tc.in); got != tc.want {
			t.Fatalf("ResolveModelPath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestExpandHome(t *testing.T) {
	home := homeDir()
	if home == "" {
		t.Skip("no home dir")
	}
	if got := expandHome("~/x"); got != filepath.Join(home, "x") {
		t.Fatalf("expandHome(~/x) = %q", got)
	}
	if got := expandHome("relative/x"); got != "relative/x" {
		t.Fatalf("expandHome(relative/x) = %q", got)
	}
}
