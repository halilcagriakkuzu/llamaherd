package control

// gpu.go — a small best-effort GPU provider abstraction. NVIDIA is the only
// path exercised on the developer machine; AMD (rocm-smi), Intel (xpu-smi) and
// Apple Silicon (unified memory) are implemented from their documented output
// and unit-tested with injected command output. Unknown values stay zero, which
// the UI renders as "—" and the VRAM guard treats as "no estimate".

import (
	"encoding/json"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// GPUProvider exposes the numbers the app needs from a vendor tool.
type GPUProvider interface {
	ID() string     // nvidia | amd | intel | apple | none
	Vendor() string // human vendor name
	Name() string   // device name, "" when unknown
	TotalVRAMMB() int
	UsedVRAMMB() int
	ProcessVRAMMB(pid int) int
	UtilTemp() (util, temp int)
}

// GPUInfo is the provider identity used by the doctor report.
type GPUInfo struct {
	ID          string
	Vendor      string
	Name        string
	TotalVRAMMB int
}

// gpuCommand is injectable so every parser can be tested without hardware.
type gpuCommand func(name string, args ...string) ([]byte, error)

func defaultGPUCommand(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

var (
	gpuCommandRun gpuCommand = defaultGPUCommand
	gpuProbe      func() GPUProvider
	gpuMu         sync.Mutex
	gpuCached     GPUProvider
	gpuAt         time.Time
)

// ActiveGPU returns the detected provider (cached for 10 s, like the old
// nvidia-smi probe). A provider that cannot report memory is not selected.
func ActiveGPU() GPUProvider {
	gpuMu.Lock()
	defer gpuMu.Unlock()
	if gpuProbe != nil {
		return gpuProbe()
	}
	if gpuCached != nil && time.Since(gpuAt) < 10*time.Second {
		return gpuCached
	}
	gpuCached = detectGPU()
	gpuAt = time.Now()
	return gpuCached
}

// GPUInfoNow is the provider identity for the doctor/UI (no device query beyond
// what ActiveGPU already does).
func GPUInfoNow() GPUInfo {
	p := ActiveGPU()
	return GPUInfo{ID: p.ID(), Vendor: p.Vendor(), Name: p.Name(), TotalVRAMMB: p.TotalVRAMMB()}
}

// SetGPUProviderForTesting pins the active provider until the returned restore
// function is called.
func SetGPUProviderForTesting(p GPUProvider) (restore func()) {
	prev := gpuProbe
	gpuProbe = func() GPUProvider { return p }
	return func() { gpuProbe = prev }
}

func detectGPU() GPUProvider {
	if runtime.GOOS == "darwin" {
		p := appleGPU{run: gpuCommandRun}
		if p.TotalVRAMMB() > 0 {
			return p
		}
	}
	for _, p := range []GPUProvider{
		nvidiaGPU{run: gpuCommandRun},
		amdGPU{run: gpuCommandRun},
		intelGPU{run: gpuCommandRun},
	} {
		if p.TotalVRAMMB() > 0 {
			return p
		}
	}
	return noGPU{}
}

// noGPU is the explicit "no provider" result.
type noGPU struct{}

func (noGPU) ID() string     { return "none" }
func (noGPU) Vendor() string { return "" }
func (noGPU) Name() string   { return "" }
func (noGPU) TotalVRAMMB() int {
	return 0
}
func (noGPU) UsedVRAMMB() int { return 0 }
func (noGPU) ProcessVRAMMB(int) int {
	return 0
}
func (noGPU) UtilTemp() (int, int) { return 0, 0 }

// --- NVIDIA ---------------------------------------------------------------

type nvidiaGPU struct{ run gpuCommand }

func (nvidiaGPU) ID() string     { return "nvidia" }
func (nvidiaGPU) Vendor() string { return "NVIDIA" }
func (p nvidiaGPU) Name() string {
	out, err := p.run("nvidia-smi", "--query-gpu=name", "--format=csv,noheader")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
}
func (p nvidiaGPU) TotalVRAMMB() int {
	return sumCSVInts(p.run("nvidia-smi", "--query-gpu=memory.total", "--format=csv,noheader,nounits"))
}
func (p nvidiaGPU) UsedVRAMMB() int {
	return sumCSVInts(p.run("nvidia-smi", "--query-gpu=memory.used", "--format=csv,noheader,nounits"))
}
func (p nvidiaGPU) ProcessVRAMMB(pid int) int {
	if pid <= 0 {
		return 0
	}
	out, err := p.run("nvidia-smi", "--query-compute-apps=pid,used_gpu_memory", "--format=csv,noheader,nounits")
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
		if mem, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
			total += mem
		}
	}
	return total
}
func (p nvidiaGPU) UtilTemp() (int, int) {
	out, err := p.run("nvidia-smi", "--query-gpu=utilization.gpu,temperature.gpu", "--format=csv,noheader,nounits")
	if err != nil {
		return 0, 0
	}
	util, temp := 0, 0
	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			continue
		}
		u, e1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		t, e2 := strconv.Atoi(strings.TrimSpace(parts[1]))
		if e1 == nil && u > util {
			util = u
		}
		if e2 == nil && t > temp {
			temp = t
		}
	}
	return util, temp
}

func sumCSVInts(out []byte, err error) int {
	if err != nil {
		return 0
	}
	total := 0
	for _, line := range strings.Split(string(out), "\n") {
		if v, err := strconv.Atoi(strings.TrimSpace(line)); err == nil {
			total += v
		}
	}
	return total
}

// --- AMD (rocm-smi) -------------------------------------------------------

type amdGPU struct{ run gpuCommand }

func (amdGPU) ID() string     { return "amd" }
func (amdGPU) Vendor() string { return "AMD" }
func (p amdGPU) Name() string {
	out, err := p.run("rocm-smi", "--showproductname", "--json")
	if err != nil {
		return ""
	}
	for k, v := range jsonStringKeys(out, nil) {
		lk := strings.ToLower(k)
		if strings.Contains(lk, "card series") || strings.Contains(lk, "product name") {
			return v
		}
	}
	return ""
}
func (p amdGPU) TotalVRAMMB() int {
	vals := jsonNumberKeys(p.rocm())
	return sumMatching(vals, "vram total memory", true)
}
func (p amdGPU) UsedVRAMMB() int {
	vals := jsonNumberKeys(p.rocm())
	return sumMatching(vals, "vram total used", true)
}
func (p amdGPU) ProcessVRAMMB(int) int { return 0 } // not exposed by rocm-smi
func (p amdGPU) UtilTemp() (int, int) {
	vals := jsonNumberKeys(p.rocm())
	return maxMatching(vals, "gpu use"), maxMatching(vals, "temperature")
}
func (p amdGPU) rocm() ([]byte, error) {
	return p.run("rocm-smi", "--showmeminfo", "vram", "--showuse", "--showtemp", "--json")
}

// --- Intel (xpu-smi) ------------------------------------------------------

type intelGPU struct{ run gpuCommand }

func (intelGPU) ID() string     { return "intel" }
func (intelGPU) Vendor() string { return "Intel" }
func (p intelGPU) Name() string {
	for k, v := range jsonStringKeys(p.run("xpu-smi", "discovery", "-j")) {
		if strings.Contains(strings.ToLower(k), "device_name") {
			return v
		}
	}
	return ""
}
func (p intelGPU) TotalVRAMMB() int {
	vals := jsonNumberKeys(p.run("xpu-smi", "discovery", "-j"))
	total := sumMatching(vals, "memory_physical_size", true)
	if total == 0 {
		total = sumMatching(vals, "total_memory", true)
	}
	return total
}
func (p intelGPU) UsedVRAMMB() int {
	vals := jsonNumberKeys(p.run("xpu-smi", "stats", "-d", "0", "-j"))
	return sumMatching(vals, "memory_used", true)
}
func (p intelGPU) ProcessVRAMMB(int) int { return 0 }
func (p intelGPU) UtilTemp() (int, int) {
	vals := jsonNumberKeys(p.run("xpu-smi", "stats", "-d", "0", "-j"))
	return maxMatching(vals, "gpu_utilization"), maxMatching(vals, "gpu_temperature")
}

// --- Apple Silicon (unified memory) ---------------------------------------

type appleGPU struct{ run gpuCommand }

func (appleGPU) ID() string     { return "apple" }
func (appleGPU) Vendor() string { return "Apple" }
func (appleGPU) Name() string   { return "Apple Silicon (unified memory)" }
func (p appleGPU) TotalVRAMMB() int {
	out, err := p.run("sysctl", "-n", "hw.memsize")
	if err != nil {
		return 0
	}
	bytes, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil || bytes <= 0 {
		return 0
	}
	return int(bytes / (1024 * 1024))
}
func (appleGPU) UsedVRAMMB() int       { return 0 }
func (appleGPU) ProcessVRAMMB(int) int { return 0 }
func (appleGPU) UtilTemp() (int, int)  { return 0, 0 }

// --- JSON helpers ---------------------------------------------------------

// jsonNumberKeys flattens a vendor JSON document to key -> numeric value. Both
// {"card0":{"VRAM Total Memory (B)":"17163"}} and arrays of objects work.
func jsonNumberKeys(out []byte, err error) map[string]float64 {
	vals := map[string]float64{}
	if err != nil || len(out) == 0 {
		return vals
	}
	var v any
	if json.Unmarshal(out, &v) != nil {
		return vals
	}
	flattenNumbers(v, vals)
	return vals
}

// jsonStringKeys flattens a vendor JSON document to key -> string value.
func jsonStringKeys(out []byte, err error) map[string]string {
	vals := map[string]string{}
	if err != nil || len(out) == 0 {
		return vals
	}
	var v any
	if json.Unmarshal(out, &v) != nil {
		return vals
	}
	flattenStrings(v, vals)
	return vals
}

func flattenStrings(v any, out map[string]string) {
	switch t := v.(type) {
	case map[string]any:
		for k, x := range t {
			switch n := x.(type) {
			case map[string]any, []any:
				flattenStrings(n, out)
			case string:
				out[k] = strings.TrimSpace(n)
			}
		}
	case []any:
		for _, x := range t {
			flattenStrings(x, out)
		}
	}
}

func flattenNumbers(v any, out map[string]float64) {
	switch t := v.(type) {
	case map[string]any:
		for k, x := range t {
			switch n := x.(type) {
			case map[string]any, []any:
				flattenNumbers(n, out)
			case string:
				if f, err := strconv.ParseFloat(strings.TrimSpace(n), 64); err == nil {
					out[k] = f
				}
			case float64:
				out[k] = n
			}
		}
	case []any:
		for _, x := range t {
			flattenNumbers(x, out)
		}
	}
}

// sumMatching adds values whose lower-cased key contains sub. byteKeys converts
// bytes to MiB when the key says "byte" / "(b)".
func sumMatching(vals map[string]float64, sub string, byteKeys bool) int {
	total := 0.0
	for k, v := range vals {
		lk := strings.ToLower(k)
		if strings.Contains(lk, sub) {
			total += maybeMiB(lk, v, byteKeys)
		}
	}
	return int(total)
}

func maxMatching(vals map[string]float64, sub string) int {
	max := 0.0
	for k, v := range vals {
		if strings.Contains(strings.ToLower(k), sub) && v > max {
			max = v
		}
	}
	return int(max)
}

func maybeMiB(key string, v float64, byteKeys bool) float64 {
	if !byteKeys {
		return v
	}
	if strings.Contains(key, "byte") || strings.Contains(key, "(b)") {
		return v / (1024 * 1024)
	}
	return v
}
