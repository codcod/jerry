package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// nestedGitFixture builds a bare git repo at t.TempDir() with the corpus
// under a docs/ subdirectory, one committed file inside it (tagged as the
// origin/main base), then a second commit changing changedRelPath (relative
// to the docs/ corpus root) — the minimal fixture changedFiles' own tests
// need, independent of the scaffolded nestedCorpusFixture golden case.
func nestedGitFixture(t *testing.T, changedRelPath, changedContent string) (corpusRoot string) {
	t.Helper()
	gitRoot := t.TempDir()
	corpusRoot = filepath.Join(gitRoot, "docs")
	if err := os.MkdirAll(corpusRoot, 0o755); err != nil {
		t.Fatalf("creating docs dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(corpusRoot, "base.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatalf("writing base file: %v", err)
	}
	gitInit(t, gitRoot)
	gitCommit(t, gitRoot, "base")
	if out, err := exec.Command("git", "-C", gitRoot, "update-ref", "refs/remotes/origin/main", "HEAD").CombinedOutput(); err != nil {
		t.Fatalf("git update-ref origin/main %s: %v\n%s", gitRoot, err, out)
	}

	path := filepath.Join(corpusRoot, filepath.FromSlash(changedRelPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("creating changed file dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(changedContent), 0o644); err != nil {
		t.Fatalf("writing changed file: %v", err)
	}
	gitCommit(t, gitRoot, "change")
	return corpusRoot
}

// TestChangedFilesRewritesToCorpusRoot is the direct regression test for
// JRY-013's live defect: git diff --name-only always reports paths relative
// to its own top level, but changedFiles must return paths relative to root
// (the corpus root, which here sits below the git top level).
func TestChangedFilesRewritesToCorpusRoot(t *testing.T) {
	corpusRoot := nestedGitFixture(t, "changed.md", "changed\n")

	changed, err := changedFiles(corpusRoot, "origin/main")
	if err != nil {
		t.Fatalf("changedFiles: %v", err)
	}
	if !changed["changed.md"] {
		t.Fatalf("expected \"changed.md\" (corpus-relative) in changed set, got %v", changed)
	}
}

// TestChangedFilesMissingBaseFailsClearly pins JRY-013 decision 3: the
// returned error must carry git's own failure reason, not just the bare
// exec exit status.
func TestChangedFilesMissingBaseFailsClearly(t *testing.T) {
	root := t.TempDir()
	gitInit(t, root)
	if err := os.WriteFile(filepath.Join(root, "base.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatalf("writing base file: %v", err)
	}
	gitCommit(t, root, "init") // no origin/main ref created

	_, err := changedFiles(root, "origin/main")
	if err == nil {
		t.Fatal("expected an error for a missing base ref, got nil")
	}
	if !strings.Contains(err.Error(), "unknown revision") {
		t.Fatalf("expected error to carry git's own reason (\"unknown revision\"), got: %v", err)
	}
}

// TestChangedFilesDetachedHead locks in behaviour already correct today so
// it can't regress silently: git diff --name-only resolves HEAD the same way
// regardless of attachment.
func TestChangedFilesDetachedHead(t *testing.T) {
	corpusRoot := nestedGitFixture(t, "changed.md", "changed\n")
	gitRoot := filepath.Dir(corpusRoot)
	if out, err := exec.Command("git", "-C", gitRoot, "checkout", "--detach", "--quiet").CombinedOutput(); err != nil {
		t.Fatalf("git checkout --detach: %v\n%s", err, out)
	}

	changed, err := changedFiles(corpusRoot, "origin/main")
	if err != nil {
		t.Fatalf("changedFiles on a detached HEAD: %v", err)
	}
	if !changed["changed.md"] {
		t.Fatalf("expected \"changed.md\" in changed set, got %v", changed)
	}
}

// TestResolveDiffBase covers JRY-013 decision 4: an explicit --base always
// wins; GITHUB_BASE_REF only fills in when the flag was left at its default.
func TestResolveDiffBase(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		envBase string
		want    string
	}{
		{
			name:    "explicit flag wins over GITHUB_BASE_REF",
			args:    []string{"--base", "origin/develop"},
			envBase: "main",
			want:    "origin/develop",
		},
		{
			name:    "autodetects from GITHUB_BASE_REF when base not given",
			args:    nil,
			envBase: "main",
			want:    "origin/main",
		},
		{
			name:    "falls back to the flag default with no env set",
			args:    nil,
			envBase: "",
			want:    "origin/main",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("GITHUB_BASE_REF", tc.envBase)

			var base string
			cmd := &cobra.Command{Run: func(*cobra.Command, []string) {}}
			cmd.Flags().StringVar(&base, "base", "origin/main", "")
			if err := cmd.ParseFlags(tc.args); err != nil {
				t.Fatalf("parsing flags: %v", err)
			}

			if got := resolveDiffBase(cmd, base); got != tc.want {
				t.Fatalf("resolveDiffBase() = %q, want %q", got, tc.want)
			}
		})
	}
}
