package control

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// gitEnv returns an environment that keeps git from touching user/global
// config and supplies the identity commits need.
func gitEnv(home string) []string {
	return append(os.Environ(),
		"HOME="+home,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL="+filepath.Join(home, ".gitconfig"),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com",
	)
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = gitEnv(dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v in %s: %v: %s", args, dir, err, out)
	}
}

func TestIsShallowRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	if IsShallowRepo("") {
		t.Fatal("empty dir reported as shallow")
	}

	repo := t.TempDir()
	runGit(t, repo, "init", "-q")
	if err := os.WriteFile(filepath.Join(repo, "f.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-qm", "one")
	if IsShallowRepo(repo) {
		t.Fatal("full clone reported as shallow")
	}

	shallow := filepath.Join(t.TempDir(), "clone")
	// --no-local forces the git transport, which --depth needs for a path.
	runGit(t, repo, "clone", "-q", "--depth=1", "--no-local", repo, shallow)
	if !IsShallowRepo(shallow) {
		t.Fatal("shallow clone not detected")
	}
}

func TestUpdateScriptPath(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll("scripts", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("scripts", "update-llamacpp.sh"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := UpdateScriptPath()
	if err != nil {
		t.Fatalf("UpdateScriptPath: %v", err)
	}
	if filepath.Base(filepath.Dir(got)) != "scripts" ||
		filepath.Base(got) != "update-llamacpp.sh" {
		t.Fatalf("unexpected path %q", got)
	}
}
