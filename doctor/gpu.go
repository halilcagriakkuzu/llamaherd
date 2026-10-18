package doctor

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
)

func checkGPU() Check {
	c := Check{ID: "gpu", Title: "GPU / CUDA"}
	out, err := execOut("nvidia-smi", "--query-gpu=name,memory.total", "--format=csv,noheader,nounits")
	if err != nil || out == "" {
		c.Status = StatusWarn
		if runtime.GOOS == "darwin" {
			c.Detail = "no NVIDIA GPU (CUDA is NVIDIA-only; Apple Metal is used automatically)"
			c.Fix = "nothing to do — llama.cpp uses Metal on macOS"
			return c
		}
		c.Detail = "no NVIDIA GPU detected (nvidia-smi not found or failed)"
		c.Fix = "install the NVIDIA driver; CPU-only inference also works"
		return c
	}
	var total int
	var names []string
	for _, line := range strings.Split(out, "\n") {
		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			continue
		}
		mb, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
		total += mb
		names = append(names, fmt.Sprintf("%s (%s)", strings.TrimSpace(parts[0]), gib(mb)))
	}
	if total == 0 {
		c.Status = StatusWarn
		c.Detail = "GPU present but total memory could not be read"
		c.Fix = "VRAM estimates fall back to model size and may be off"
		return c
	}
	c.Status = StatusOK
	c.Detail = fmt.Sprintf("%d GPU(s), %s total: %s", len(names), gib(total), strings.Join(names, "; "))
	return c
}

func gib(mb int) string { return fmt.Sprintf("%.1f GiB", float64(mb)/1024) }
