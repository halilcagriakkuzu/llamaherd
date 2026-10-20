package doctor

import (
	"fmt"
	"os"

	"llamaherd/settings"
)

func checkDisk() Check {
	c := Check{ID: "disk-space", Title: "Free disk space"}
	target := settings.ModelsDir()
	if _, err := os.Stat(target); err != nil {
		target = settings.DataDir()
	}
	free, total, ok := diskFree(target)
	if !ok {
		c.Status, c.Detail = StatusSkip, "could not read free space for "+target
		return c
	}
	c.Detail = fmt.Sprintf("%s free of %s (%s)", bytesGiB(free), bytesGiB(total), target)
	if free < 10<<30 {
		c.Status = StatusWarn
		c.Fix = "free space or point LLAMAHERD_MODELS_DIR at a larger volume"
		return c
	}
	c.Status = StatusOK
	return c
}

func bytesGiB(b uint64) string { return fmt.Sprintf("%.1f GiB", float64(b)/(1<<30)) }

// DiskFree reports the free and total bytes of the volume holding path.
func DiskFree(path string) (free, total uint64, ok bool) { return diskFree(path) }
