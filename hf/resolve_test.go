package hf

import "testing"

func TestResolveFile(t *testing.T) {
	tree := []TreeFile{
		{Path: "README.md", Size: 10},
		{Path: "mmproj-F16.gguf", Size: 100},
		{Path: "Model-Q4_K_M-00001-of-00002.gguf", Size: 1},
		{Path: "Model-Q4_K_M-00002-of-00002.gguf", Size: 1},
		{Path: "Model-Q8_0.gguf", Size: 500},
		{Path: "Model-Q4_K_M.gguf", Size: 400},
	}
	f, err := ResolveFile(tree, "")
	if err != nil || f.Path != "Model-Q4_K_M.gguf" {
		t.Fatalf("default pick = %v, %v", f, err)
	}
	if f, _ := ResolveFile(tree, "q8_0"); f.Path != "Model-Q8_0.gguf" {
		t.Fatalf("suffix pick = %v", f)
	}
	if _, err := ResolveFile(tree, "does-not-exist.gguf"); err == nil {
		t.Fatal("expected an error for a missing file")
	}
	if mm, ok := ResolveMMProj(tree, ""); !ok || mm.Path != "mmproj-F16.gguf" {
		t.Fatalf("mmproj = %v %v", mm, ok)
	}
}

func TestResolveMTPDraft(t *testing.T) {
	tree := []TreeFile{
		{Path: "README.md", Size: 10},
		{Path: "Model-Q4_K_M.gguf", Size: 400},
		{Path: "mmproj-F16.gguf", Size: 100},
		{Path: "Model-FastMTP-Q4_K_M.gguf", Size: 200},
	}
	if !IsMTPDraftFile("Model-FastMTP-Q4_K_M.gguf") {
		t.Fatal("fastmtp file not detected as an MTP draft sidecar")
	}
	if IsMTPDraftFile("Model-Q4_K_M.gguf") {
		t.Fatal("plain model wrongly detected as an MTP draft sidecar")
	}
	d, ok := ResolveMTPDraft(tree, "")
	if !ok || d.Path != "Model-FastMTP-Q4_K_M.gguf" {
		t.Fatalf("draft = %v, %v", d, ok)
	}
	if _, ok := ResolveMTPDraft([]TreeFile{{Path: "Model-Q4_K_M.gguf", Size: 1}}, ""); ok {
		t.Fatal("expected no draft sidecar")
	}
}

func TestStarter(t *testing.T) {
	s := Starter()
	if s.ID == "" || !s.Starter {
		t.Fatalf("starter = %+v", s)
	}
}
