package cli

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/codcod/jerry/internal/config"
)

// updateGolden regenerates the committed fixtures, following the same
// convention as internal/rules/rules_test.go and internal/index/index_test.go
// (decision 5, JRY-007) — `just test-update` already runs `go test ./... -update`,
// so this package needs no separate wiring.
var updateGolden = flag.Bool("update", false, "regenerate golden fixtures")

// goldenResult is what a case's golden file pins: the two observable output
// channels and whether the command failed. Decision 6 (JRY-007): main.go maps
// every non-nil error to os.Exit(1) uniformly, so a numeric exit code would
// only restate that constant — `failed` is the only distinction that exists.
type goldenResult struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	Failed bool   `json:"failed"`
}

// runCLI builds a fresh, isolated tree (newRoot) and runs it in-process
// against root, per decision 3: g.configPath is set directly rather than
// threading --config through args. now() is pinned for the duration
// (decision 4).
//
// `init` is the one leaf that resolves its target from os.Getwd() rather than
// g.configPath (internal/cli/init.go) — an assumption the plan did not carry,
// found during this pickup (see the ticket's plan-amended-inline History
// line). chdir accommodates it via t.Chdir, which auto-restores.
func runCLI(t *testing.T, root string, chdir bool, args ...string) goldenResult {
	t.Helper()

	restore := now
	now = func() time.Time { return goldenClock }
	t.Cleanup(func() { now = restore })

	if chdir {
		t.Chdir(root)
	}

	tree, g := newRoot("golden")
	g.configPath = filepath.Join(root, config.DefaultFile)
	tree.SetArgs(args)

	stdout, stderr, failed := captureStreams(t, tree.Execute)

	// The fixture root is a t.TempDir(), a fresh path every run; hooks
	// install/uninstall/status print it verbatim (internal/hooks/hooks.go's
	// Path), so it must be normalised before the output can be a stable
	// golden file.
	normalise := func(s string) string { return strings.ReplaceAll(s, root, "<root>") }
	return goldenResult{
		Stdout: normalise(stdout),
		Stderr: normalise(stderr),
		Failed: failed,
	}
}

// captureStreams runs fn with the real os.Stdout/os.Stderr swapped for pipes,
// then returns what each captured plus whether fn returned an error.
//
// This is the F1 rework fix (see the ticket's Review): cobra's Print*/Println
// resolve through OutOrStderr(), which is `getOut(os.Stderr)` — it returns
// c.outWriter (the SetOut buffer) whenever one is set, never c.errWriter. A
// tree with both SetOut and SetErr configured therefore funnels every
// cmd.Print* call into the SetOut buffer regardless of which real stream
// production code would use, making the two channels indistinguishable in a
// golden file. cli.Execute never calls SetOut/SetErr in production, so
// OutOrStdout()/OutOrStderr() fall through to the literal os.Stdout/os.Stderr
// package variables — swapping those, rather than cobra's own seam, is what
// actually reproduces the real split.
func captureStreams(t *testing.T, fn func() error) (stdout, stderr string, failed bool) {
	t.Helper()

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating stdout pipe: %v", err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating stderr pipe: %v", err)
	}

	origStdout, origStderr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = stdoutW, stderrW

	var stdoutBuf, stderrBuf bytes.Buffer
	copied := make(chan struct{}, 2)
	go func() { io.Copy(&stdoutBuf, stdoutR); copied <- struct{}{} }()
	go func() { io.Copy(&stderrBuf, stderrR); copied <- struct{}{} }()

	runErr := fn()

	// Restore before closing the write ends: a later case must never
	// observe the pipe, and closing is what lets the copying goroutines see
	// EOF and return.
	os.Stdout, os.Stderr = origStdout, origStderr
	stdoutW.Close()
	stderrW.Close()
	<-copied
	<-copied
	stdoutR.Close()
	stderrR.Close()

	return stdoutBuf.String(), stderrBuf.String(), runErr != nil
}

// goldenCase is one command invocation pinned as a golden file. leafPath is
// the cobra CommandPath() of the leaf it exercises — TestGoldenCoversEveryLeaf
// checks every real leaf is named by at least one case.
type goldenCase struct {
	name     string
	leafPath string
	fixture  func(t *testing.T) string
	chdir    bool
	args     []string
}

var goldenCases = []goldenCase{
	{
		name:     "version",
		leafPath: "jerry version",
		fixture:  cleanFixture,
		args:     []string{"version"},
	},
	{
		name:     "init-dry-run",
		leafPath: "jerry init",
		fixture:  func(t *testing.T) string { return t.TempDir() },
		chdir:    true,
		args:     []string{"init", "--dry-run"},
	},
	{
		name:     "new-adr-dry-run",
		leafPath: "jerry new adr",
		fixture:  cleanFixture,
		args:     []string{"new", "adr", "Test Decision", "--team", "example-team", "--deciders", "Golden Author", "--dry-run"},
	},
	{
		name:     "new-sd-dry-run",
		leafPath: "jerry new sd",
		fixture:  cleanFixture,
		args:     []string{"new", "sd", "Test Design", "--team", "example-team", "--authors", "Golden Author", "--dry-run"},
	},
	{
		name:     "validate-clean",
		leafPath: "jerry validate",
		fixture:  cleanFixture,
		args:     []string{"validate"},
	},
	{
		name:     "validate-dirty",
		leafPath: "jerry validate",
		fixture:  dirtyFixture,
		args:     []string{"validate"},
	},
	{
		name:     "fmt-check",
		leafPath: "jerry fmt",
		fixture:  cleanFixture,
		args:     []string{"fmt", "--check"},
	},
	{
		name:     "index-check",
		leafPath: "jerry index",
		fixture:  cleanFixture,
		args:     []string{"index", "--check"},
	},
	{
		name:     "supersede-dry-run",
		leafPath: "jerry supersede",
		fixture:  cleanFixture,
		args:     []string{"supersede", "example-team/ADR-0001", "--with", "New Decision", "--dry-run"},
	},
	{
		name:     "status-real",
		leafPath: "jerry status",
		fixture:  cleanFixture,
		args:     []string{"status", "example-team/ADR-0001", "Deprecated"},
	},
	{
		name:     "schema",
		leafPath: "jerry schema",
		fixture:  cleanFixture,
		args:     []string{"schema"},
	},
	{
		name:     "hooks-install-dry-run",
		leafPath: "jerry hooks install",
		fixture:  cleanFixture,
		args:     []string{"hooks", "install", "--dry-run"},
	},
	{
		name:     "hooks-uninstall-dry-run",
		leafPath: "jerry hooks uninstall",
		fixture:  cleanFixture,
		args:     []string{"hooks", "uninstall", "--dry-run"},
	},
	{
		name:     "hooks-status",
		leafPath: "jerry hooks status",
		fixture:  cleanFixture,
		args:     []string{"hooks", "status"},
	},
}

// TestGolden pins stdout, stderr and failure for every case in goldenCases
// against internal/cli/testdata/golden/<name>.json, following the
// updateGolden convention internal/rules and internal/index already use.
func TestGolden(t *testing.T) {
	// Resolved before any subtest runs: the init-dry-run case chdirs the
	// process (runCLI's chdir path) for the lifetime of its subtest, and a
	// path relative to that point would resolve under the fixture's temp
	// dir instead of this package.
	pkgDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	for _, tc := range goldenCases {
		t.Run(tc.name, func(t *testing.T) {
			root := tc.fixture(t)
			got := runCLI(t, root, tc.chdir, tc.args...)

			rendered, err := json.MarshalIndent(got, "", "  ")
			if err != nil {
				t.Fatalf("marshalling result: %v", err)
			}
			rendered = append(rendered, '\n')

			golden := filepath.Join(pkgDir, "testdata", "golden", tc.name+".json")
			if *updateGolden {
				if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
					t.Fatalf("creating golden dir: %v", err)
				}
				if err := os.WriteFile(golden, rendered, 0o644); err != nil {
					t.Fatalf("writing fixture: %v", err)
				}
				return
			}

			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("reading fixture (run: go test ./internal/cli -update): %v", err)
			}
			if string(rendered) != string(want) {
				t.Fatalf("committed fixture does not match rendered output.\nRun: go test ./internal/cli -update\n--- got ---\n%s\n--- want ---\n%s",
					rendered, want)
			}
		})
	}
}

// TestGoldenCoversEveryLeaf asserts every real, non-generated leaf in the
// cobra tree is named by at least one goldenCases entry — so a new leaf
// command ships with no golden case as a test failure, not a silent gap.
// leaves() and the generated filter are reused verbatim from
// write_safety_test.go; per the JRY-007 pre-flight audit, the filter is
// applied at the call site rather than inside leaves() itself, so it must be
// repeated here rather than assumed.
func TestGoldenCoversEveryLeaf(t *testing.T) {
	root, _ := newRoot("test")

	covered := make(map[string]bool, len(goldenCases))
	for _, tc := range goldenCases {
		covered[tc.leafPath] = true
	}

	for _, cmd := range leaves(root) {
		if generated[cmd.Name()] || generated[cmd.Parent().Name()] {
			continue
		}
		if !covered[cmd.CommandPath()] {
			t.Errorf("%s has no golden case in goldenCases", cmd.CommandPath())
		}
	}
}
