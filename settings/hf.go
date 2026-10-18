package settings

import (
	"os"
	"path/filepath"
	"strings"
)

// Hugging Face token resolution. The token is stored in a 0600 file next to the
// user config (never in config.json, which is world-readable, and never in a
// preset). Precedence: HF_TOKEN / HUGGING_FACE_HUB_TOKEN env -> the llamaherd
// token file -> the standard Hugging Face cache written by `hf auth login`.

// HFTokenFile is the llamaherd-managed token file path.
func HFTokenFile() string {
	return filepath.Join(filepath.Dir(ConfigFile()), "hf_token")
}

// standardHFTokenFile is where the official CLI stores a login token.
func standardHFTokenFile() string {
	if v := strings.TrimSpace(os.Getenv("HF_TOKEN_PATH")); v != "" {
		return expandHome(v)
	}
	if v := strings.TrimSpace(os.Getenv("HF_HOME")); v != "" {
		return filepath.Join(v, "token")
	}
	if h := homeDir(); h != "" {
		return filepath.Join(h, ".cache", "huggingface", "token")
	}
	return ""
}

// HFToken returns the Hugging Face access token, or "" when anonymous.
func HFToken() string {
	for _, e := range []string{"HF_TOKEN", "HUGGING_FACE_HUB_TOKEN"} {
		if v := strings.TrimSpace(os.Getenv(e)); v != "" {
			return v
		}
	}
	for _, p := range []string{HFTokenFile(), standardHFTokenFile()} {
		if p == "" {
			continue
		}
		if b, err := os.ReadFile(p); err == nil {
			if t := strings.TrimSpace(string(b)); t != "" {
				return t
			}
		}
	}
	return ""
}

// HFTokenSource reports where the active token came from ("env", "file",
// "hf-cache" or "" when none).
func HFTokenSource() string {
	for _, e := range []string{"HF_TOKEN", "HUGGING_FACE_HUB_TOKEN"} {
		if strings.TrimSpace(os.Getenv(e)) != "" {
			return "env"
		}
	}
	if p := HFTokenFile(); p != "" {
		if b, err := os.ReadFile(p); err == nil && strings.TrimSpace(string(b)) != "" {
			return "file"
		}
	}
	if p := standardHFTokenFile(); p != "" {
		if b, err := os.ReadFile(p); err == nil && strings.TrimSpace(string(b)) != "" {
			return "hf-cache"
		}
	}
	return ""
}

// SaveHFToken writes the token to the llamaherd token file with 0600 perms.
// An empty token removes the file.
func SaveHFToken(token string) error {
	p := HFTokenFile()
	token = strings.TrimSpace(token)
	if token == "" {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(token+"\n"), 0o600)
}
