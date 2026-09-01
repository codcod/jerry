package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// chdir moves into dir for the duration of a test, restoring the previous
// working directory afterwards. Discovery is defined in terms of the working
// directory, so it cannot be tested without one.
func chdir(t *testing.T, dir string) {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })
}

func TestDiscoverWalksUpward(t *testing.T) {
	// The point of walking up is that jerry works from anywhere inside a docs
	// repo, the way git and cargo find their manifests.
	root := t.TempDir()
	nested := filepath.Join(root, "teams", "payments", "adr")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, DefaultFile), []byte("proposed-stale-days: 30\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	chdir(t, nested)

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// macOS reports /var and /private/var for the same directory, so compare
	// resolved paths rather than the strings.
	wantRoot, _ := filepath.EvalSymlinks(root)
	gotRoot, _ := filepath.EvalSymlinks(cfg.Root)
	if gotRoot != wantRoot {
		t.Errorf("Root = %q, want %q", gotRoot, wantRoot)
	}
	if *cfg.ProposedStaleDays != 30 {
		t.Errorf("ProposedStaleDays = %d, want 30", *cfg.ProposedStaleDays)
	}
}

func TestMissingConfigIsNotAnError(t *testing.T) {
	// A repo jerry did not create must still be usable.
	dir := t.TempDir()
	chdir(t, dir)

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ADRDir != "adr" || cfg.IndexPath != "index/index.md" {
		t.Errorf("defaults were not applied: %+v", cfg)
	}
	if len(cfg.RequiredADRSections) == 0 || len(cfg.Placeholders) == 0 {
		t.Error("rule defaults were not applied")
	}
}

func TestPartialConfigKeepsOtherDefaults(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, DefaultFile), []byte("adr-dir: decisions\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	chdir(t, dir)

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ADRDir != "decisions" {
		t.Errorf("ADRDir = %q", cfg.ADRDir)
	}
	if cfg.SDDir != "solution-designs" {
		t.Errorf("overriding one key dropped another's default: SDDir = %q", cfg.SDDir)
	}
}

func TestZeroStaleDaysIsHonoured(t *testing.T) {
	// ProposedStaleDays is a *int precisely so that 0 ("warn immediately") is
	// distinguishable from "unset".
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, DefaultFile), []byte("proposed-stale-days: 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	chdir(t, dir)

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if *cfg.ProposedStaleDays != 0 {
		t.Errorf("an explicit 0 was replaced by the default %d", *cfg.ProposedStaleDays)
	}
}

func TestValidateRejectsCollidingDirs(t *testing.T) {
	dir := t.TempDir()
	content := "adr-dir: docs\nsolution-design-dir: docs\n"
	if err := os.WriteFile(filepath.Join(dir, DefaultFile), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	chdir(t, dir)

	if _, err := Load(""); err == nil {
		t.Error("two document kinds in one directory must be rejected, not silently merged")
	}
}

// TestStarterParsesAsItsOwnConfig is the guard against Starter drifting from
// the schema it documents — a starter file that fails to load is the worst
// possible first experience.
func TestStarterParsesAsItsOwnConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, DefaultFile), []byte(Starter), 0o644); err != nil {
		t.Fatal(err)
	}
	chdir(t, dir)

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("the starter config does not load: %v", err)
	}

	// Every value the starter states must match the default it claims to show.
	defaults := &Config{}
	defaults.applyDefaults()
	if cfg.ADRDir != defaults.ADRDir || cfg.SDDir != defaults.SDDir || cfg.IndexPath != defaults.IndexPath {
		t.Errorf("starter disagrees with the defaults it documents:\nstarter %+v\ndefault %+v", cfg, defaults)
	}
	if *cfg.ProposedStaleDays != *defaults.ProposedStaleDays {
		t.Errorf("starter stale-days %d, default %d", *cfg.ProposedStaleDays, *defaults.ProposedStaleDays)
	}
	if len(cfg.RequiredADRSections) != len(defaults.RequiredADRSections) {
		t.Errorf("starter required sections %v, default %v", cfg.RequiredADRSections, defaults.RequiredADRSections)
	}
}

func TestRuleOptionsCarriesTheInjectedClock(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	pinned := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	if got := cfg.RuleOptions(pinned).Today; !got.Equal(pinned) {
		t.Errorf("Today = %v, want %v", got, pinned)
	}
}
