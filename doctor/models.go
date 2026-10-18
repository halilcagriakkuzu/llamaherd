package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"llamaherd/settings"
)

func checkModelsDir() Check {
	c := Check{ID: "models-dir", Title: "Models directory"}
	d := settings.ModelsDir()
	st, err := os.Stat(d)
	if err != nil || !st.IsDir() {
		c.Status = StatusFail
		c.Detail = "missing: " + d
		c.Fix = "add *.gguf files, or set LLAMAHERD_MODELS_DIR=/path/to/models\n" +
			"  (or set \"modelsDir\" in " + settings.ConfigFile() + ")"
		return c
	}
	if _, err := os.ReadDir(d); err != nil {
		c.Status = StatusFail
		c.Detail = "not readable: " + d
		c.Fix = "check permissions: ls -ld " + d
		return c
	}
	n, mmp := countModels(d)
	if n == 0 {
		c.Status = StatusWarn
		c.Detail = "no *.gguf models found under " + d
		c.Fix = "download a model (e.g. from Hugging Face) into " + d
		return c
	}
	c.Status = StatusOK
	c.Detail = fmt.Sprintf("%d model(s), %d mmproj sidecar(s) under %s", n, mmp, d)
	return c
}

func countModels(root string) (models, mmp int) {
	filepath.WalkDir(root, func(p string, e os.DirEntry, err error) error {
		if err != nil || e.IsDir() {
			return nil
		}
		name := strings.ToLower(e.Name())
		if !strings.HasSuffix(name, ".gguf") {
			return nil
		}
		if strings.HasPrefix(name, "mmproj") {
			mmp++
		} else {
			models++
		}
		return nil
	})
	return
}
