// Package settings resolves machine-specific configuration without hard-coding
// it. Values come from (in order of precedence):
//
//  1. an explicit environment variable (user intent always wins)
//  2. the user config file (JSON, standard per-OS location)
//  3. auto-discovery on this machine (PATH, common checkout locations)
//  4. a built-in default
//
// Environment variables:
//
//	LLAMAHERD_SERVER_BIN        full path to the llama-server binary
//	LLAMAHERD_LLAMA_DIR         the llama.cpp checkout directory
//	LLAMAHERD_LLAMA_BUILD_DIR   the build subdirectory (e.g. "build-latest")
//	LLAMAHERD_MODELS_DIR        the directory scanned for *.gguf models
//	LLAMAHERD_API_PORT          the loopback control-API port (default 8517)
//	LLAMAHERD_CONFIG            override the config-file path (mostly for tests)
//
// The user config file lives at (first match):
//
//	Linux/BSD:  $XDG_CONFIG_HOME/llamaherd/config.json
//	            else ~/.config/llamaherd/config.json
//	Windows:    %APPDATA%\llamaherd\config.json
//	macOS:      ~/Library/Application Support/llamaherd/config.json
package settings

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

const (
	// EnvServerBin overrides the llama-server binary path.
	EnvServerBin = "LLAMAHERD_SERVER_BIN"
	// EnvLlamaDir overrides the llama.cpp checkout directory.
	EnvLlamaDir = "LLAMAHERD_LLAMA_DIR"
	// EnvBuildDir overrides the build subdirectory name within the checkout.
	EnvBuildDir = "LLAMAHERD_LLAMA_BUILD_DIR"
	// EnvModelsDir overrides the directory scanned for *.gguf models.
	EnvModelsDir = "LLAMAHERD_MODELS_DIR"
	// EnvAPIPort overrides the loopback control-API port.
	EnvAPIPort = "LLAMAHERD_API_PORT"
	// EnvAPIToken sets a bearer token required by the loopback control API.
	EnvAPIToken = "LLAMAHERD_API_TOKEN"
	// EnvConfig overrides the user config-file path.
	EnvConfig = "LLAMAHERD_CONFIG"
	// EnvDataDir overrides the per-user data directory (configs/, library/).
	EnvDataDir = "LLAMAHERD_DATA_DIR"
	// EnvConfigsDir overrides the per-alias preset directory.
	EnvConfigsDir = "LLAMAHERD_CONFIGS_DIR"
	// EnvLibraryDir overrides the per-model metadata directory.
	EnvLibraryDir = "LLAMAHERD_LIBRARY_DIR"

	// DefaultBuildDir is used when the build directory cannot be derived.
	DefaultBuildDir = "build-latest"
	// DefaultAPIPort is the control-API port when none is configured.
	DefaultAPIPort = "8517"

	// defaultBinName is the bare executable name searched on PATH.
	defaultBinName = "llama-server"
	// appDirName is the per-app subdirectory for the config file.
	appDirName = "llamaherd"
)

// File is the on-disk user configuration (JSON). Empty fields fall through to
// discovery/defaults, so a partial file is fine.
type File struct {
	ServerBin     string `json:"serverBin,omitempty"`
	LlamaDir      string `json:"llamaDir,omitempty"`
	LlamaBuildDir string `json:"llamaBuildDir,omitempty"`
	ModelsDir     string `json:"modelsDir,omitempty"`
	APIPort       string `json:"apiPort,omitempty"`
	APIToken      string `json:"apiToken,omitempty"`
}

// ConfigFile returns the path of the user config file.
func ConfigFile() string {
	if v := strings.TrimSpace(os.Getenv(EnvConfig)); v != "" {
		return expandHome(v)
	}
	switch runtime.GOOS {
	case "windows":
		if d := os.Getenv("APPDATA"); d != "" {
			return filepath.Join(d, appDirName, "config.json")
		}
	case "darwin":
		if h := homeDir(); h != "" {
			return filepath.Join(h, "Library", "Application Support", appDirName, "config.json")
		}
	}
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, appDirName, "config.json")
	}
	if h := homeDir(); h != "" {
		return filepath.Join(h, ".config", appDirName, "config.json")
	}
	return filepath.Join(appDirName, "config.json")
}

var (
	cfgMu    sync.Mutex
	cfgPath  string
	cfgCache File
	cfgValid bool
)

// Config returns the parsed user config file (zero File when absent). The
// result is cached and re-read only when the config path changes.
func Config() File {
	cfgMu.Lock()
	defer cfgMu.Unlock()
	p := ConfigFile()
	if cfgValid && cfgPath == p {
		return cfgCache
	}
	cfgPath, cfgValid = p, true
	cfgCache = File{}
	if data, err := os.ReadFile(p); err == nil {
		_ = json.Unmarshal(data, &cfgCache)
	}
	return cfgCache
}

// Reload drops the cached config so the next read re-reads the file.
func Reload() {
	cfgMu.Lock()
	cfgValid = false
	cfgMu.Unlock()
}

// Save writes f to ConfigFile (creating parent directories) and refreshes the
// cache.
func Save(f File) error {
	cfgMu.Lock()
	defer cfgMu.Unlock()
	p := ConfigFile()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(p, append(data, '\n'), 0o600); err != nil {
		return err
	}
	cfgPath, cfgCache, cfgValid = p, f, true
	return nil
}

// ConfigExists reports whether the user config file is present.
func ConfigExists() bool {
	st, err := os.Stat(ConfigFile())
	return err == nil && !st.IsDir()
}

// ServerBin returns the llama-server binary path. An explicit
// LLAMAHERD_SERVER_BIN always wins (even if the file is missing, so the user
// sees their own value in errors); then the config file; then a PATH lookup
// and the usual local build locations; the last resort is the expected default
// path so the "binary missing" warning is informative.
func ServerBin() string {
	if v := strings.TrimSpace(os.Getenv(EnvServerBin)); v != "" {
		return expandHome(v)
	}
	if v := strings.TrimSpace(Config().ServerBin); v != "" {
		return expandHome(v)
	}
	if p, err := exec.LookPath(binFileName()); err == nil {
		return p
	}
	cands := serverBinCandidates()
	for _, p := range cands {
		if fileExists(p) {
			return p
		}
	}
	if len(cands) > 0 {
		return cands[0]
	}
	return binFileName()
}

// binFileName is the platform's llama-server executable name (".exe" on
// Windows).
func binFileName() string {
	if runtime.GOOS == "windows" {
		return defaultBinName + ".exe"
	}
	return defaultBinName
}

// LlamaDir returns the llama.cpp checkout directory. An explicit
// LLAMAHERD_LLAMA_DIR or config-file value wins; otherwise it is derived from
// the resolved binary path when that looks like a build-tree layout; otherwise
// the usual ~/llama.cpp location is used.
func LlamaDir() string {
	if v := configuredLlamaDir(); v != "" {
		return v
	}
	if d, _ := deriveFromBinary(ServerBin()); d != "" {
		return d
	}
	var cands []string
	if h := homeDir(); h != "" {
		cands = append(cands, filepath.Join(h, "llama.cpp"))
	}
	cands = append(cands, "llama.cpp")
	for _, p := range cands {
		if isDir(p) {
			return p
		}
	}
	if len(cands) > 0 {
		return cands[0]
	}
	return "llama.cpp"
}

// LlamaBuildDir returns the build subdirectory name inside the checkout. An
// explicit env/config value wins; otherwise it is derived from the binary path
// (e.g. .../build-latest/bin/llama-server -> "build-latest"); otherwise
// DefaultBuildDir.
func LlamaBuildDir() string {
	if v := strings.TrimSpace(os.Getenv(EnvBuildDir)); v != "" {
		return v
	}
	if v := strings.TrimSpace(Config().LlamaBuildDir); v != "" {
		return v
	}
	if _, b := deriveFromBinary(ServerBin()); b != "" {
		return b
	}
	return DefaultBuildDir
}

// ModelsDir returns the directory scanned (recursively) for *.gguf models. An
// explicit LLAMAHERD_MODELS_DIR or config-file value wins; otherwise
// DefaultModelsDir.
func ModelsDir() string {
	if v := strings.TrimSpace(os.Getenv(EnvModelsDir)); v != "" {
		return expandHome(v)
	}
	if v := strings.TrimSpace(Config().ModelsDir); v != "" {
		return expandHome(v)
	}
	return DefaultModelsDir()
}

// DefaultModelsDir is <home>/LlamaHerd/models (e.g. C:\Users\you\LlamaHerd\models
// on Windows, ~/LlamaHerd/models elsewhere); it falls back to the project root
// only when no home directory can be determined.
func DefaultModelsDir() string {
	if h := homeDir(); h != "" {
		return filepath.Join(h, "LlamaHerd", "models")
	}
	return filepath.Join(ProjectRoot(), "models")
}

// IsDefaultModelsDir reports whether the models directory is the built-in
// default (not set via env or config), i.e. safe for the app to create.
func IsDefaultModelsDir() bool {
	return strings.TrimSpace(os.Getenv(EnvModelsDir)) == "" && strings.TrimSpace(Config().ModelsDir) == ""
}

// APIPort returns the loopback control-API port as a string. An explicit
// LLAMAHERD_API_PORT or config-file value wins; otherwise DefaultAPIPort.
func APIPort() string {
	if v := strings.TrimSpace(os.Getenv(EnvAPIPort)); v != "" {
		return v
	}
	if v := strings.TrimSpace(Config().APIPort); v != "" {
		return v
	}
	return DefaultAPIPort
}

// APIAddr returns the full loopback control-API address (host fixed to
// 127.0.0.1, port from APIPort).
func APIAddr() string {
	return "127.0.0.1:" + APIPort()
}

// APIToken returns the bearer token the loopback control API requires, or ""
// for no authentication. An explicit LLAMAHERD_API_TOKEN or config-file value
// wins.
func APIToken() string {
	if v := strings.TrimSpace(os.Getenv(EnvAPIToken)); v != "" {
		return v
	}
	return strings.TrimSpace(Config().APIToken)
}

// DataDir returns the per-user writable directory for generated data
// (configs/, library/). Defaults:
//
//	Linux/BSD:  $XDG_DATA_HOME/llamaherd, else ~/.local/share/llamaherd
//	Windows:    %LOCALAPPDATA%\llamaherd
//	macOS:      ~/Library/Application Support/llamaherd
func DataDir() string {
	if v := strings.TrimSpace(os.Getenv(EnvDataDir)); v != "" {
		return expandHome(v)
	}
	switch runtime.GOOS {
	case "windows":
		if d := os.Getenv("LOCALAPPDATA"); d != "" {
			return filepath.Join(d, appDirName)
		}
	case "darwin":
		if h := homeDir(); h != "" {
			return filepath.Join(h, "Library", "Application Support", appDirName)
		}
	}
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return filepath.Join(d, appDirName)
	}
	if h := homeDir(); h != "" {
		return filepath.Join(h, ".local", "share", appDirName)
	}
	return appDirName
}

// ConfigsDir returns the directory holding per-alias preset files
// (configs/<alias>.json). A checkout's ./configs (developer layout) wins when
// present so dev and `go test` keep working; otherwise the per-user data
// directory is used, so an installed binary never writes into the current
// working directory.
func ConfigsDir() string {
	if v := strings.TrimSpace(os.Getenv(EnvConfigsDir)); v != "" {
		return expandHome(v)
	}
	if isDir("configs") {
		return "configs"
	}
	return filepath.Join(DataDir(), "configs")
}

// LibraryDir returns the directory holding per-model metadata files
// (library/<id>.json). Same resolution rule as ConfigsDir.
func LibraryDir() string {
	if v := strings.TrimSpace(os.Getenv(EnvLibraryDir)); v != "" {
		return expandHome(v)
	}
	if isDir("library") {
		return "library"
	}
	return filepath.Join(DataDir(), "library")
}

// ResolveModelPath turns a configured model / mmproj path into an absolute
// path. Shipped configs store paths relative to the models directory; absolute
// paths (e.g. ones a user saved) pass through unchanged. A leading ~ and the
// tokens ${MODELS_DIR} / ${LLAMAHERD_MODELS_DIR} (and their $VAR forms) are
// expanded.
func ResolveModelPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	p = expandHome(p)
	md := ModelsDir()
	for _, tok := range []string{
		"${MODELS_DIR}", "$MODELS_DIR",
		"${LLAMAHERD_MODELS_DIR}", "$LLAMAHERD_MODELS_DIR",
	} {
		p = strings.ReplaceAll(p, tok, md)
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Clean(filepath.Join(md, p))
}

// RelativizeModelPath is the inverse of ResolveModelPath: when p lies under
// the models directory it returns a slash-separated relative path (portable
// across machines), otherwise it returns p unchanged. Used when saving configs
// so they never capture a machine-specific absolute path.
func RelativizeModelPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	md := ModelsDir()
	if filepath.IsAbs(p) {
		if rel, err := filepath.Rel(md, p); err == nil &&
			rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return filepath.ToSlash(rel)
		}
	}
	return p
}

// configuredLlamaDir returns the env- or config-file-supplied checkout dir
// (no discovery), used to avoid a ServerBin/LlamaDir recursion.
func configuredLlamaDir() string {
	if v := strings.TrimSpace(os.Getenv(EnvLlamaDir)); v != "" {
		return expandHome(v)
	}
	if v := strings.TrimSpace(Config().LlamaDir); v != "" {
		return expandHome(v)
	}
	return ""
}

// serverBinCandidates lists discovery locations in priority order.
func serverBinCandidates() []string {
	var c []string
	if d := configuredLlamaDir(); d != "" {
		c = append(c, filepath.Join(d, DefaultBuildDir, "bin", binFileName()))
		c = append(c, filepath.Join(d, "build", "bin", binFileName()))
	}
	if h := homeDir(); h != "" {
		c = append(c, filepath.Join(h, "llama.cpp", DefaultBuildDir, "bin", binFileName()))
		c = append(c, filepath.Join(h, "llama.cpp", "build", "bin", binFileName()))
	}
	c = append(c, filepath.Join("llama.cpp", DefaultBuildDir, "bin", binFileName()))
	c = append(c, filepath.Join("llama.cpp", "build", "bin", binFileName()))
	return c
}

// ProjectRoot returns the LlamaHerd project root: the nearest ancestor of the
// working directory that holds a go.mod, so the same default works whether the
// app runs from the repo root, from build/bin/, or under `go test`. If no
// module root is found it falls back to the executable's directory, then the
// working directory.
func ProjectRoot() string {
	if wd, err := os.Getwd(); err == nil {
		if r := findRoot(wd); r != "" {
			return r
		}
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		if r := findRoot(dir); r != "" {
			return r
		}
		return dir
	}
	return "."
}

// findRoot walks up from start looking for a directory that contains go.mod.
func findRoot(start string) string {
	dir := filepath.Clean(start)
	for {
		if fileExists(filepath.Join(dir, "go.mod")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// deriveFromBinary splits a build-tree binary path like
// <repo>/<buildDir>/bin/llama-server into (repo, buildDir). It returns empty
// strings when the path does not look like a llama.cpp build tree (e.g. a
// plain PATH install such as /usr/bin/llama-server), where the build
// directory is meaningless.
func deriveFromBinary(bin string) (dir, buildDir string) {
	if bin == "" {
		return "", ""
	}
	clean := filepath.Clean(bin)
	base := filepath.Base(clean)
	if base != defaultBinName && base != defaultBinName+".exe" {
		return "", ""
	}
	binDir := filepath.Dir(clean)
	if filepath.Base(binDir) != "bin" {
		return "", ""
	}
	buildPath := filepath.Dir(binDir)
	buildDir = filepath.Base(buildPath)
	if !strings.HasPrefix(buildDir, "build") {
		return "", ""
	}
	return filepath.Dir(buildPath), buildDir
}

func homeDir() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return h
}

// expandHome expands a leading ~ in a user-supplied path.
func expandHome(p string) string {
	if p == "~" {
		return homeDir()
	}
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
		if h := homeDir(); h != "" {
			return filepath.Join(h, p[2:])
		}
	}
	return p
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}
