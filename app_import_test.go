package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"llamaherd/internal/ggufgen"
	"llamaherd/settings"
	"llamaherd/store"
)

func TestImportModelWithMMProj(t *testing.T) {
	a, root := libApp(t, "existing")
	incoming := filepath.Join(root, "incoming")
	src, err := ggufgen.Write(filepath.Join(incoming, "vision-model.gguf"),
		ggufgen.Spec{Layers: 9, Heads: 8, KVHeads: 8, Embedding: 768})
	if err != nil {
		t.Fatal(err)
	}
	mm, err := ggufgen.Write(filepath.Join(incoming, "mmproj-vision-model.gguf"),
		ggufgen.Spec{Layers: 1, Embedding: 64})
	if err != nil {
		t.Fatal(err)
	}

	if err := a.ImportModel(src, mm, false); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(settings.ModelsDir(), "vision-model.gguf")
	mmDest := filepath.Join(settings.ModelsDir(), "mmproj-vision-model.gguf")
	for _, p := range []string{dest, mmDest, src, mm} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("missing %s: %v", p, err)
		}
	}
	i := a.indexOfModelPath(dest)
	if i < 0 {
		t.Fatalf("imported model is not in the list: %+v", a.models)
	}
	id := a.models[i].ID
	meta, ok := store.LoadMeta(settings.LibraryDir(), id)
	if !ok || meta.MMProj != "mmproj-vision-model.gguf" {
		t.Fatalf("library meta = %+v ok=%v", meta, ok)
	}
	if a.selModel != i {
		t.Errorf("import did not select the new model")
	}
	if got := a.cfg.Advanced.MMProj; got != mmDest {
		t.Errorf("preset mmproj = %q, want %q", got, mmDest)
	}

	// Importing the same file name again is refused, not silently overwritten.
	if err := a.ImportModel(src, "", false); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("duplicate import = %v, want an already-exists error", err)
	}
}

func TestImportModelMovesAndRejectsNonGGUF(t *testing.T) {
	a, root := libApp(t)
	src, err := ggufgen.Write(filepath.Join(root, "incoming", "moved.gguf"), ggufgen.Spec{Layers: 5})
	if err != nil {
		t.Fatal(err)
	}
	if err := a.ImportModel(src, "", true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("move left the source behind: %v", err)
	}
	if a.indexOfModelPath(filepath.Join(settings.ModelsDir(), "moved.gguf")) < 0 {
		t.Error("moved model missing")
	}

	notGGUF := filepath.Join(root, "incoming", "notes.txt")
	_ = os.WriteFile(notGGUF, []byte("x"), 0o644)
	if err := a.ImportModel(notGGUF, "", false); err == nil || !strings.Contains(err.Error(), ".gguf") {
		t.Errorf("non-gguf = %v, want a .gguf error", err)
	}
	if err := a.ImportModel("", "", false); err == nil {
		t.Error("empty source must fail")
	}
}

func TestImportModelsBatchAndDuplicates(t *testing.T) {
	a, root := libApp(t)
	inc := filepath.Join(root, "incoming")
	var srcs []string
	for _, name := range []string{"one", "two", "three"} {
		p, err := ggufgen.Write(filepath.Join(inc, name+".gguf"),
			ggufgen.Spec{Layers: 8, Heads: 8, KVHeads: 8, Embedding: 512})
		if err != nil {
			t.Fatal(err)
		}
		srcs = append(srcs, p)
	}
	rep, err := a.ImportModels(srcs, "", false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Items) != 3 {
		t.Fatalf("items = %d, want 3: %+v", len(rep.Items), rep.Items)
	}
	for _, it := range rep.Items {
		if it.Status != "imported" || it.Alias == "" {
			t.Errorf("item = %+v, want imported with an alias", it)
		}
	}
	if n := len(a.models); n != 3 {
		t.Errorf("models after batch = %d, want 3", n)
	}

	// A second import of the same content is a duplicate, not a copy.
	rep, err = a.ImportModels([]string{srcs[0]}, "", false, false)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Items[0].Status != "duplicate" || !strings.Contains(rep.Items[0].Message, "already exists") {
		t.Errorf("duplicate import = %+v", rep.Items[0])
	}
	// allowDuplicate bypasses the content check, but the destination name is
	// still taken, so the item fails instead of overwriting.
	rep, err = a.ImportModels([]string{srcs[0]}, "", false, true)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Items[0].Status != "error" || !strings.Contains(rep.Items[0].Message, "already exists") {
		t.Errorf("allowDuplicate with a taken name = %+v", rep.Items[0])
	}
}

func TestImportModelsFolderAndCheckImport(t *testing.T) {
	a, root := libApp(t)
	folder := filepath.Join(root, "incoming", "batch")
	for _, name := range []string{"alpha", "beta"} {
		if _, err := ggufgen.Write(filepath.Join(folder, name+".gguf"), ggufgen.Spec{Layers: 4}); err != nil {
			t.Fatal(err)
		}
	}
	// Sidecars inside the folder are skipped by the scan.
	if _, err := ggufgen.Write(filepath.Join(folder, "mmproj-alpha.gguf"), ggufgen.Spec{Layers: 1}); err != nil {
		t.Fatal(err)
	}

	checks := a.CheckImport([]string{folder})
	if len(checks) != 2 {
		t.Fatalf("checks = %+v, want 2 models", checks)
	}
	for _, c := range checks {
		if c.Error != "" || c.Duplicate {
			t.Errorf("fresh check = %+v", c)
		}
	}
	rep, err := a.ImportModels([]string{folder}, "", false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Items) != 2 {
		t.Fatalf("folder import = %+v", rep.Items)
	}
	if n := len(a.models); n != 2 {
		t.Errorf("models = %d, want 2", n)
	}

	// After import, the same folder is all duplicates.
	a.RescanModels()
	checks = a.CheckImport([]string{folder})
	dups := 0
	for _, c := range checks {
		if c.Duplicate && c.ExistingAlias != "" {
			dups++
		}
	}
	if dups != 2 {
		t.Errorf("duplicate checks = %+v, want 2", checks)
	}
}

func TestImportRejectsSidecarAsModel(t *testing.T) {
	a, root := libApp(t)
	side, err := ggufgen.Write(filepath.Join(root, "incoming", "mmproj-x.gguf"), ggufgen.Spec{Layers: 1})
	if err != nil {
		t.Fatal(err)
	}
	rep, err := a.ImportModels([]string{side}, "", false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Items) != 1 || rep.Items[0].Status != "error" {
		t.Fatalf("sidecar import = %+v", rep.Items)
	}
	if _, err := os.Stat(filepath.Join(settings.ModelsDir(), "mmproj-x.gguf")); err == nil {
		t.Error("sidecar must not be copied as a model")
	}
}
