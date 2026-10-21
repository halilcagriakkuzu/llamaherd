package control

import (
	"fmt"
	"strings"
	"testing"
)

// fakeGPUCommand returns canned output keyed by "name arg1 arg2 ...".
func fakeGPUCommand(responses map[string]string) gpuCommand {
	return func(name string, args ...string) ([]byte, error) {
		key := strings.TrimSpace(name + " " + strings.Join(args, " "))
		if out, ok := responses[key]; ok {
			return []byte(out), nil
		}
		return nil, fmt.Errorf("%s: not available", name)
	}
}

func TestNVIDIAProvider(t *testing.T) {
	p := nvidiaGPU{run: fakeGPUCommand(map[string]string{
		"nvidia-smi --query-gpu=name --format=csv,noheader":                                    "NVIDIA GeForce RTX 4070 Ti SUPER\n",
		"nvidia-smi --query-gpu=memory.total --format=csv,noheader,nounits":                    "8192\n8192",
		"nvidia-smi --query-gpu=memory.used --format=csv,noheader,nounits":                     "1000",
		"nvidia-smi --query-compute-apps=pid,used_gpu_memory --format=csv,noheader,nounits":    "123, 500\n456, 100",
		"nvidia-smi --query-gpu=utilization.gpu,temperature.gpu --format=csv,noheader,nounits": "10, 50\n80, 61",
	})}
	if got := p.TotalVRAMMB(); got != 16384 {
		t.Errorf("total = %d", got)
	}
	if got := p.UsedVRAMMB(); got != 1000 {
		t.Errorf("used = %d", got)
	}
	if got := p.ProcessVRAMMB(123); got != 500 {
		t.Errorf("process = %d", got)
	}
	if u, temp := p.UtilTemp(); u != 80 || temp != 61 {
		t.Errorf("util/temp = %d/%d", u, temp)
	}
	if !strings.Contains(p.Name(), "4070") {
		t.Errorf("name = %q", p.Name())
	}
}

func TestNVIDIAMissingAndMalformed(t *testing.T) {
	missing := nvidiaGPU{run: fakeGPUCommand(nil)}
	if missing.TotalVRAMMB() != 0 || missing.UsedVRAMMB() != 0 || missing.ProcessVRAMMB(1) != 0 {
		t.Error("missing nvidia-smi must report zeros")
	}
	if u, temp := missing.UtilTemp(); u != 0 || temp != 0 {
		t.Error("missing nvidia-smi must report no util/temp")
	}
	bad := nvidiaGPU{run: fakeGPUCommand(map[string]string{
		"nvidia-smi --query-gpu=memory.total --format=csv,noheader,nounits":                    "not-a-number\nn/a",
		"nvidia-smi --query-gpu=utilization.gpu,temperature.gpu --format=csv,noheader,nounits": "?,?",
	})}
	if bad.TotalVRAMMB() != 0 {
		t.Error("malformed total must be zero")
	}
	if u, temp := bad.UtilTemp(); u != 0 || temp != 0 {
		t.Error("malformed util/temp must be zero")
	}
}

func TestAMDProvider(t *testing.T) {
	doc := `{"card0":{
		"VRAM Total Memory (B)":"8589934592",
		"VRAM Total Used Memory (B)":"1073741824",
		"GPU use (%)":"42",
		"Temperature (Sensor edge) (C)":"55.0"
	}}`
	p := amdGPU{run: fakeGPUCommand(map[string]string{
		"rocm-smi --showmeminfo vram --showuse --showtemp --json": doc,
	})}
	if got := p.TotalVRAMMB(); got != 8192 {
		t.Errorf("total = %d", got)
	}
	if got := p.UsedVRAMMB(); got != 1024 {
		t.Errorf("used = %d", got)
	}
	if u, temp := p.UtilTemp(); u != 42 || temp != 55 {
		t.Errorf("util/temp = %d/%d", u, temp)
	}
	if p.ProcessVRAMMB(1) != 0 {
		t.Error("rocm-smi per-process VRAM is not exposed")
	}
}

func TestAMDMissingAndMalformed(t *testing.T) {
	p := amdGPU{run: fakeGPUCommand(nil)}
	if p.TotalVRAMMB() != 0 || p.UsedVRAMMB() != 0 {
		t.Error("missing rocm-smi must report zeros")
	}
	bad := amdGPU{run: fakeGPUCommand(map[string]string{
		"rocm-smi --showmeminfo vram --showuse --showtemp --json": "{not json",
	})}
	if bad.TotalVRAMMB() != 0 || bad.UsedVRAMMB() != 0 {
		t.Error("malformed JSON must report zeros")
	}
}

func TestIntelProvider(t *testing.T) {
	p := intelGPU{run: fakeGPUCommand(map[string]string{
		"xpu-smi discovery -j":  `{"device_list":[{"device_name":"Intel Arc A770","memory_physical_size_byte":"8589934592"}]}`,
		"xpu-smi stats -d 0 -j": `{"device_level":[{"gpu_utilization":"20","gpu_temperature":"45","memory_used_byte":"536870912"}]}`,
	})}
	if got := p.TotalVRAMMB(); got != 8192 {
		t.Errorf("total = %d", got)
	}
	if got := p.UsedVRAMMB(); got != 512 {
		t.Errorf("used = %d", got)
	}
	if u, temp := p.UtilTemp(); u != 20 || temp != 45 {
		t.Errorf("util/temp = %d/%d", u, temp)
	}
	if p.Name() != "Intel Arc A770" {
		t.Errorf("name = %q", p.Name())
	}
}

func TestIntelMissingAndMalformed(t *testing.T) {
	p := intelGPU{run: fakeGPUCommand(nil)}
	if p.TotalVRAMMB() != 0 {
		t.Error("missing xpu-smi must report zero")
	}
	bad := intelGPU{run: fakeGPUCommand(map[string]string{"xpu-smi discovery -j": "[]"})}
	if bad.TotalVRAMMB() != 0 {
		t.Error("malformed discovery must report zero")
	}
}

func TestAppleProvider(t *testing.T) {
	p := appleGPU{run: fakeGPUCommand(map[string]string{"sysctl -n hw.memsize": "17179869184\n"})}
	if got := p.TotalVRAMMB(); got != 16384 {
		t.Errorf("unified memory = %d MiB", got)
	}
	if u, temp := p.UtilTemp(); u != 0 || temp != 0 {
		t.Error("Apple util/temp are unknown")
	}
	bad := appleGPU{run: fakeGPUCommand(map[string]string{"sysctl -n hw.memsize": "garbage"})}
	if bad.TotalVRAMMB() != 0 {
		t.Error("malformed sysctl output must report zero")
	}
}

func TestDetectGPUChoosesFirstAvailableProvider(t *testing.T) {
	prev := gpuCommandRun
	t.Cleanup(func() { gpuCommandRun = prev })
	gpuCommandRun = fakeGPUCommand(map[string]string{
		"rocm-smi --showmeminfo vram --showuse --showtemp --json": `{"card0":{"VRAM Total Memory (B)":"8589934592"}}`,
	})
	p := detectGPU()
	if p.ID() != "amd" || p.TotalVRAMMB() != 8192 {
		t.Fatalf("detected %s total=%d, want amd/8192", p.ID(), p.TotalVRAMMB())
	}
	gpuCommandRun = fakeGPUCommand(nil)
	if p := detectGPU(); p.ID() != "none" {
		t.Fatalf("no tools must detect none, got %s", p.ID())
	}
}
