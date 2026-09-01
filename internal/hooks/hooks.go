// Package hooks installs the git pre-commit hook.
//
// Hooks are UX, not enforcement — `--no-verify` exists, so CI remains the gate.
// What the hook buys is that nobody discovers a stale index or a malformed
// document after pushing.
package hooks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Name is the hook jerry manages.
const Name = "pre-commit"

// marker identifies a hook jerry wrote, so uninstall never deletes somebody
// else's hook and status can tell the two apart.
const marker = "# installed by jerry"

const script = `#!/usr/bin/env sh
` + marker + `
set -e
jerry validate
jerry index
git add "$(jerry index --print-path)"
`

// Dir returns the repository's hooks directory, honouring core.hooksPath.
func Dir(root string) (string, error) {
	output, err := exec.Command("git", "-C", root, "rev-parse", "--git-path", "hooks").Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository (or git is unavailable): %w", err)
	}
	hooks := strings.TrimSpace(string(output))
	if !filepath.IsAbs(hooks) {
		hooks = filepath.Join(root, hooks)
	}
	return hooks, nil
}

// Path returns the hook file's location.
func Path(root string) (string, error) {
	dir, err := Dir(root)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, Name), nil
}

// Install writes the hook. An existing hook jerry did not write is left alone
// unless force is set.
func Install(root string, force bool) (string, error) {
	target, err := Path(root)
	if err != nil {
		return "", err
	}
	if existing, err := os.ReadFile(target); err == nil && !strings.Contains(string(existing), marker) && !force {
		return "", fmt.Errorf("%s already exists and was not written by jerry — inspect it, then re-run with --force", target)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(target, []byte(script), 0o755); err != nil {
		return "", err
	}
	return target, nil
}

// Uninstall removes a jerry-written hook, and refuses to remove any other.
func Uninstall(root string) (string, error) {
	target, err := Path(root)
	if err != nil {
		return "", err
	}
	existing, err := os.ReadFile(target)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !strings.Contains(string(existing), marker) {
		return "", fmt.Errorf("%s was not written by jerry — leaving it alone", target)
	}
	return target, os.Remove(target)
}

// Status reports whether the hook is installed and whether jerry owns it.
func Status(root string) (installed, owned bool, path string, err error) {
	path, err = Path(root)
	if err != nil {
		return false, false, "", err
	}
	existing, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, false, path, nil
	}
	if err != nil {
		return false, false, path, err
	}
	return true, strings.Contains(string(existing), marker), path, nil
}
