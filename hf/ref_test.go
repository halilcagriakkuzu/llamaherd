package hf

import "testing"

func TestParseRef(t *testing.T) {
	cases := []struct{ in, repo, file string }{
		{"Qwen/Qwen2.5-0.5B-Instruct-GGUF", "Qwen/Qwen2.5-0.5B-Instruct-GGUF", ""},
		{"Qwen/Qwen2.5-0.5B-Instruct-GGUF/qwen2.5-0.5b-instruct-q4_k_m.gguf",
			"Qwen/Qwen2.5-0.5B-Instruct-GGUF", "qwen2.5-0.5b-instruct-q4_k_m.gguf"},
		{"https://huggingface.co/bartowski/Qwen2.5-7B-Instruct-GGUF/resolve/main/Qwen2.5-7B-Instruct-Q4_K_M.gguf",
			"bartowski/Qwen2.5-7B-Instruct-GGUF", "Qwen2.5-7B-Instruct-Q4_K_M.gguf"},
		{"https://huggingface.co/org/name/blob/main/sub/file.gguf", "org/name", "sub/file.gguf"},
		{"hf:org/name", "org/name", ""},
	}
	for _, c := range cases {
		repo, file := ParseRef(c.in)
		if repo != c.repo || file != c.file {
			t.Errorf("ParseRef(%q) = %q, %q; want %q, %q", c.in, repo, file, c.repo, c.file)
		}
	}
}
