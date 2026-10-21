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

func TestResolveModelSetShards(t *testing.T) {
	tree := []TreeFile{
		{Path: "BF16/Model-00002-of-00002.gguf", Size: 20},
		{Path: "BF16/Model-00001-of-00002.gguf", Size: 10},
		{Path: "Model-Q4_K_M.gguf", Size: 400},
	}
	for _, want := range []string{
		"BF16/Model-00001-of-00002.gguf",
		"Model-00001-of-00002.gguf",
		"BF16/Model.gguf",
	} {
		files, err := ResolveModelSet(tree, want)
		if err != nil {
			t.Fatalf("%s: %v", want, err)
		}
		if len(files) != 2 || files[0].Path != "BF16/Model-00001-of-00002.gguf" || files[1].Path != "BF16/Model-00002-of-00002.gguf" {
			t.Fatalf("%s: shards = %+v", want, files)
		}
	}
	one, err := ResolveModelSet(tree, "Model-Q4_K_M.gguf")
	if err != nil || len(one) != 1 || one[0].Path != "Model-Q4_K_M.gguf" {
		t.Fatalf("single file = %+v %v", one, err)
	}
	if _, err := ResolveModelSet(tree, "BF16/Model-00003-of-00003.gguf"); err == nil {
		t.Fatal("a set with missing shards must fail")
	}
}

func TestClassifyMultiPartShards(t *testing.T) {
	tree := []TreeFile{
		{Path: "BF16/Model-00001-of-00002.gguf", Size: 10},
		{Path: "BF16/Model-00002-of-00002.gguf", Size: 20},
	}
	l := classify("o/r", tree)
	if len(l.Models) != 1 {
		t.Fatalf("models = %+v", l.Models)
	}
	m := l.Models[0]
	if m.Parts != 2 || m.Size != 30 || len(m.Shards) != 2 || m.Path != "BF16/Model-00001-of-00002.gguf" {
		t.Fatalf("multi-part row = %+v", m)
	}
}
