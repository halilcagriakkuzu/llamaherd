package control

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// VramGPU reports GPU memory. In WSL the per-process query answers
// "[N/A]" (a known WSL nvidia-smi limitation), so we fall back to the
// total GPU memory used — good enough on a single-user workstation.
func VramMB(pid int) int {
	if pid <= 0 {
		return 0
	}
	total := perPidVram(pid)
	if total > 0 {
		return total
	}
	return totalGpuVram()
}

// perPidVram sums the process's GPU memory across all visible GPUs.
func perPidVram(pid int) int {
	out, err := exec.Command("nvidia-smi",
		"--query-compute-apps=pid,used_gpu_memory",
		"--format=csv,noheader,nounits").Output()
	if err != nil {
		return 0
	}
	total := 0
	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			continue
		}
		p, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil || p != pid {
			continue
		}
		mem, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			continue
		}
		total += mem
	}
	return total
}

// totalGpuVram sums memory.used across all visible GPUs (WSL fallback: the
// per-process metric is not reported inside WSL).
func totalGpuVram() int {
	out, err := exec.Command("nvidia-smi",
		"--query-gpu=memory.used", "--format=csv,noheader,nounits").Output()
	if err != nil {
		return 0
	}
	used := 0
	for _, line := range strings.Split(string(out), "\n") {
		v, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil {
			continue
		}
		used += v
	}
	return used
}

// RamMB reports the resident set size of the given pid in MiB (0 when the
// platform cannot report it). Implementation is platform-specific
// (proc_linux.go / proc_darwin.go / proc_windows.go).
func RamMB(pid int) int {
	if pid <= 0 {
		return 0
	}
	return procRSS(pid)
}

// ServerProps is what a running llama-server reports about the model it has
// actually loaded (queried from /props). This is authoritative: it describes
// the running process, not whatever config the UI currently has selected.
type ServerProps struct {
	ModelPath  string
	ModelAlias string
	Ftype      string
	NCtx       int
	Slots      int
}

// FetchServerProps reads /props and returns the loaded model's identity and
// runtime parameters. Zero values mean "unknown" (server not up / old build).
func FetchServerProps(addr string) ServerProps {
	cl := &http.Client{Timeout: 2 * time.Second}
	resp, err := cl.Get("http://" + addr + "/props")
	if err != nil {
		return ServerProps{}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ServerProps{}
	}
	var props struct {
		DefaultGenerationSettings struct {
			NCtx int `json:"n_ctx"`
		} `json:"default_generation_settings"`
		NCtx       int    `json:"n_ctx"`
		ModelPath  string `json:"model_path"`
		ModelAlias string `json:"model_alias"`
		ModelFtype string `json:"model_ftype"`
		TotalSlots int    `json:"total_slots"`
	}
	if json.NewDecoder(resp.Body).Decode(&props) != nil {
		return ServerProps{}
	}
	p := ServerProps{
		ModelPath:  props.ModelPath,
		ModelAlias: props.ModelAlias,
		Ftype:      props.ModelFtype,
		Slots:      props.TotalSlots,
	}
	if props.DefaultGenerationSettings.NCtx > 0 {
		p.NCtx = props.DefaultGenerationSettings.NCtx
	} else {
		p.NCtx = props.NCtx
	}
	return p
}

// NCtx fetches the server's actual context size from /props, or 0 on any
// failure (the caller falls back to the configured value).
func NCtx(addr string) int {
	return FetchServerProps(addr).NCtx
}

// HumanMB formats a memory amount the way the status panel shows it
// ("16.8 GB"); <=0 returns an em-dash.
func HumanMB(mb int) string {
	if mb <= 0 {
		return "—"
	}
	if mb < 1024 {
		return fmt.Sprintf("%d MB", mb)
	}
	return fmt.Sprintf("%.1f GB", float64(mb)/1024)
}
