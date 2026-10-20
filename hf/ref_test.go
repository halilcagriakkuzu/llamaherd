package hf

import (
	"strings"
	"testing"
)

func TestParseRef(t *testing.T) {
	cases := []struct{ in, repo, file string }{
		{"Qwen/Qwen2.5-0.5B-Instruct-GGUF", "Qwen/Qwen2.5-0.5B-Instruct-GGUF", ""},
		{"Qwen/Qwen2.5-0.5B-Instruct-GGUF/qwen2.5-0.5b-instruct-q4_k_m.gguf",
			"Qwen/Qwen2.5-0.5B-Instruct-GGUF", "qwen2.5-0.5b-instruct-q4_k_m.gguf"},
		{"https://huggingface.co/bartowski/Qwen2.5-7B-Instruct-GGUF/resolve/main/Qwen2.5-7B-Instruct-Q4_K_M.gguf",
			"bartowski/Qwen2.5-7B-Instruct-GGUF", "Qwen2.5-7B-Instruct-Q4_K_M.gguf"},
		{"https://huggingface.co/org/name/blob/main/sub/file.gguf", "org/name", "sub/file.gguf"},
		{"hf:org/name", "org/name", ""},
		{"  org/name  ", "org/name", ""},
		{"org/name/My%20Model.gguf", "org/name", "My Model.gguf"},
	}
	for _, c := range cases {
		repo, file, err := ParseRef(c.in)
		if err != nil {
			t.Errorf("ParseRef(%q) error: %v", c.in, err)
			continue
		}
		if repo != c.repo || file != c.file {
			t.Errorf("ParseRef(%q) = %q, %q; want %q, %q", c.in, repo, file, c.repo, c.file)
		}
	}
}

func TestParseRefRejectsUnsafeInput(t *testing.T) {
	bad := []string{
		"",
		"owner",                         // no repo name
		"owner/name/../secret.gguf",     // traversal in the file
		"../owner/name",                 // traversal in the repo
		"owner/../name",                 // traversal in the repo
		"owner/name/%2e%2e/secret.gguf", // encoded traversal
		"owner/name/..%2Fsecret.gguf",   // encoded separator + traversal
		"owner/name/file%5C.gguf",       // encoded backslash
		"owner/name//file.gguf",         // empty segment
		"owner/name/..",                 // stray traversal segment
		"owner/name/not-a-gguf",         // stray non-gguf segment
		"/owner/name",                   // absolute
		"owner/name/file.gguf\x00",      // control character
		"owner/name/file\x01.gguf",      // control character inside
		"owner/bad name/name",           // space in repo segment
		"owner/.hidden/model.gguf",      // hidden segment in repo
		"https://huggingface.co/owner/name/resolve/main/../../x.gguf",
	}
	for _, in := range bad {
		if repo, file, err := ParseRef(in); err == nil {
			t.Errorf("ParseRef(%q) = %q, %q; want an error", in, repo, file)
		}
	}
}

func TestValidateFileRef(t *testing.T) {
	for _, in := range []string{"file.gguf", "sub/dir/Model Q4.gguf", "a.b-c_d.gguf"} {
		if got, err := ValidateFileRef(in); err != nil || got != in {
			t.Errorf("ValidateFileRef(%q) = %q, %v", in, got, err)
		}
	}
	for _, in := range []string{"", "../x.gguf", "/abs.gguf", "a//b.gguf", "a\\b.gguf", "%2e%2e/x.gguf"} {
		if got, err := ValidateFileRef(in); err == nil {
			t.Errorf("ValidateFileRef(%q) = %q; want an error", in, got)
		}
	}
}

// FuzzParseRef asserts the security invariants on arbitrary input: a reference
// that parses can never contain traversal, a backslash or a non-owner/name
// repository.
func FuzzParseRef(f *testing.F) {
	seeds := []string{
		"org/name", "org/name/file.gguf", "https://huggingface.co/org/name/resolve/main/x.gguf",
		"../x", "org/name/../../etc/passwd", "%2e%2e/%2e%2e/x.gguf", "owner/name/%00.gguf",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		repo, file, err := ParseRef(in)
		if err != nil {
			return
		}
		segs := strings.Split(repo, "/")
		if len(segs) != 2 {
			t.Fatalf("repo %q is not owner/name (input %q)", repo, in)
		}
		for _, seg := range segs {
			if seg == ".." || seg == "." || strings.Contains(seg, "\\") {
				t.Fatalf("unsafe repo %q from %q", repo, in)
			}
		}
		if file != "" {
			if strings.HasPrefix(file, "/") || strings.Contains(file, "\\") ||
				strings.Contains("/"+file+"/", "/../") || strings.Contains("/"+file+"/", "/./") {
				t.Fatalf("unsafe file %q from %q", file, in)
			}
		}
	})
}
