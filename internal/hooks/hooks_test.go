package hooks

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if out, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	return root
}

func TestInstallAndUninstall(t *testing.T) {
	root := gitRepo(t)

	path, err := Install(root, false)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("hook not written: %v", err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Error("hook is not executable, so git will silently ignore it")
	}

	installed, owned, _, err := Status(root)
	if err != nil || !installed || !owned {
		t.Fatalf("Status = %v, %v, %v", installed, owned, err)
	}

	if _, err := Uninstall(root); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("hook survived uninstall")
	}
}

// TestForeignHookIsNotClobbered is the one that matters: somebody else's
// pre-commit hook is not ours to overwrite or delete.
func TestForeignHookIsNotClobbered(t *testing.T) {
	root := gitRepo(t)
	path, err := Path(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	foreign := "#!/bin/sh\necho someone else's hook\n"
	if err := os.WriteFile(path, []byte(foreign), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := Install(root, false); err == nil {
		t.Error("Install overwrote a hook jerry did not write")
	}
	if _, err := Uninstall(root); err == nil {
		t.Error("Uninstall removed a hook jerry did not write")
	}
	current, err := os.ReadFile(path)
	if err != nil || string(current) != foreign {
		t.Error("the foreign hook was modified")
	}

	t.Run("ForceOverwrites", func(t *testing.T) {
		if _, err := Install(root, true); err != nil {
			t.Fatalf("Install --force: %v", err)
		}
		content, err := os.ReadFile(path)
		if err != nil || !strings.Contains(string(content), marker) {
			t.Error("--force did not install jerry's hook")
		}
	})
}

func TestInstallIsIdempotent(t *testing.T) {
	root := gitRepo(t)
	if _, err := Install(root, false); err != nil {
		t.Fatalf("first install: %v", err)
	}
	if _, err := Install(root, false); err != nil {
		t.Errorf("re-installing jerry's own hook must be safe, got: %v", err)
	}
}

func TestUninstallWithNoHookIsNotAnError(t *testing.T) {
	root := gitRepo(t)
	path, err := Uninstall(root)
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if path != "" {
		t.Errorf("reported removing %q when nothing was installed", path)
	}
}

func TestDirFailsOutsideAGitRepository(t *testing.T) {
	if _, err := Dir(t.TempDir()); err == nil {
		t.Error("expected an error outside a git repository")
	}
}
