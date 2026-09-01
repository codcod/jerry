// Package scaffold writes a new architecture-docs repository.
//
// Every file is embedded in the binary, so `jerry init` needs no network and
// no template repository to clone. The emitted repo deliberately contains no
// scripts/ directory: jerry owns the rules, which is what stops them drifting
// between the repositories that share them.
package scaffold

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/codcod/jerry/internal/config"
	"github.com/codcod/jerry/internal/doc"
	"github.com/codcod/jerry/internal/index"
)

//go:embed all:templates
var templates embed.FS

// Forge selects the CI and CODEOWNERS variant.
type Forge string

const (
	ForgeGitLab Forge = "gitlab"
	ForgeGitHub Forge = "github"
)

// Forges lists the supported values, for flag help and error messages.
var Forges = []string{string(ForgeGitLab), string(ForgeGitHub)}

// ParseForge validates a user-supplied forge name.
func ParseForge(value string) (Forge, error) {
	switch Forge(value) {
	case ForgeGitLab:
		return ForgeGitLab, nil
	case ForgeGitHub:
		return ForgeGitHub, nil
	default:
		return "", fmt.Errorf("unknown forge %q (want one of: %s)", value, strings.Join(Forges, ", "))
	}
}

// versionToken is replaced with the version of the jerry that wrote the
// scaffold, so the emitted CI pins the exact binary its rules came from
// instead of floating on @latest.
const versionToken = "__JERRY_VERSION__"

// Options controls one init run.
type Options struct {
	Root    string
	Forge   Forge
	Version string
	Force   bool
	DryRun  bool
}

// Result records what happened, so the caller can print it and tests can
// assert on it without re-reading the filesystem.
type Result struct {
	Created []string
	Skipped []string
	// Generated lists files rebuilt from the corpus on every run rather than
	// copied once. The index is regenerated even when init is re-run over an
	// existing repo, because it describes the documents, not the scaffold.
	Generated []string
}

// Run writes the scaffold.
func Run(options Options) (*Result, error) {
	files, err := collect(options.Forge)
	if err != nil {
		return nil, err
	}
	// jerry.yaml comes from config.Starter rather than a template file, so the
	// starter config and the defaults it documents cannot drift apart.
	files[config.DefaultFile] = []byte(config.Starter)

	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)

	result := &Result{}
	for _, name := range names {
		target := filepath.Join(options.Root, filepath.FromSlash(name))
		if _, err := os.Stat(target); err == nil && !options.Force {
			result.Skipped = append(result.Skipped, name)
			continue
		}
		content := replaceTokens(files[name], options.Version)
		if options.DryRun {
			result.Created = append(result.Created, name)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(target, content, 0o644); err != nil {
			return nil, err
		}
		result.Created = append(result.Created, name)
	}

	if options.DryRun {
		result.Generated = append(result.Generated, index.DefaultPath)
		return result, nil
	}
	// The index is generated, never embedded: a checked-in template index
	// would be stale the moment the example ADR changed, and a freshly
	// scaffolded repo would fail its own `jerry index --check` on the first
	// pipeline. It is written last, so it describes what was actually emitted.
	if err := writeIndex(options.Root); err != nil {
		return nil, err
	}
	result.Generated = append(result.Generated, index.DefaultPath)
	return result, nil
}

func writeIndex(root string) error {
	corpus, err := doc.Load(root, doc.DefaultLayout())
	if err != nil {
		return err
	}
	rendered := index.Markdown(index.Build(corpus, index.DefaultPath))
	target := filepath.Join(root, filepath.FromSlash(index.DefaultPath))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, []byte(rendered), 0o644)
}

// collect reads the common tree plus the selected forge's overlay. A forge
// file may not collide with a common one — that would make which layer wins
// depend on map iteration order, so it is an error rather than a silent choice.
func collect(forge Forge) (map[string][]byte, error) {
	files := map[string][]byte{}
	for _, layer := range []string{"common", string(forge)} {
		root := path.Join("templates", layer)
		err := fs.WalkDir(templates, root, func(current string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			content, err := templates.ReadFile(current)
			if err != nil {
				return err
			}
			name := strings.TrimPrefix(current, root+"/")
			if _, clash := files[name]; clash {
				return fmt.Errorf("scaffold template %q is defined in more than one layer", name)
			}
			files[name] = content
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}

func replaceTokens(content []byte, version string) []byte {
	pin := version
	// A development build has no released version to pin to, so it falls back
	// to latest rather than emitting CI that cannot resolve.
	if pin == "" || pin == "dev" || strings.Contains(pin, "-dirty") {
		pin = "latest"
	} else if !strings.HasPrefix(pin, "v") {
		pin = "v" + pin
	}
	return []byte(strings.ReplaceAll(string(content), versionToken, pin))
}

// Print writes a human summary of a run.
func Print(w io.Writer, result *Result, dryRun bool) {
	if dryRun {
		fmt.Fprintln(w, "(dry run — nothing was written)")
	}
	for _, name := range result.Created {
		verb := "+"
		if dryRun {
			verb = "~"
		}
		fmt.Fprintf(w, "  %s %s\n", verb, name)
	}
	for _, name := range result.Skipped {
		fmt.Fprintf(w, "  = %s (exists)\n", name)
	}
	for _, name := range result.Generated {
		fmt.Fprintf(w, "  * %s (generated)\n", name)
	}
}
