// Package config loads jerry.yaml.
//
// Discovery walks upward from the working directory the way git, cargo and npm
// locate their manifests, so jerry works from anywhere inside a docs repo. The
// file is optional: a repo with no jerry.yaml gets the defaults `jerry init`
// would have written, which keeps the tool usable in a repo it did not create.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/codcod/jerry/internal/doc"
	"github.com/codcod/jerry/internal/rules"
)

// DefaultFile is the config filename.
const DefaultFile = "jerry.yaml"

// Config is the on-disk schema. Every field is optional; applyDefaults fills
// the gaps, so a two-line jerry.yaml is a legitimate config.
type Config struct {
	ADRDir              string   `yaml:"adr-dir"`
	SDDir               string   `yaml:"solution-design-dir"`
	SkipDirs            []string `yaml:"skip-dirs"`
	IndexPath           string   `yaml:"index-path"`
	RequiredADRSections []string `yaml:"required-adr-sections"`
	RequiredSDSections  []string `yaml:"required-sd-sections"`
	ProposedStaleDays   *int     `yaml:"proposed-stale-days"`
	Placeholders        []string `yaml:"placeholders"`

	// Root is the directory the config was found in — the repo root for every
	// path jerry resolves. It is never read from the file.
	Root string `yaml:"-"`
}

// Load reads the config at an explicit path, or the nearest one above the
// working directory. A missing file is not an error.
func Load(explicitPath string) (*Config, error) {
	path := explicitPath
	if path == "" {
		found, err := discover()
		if err != nil {
			return nil, err
		}
		path = found
	}

	config := &Config{}
	if path == "" {
		root, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		config.Root = root
	} else {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}
		if err := yaml.Unmarshal(raw, config); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		absolute, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		config.Root = filepath.Dir(absolute)
	}

	config.applyDefaults()
	if err := config.validate(); err != nil {
		return nil, err
	}
	return config, nil
}

// discover walks upward looking for jerry.yaml, returning "" when there is none.
func discover() (string, error) {
	current, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(current, DefaultFile)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", nil
		}
		current = parent
	}
}

func (c *Config) applyDefaults() {
	layout := doc.DefaultLayout()
	if c.ADRDir == "" {
		c.ADRDir = layout.ADRDir
	}
	if c.SDDir == "" {
		c.SDDir = layout.SDDir
	}
	if len(c.SkipDirs) == 0 {
		c.SkipDirs = layout.Skip
	}
	if c.IndexPath == "" {
		c.IndexPath = "index/index.md"
	}

	defaults := rules.DefaultOptions()
	if len(c.RequiredADRSections) == 0 {
		c.RequiredADRSections = defaults.RequiredADRSections
	}
	if len(c.RequiredSDSections) == 0 {
		c.RequiredSDSections = defaults.RequiredSDSections
	}
	if c.ProposedStaleDays == nil {
		days := defaults.ProposedStaleDays
		c.ProposedStaleDays = &days
	}
	if len(c.Placeholders) == 0 {
		c.Placeholders = defaults.Placeholders
	}
}

func (c *Config) validate() error {
	if c.ADRDir == c.SDDir {
		return fmt.Errorf("config: `adr-dir` and `solution-design-dir` must differ (both are %q)", c.ADRDir)
	}
	if *c.ProposedStaleDays < 0 {
		return fmt.Errorf("config: `proposed-stale-days` must not be negative (got %d)", *c.ProposedStaleDays)
	}
	return nil
}

// Layout is the corpus layout this config describes.
func (c *Config) Layout() doc.Layout {
	return doc.Layout{ADRDir: c.ADRDir, SDDir: c.SDDir, Skip: c.SkipDirs}
}

// RuleOptions is the rule set this config describes. today is injected so
// callers under test can pin the staleness clock.
func (c *Config) RuleOptions(today time.Time) rules.Options {
	return rules.Options{
		RequiredADRSections: c.RequiredADRSections,
		RequiredSDSections:  c.RequiredSDSections,
		ProposedStaleDays:   *c.ProposedStaleDays,
		Today:               today,
		Placeholders:        c.Placeholders,
	}
}

// Starter is what `jerry init` writes. It lives next to the defaults it
// documents so the two cannot drift apart.
const Starter = `# jerry.yaml — configuration for this architecture-docs repository.
# Every key is optional; the values shown are the defaults.

# Directory names jerry treats as document folders.
adr-dir: adr
solution-design-dir: solution-designs

# Directories never walked when collecting documents.
skip-dirs: [templates, index, .git, node_modules, dist]

# Where "jerry index" writes. Links are made relative to this file's directory.
index-path: index/index.md

# Sections every document must contain. Empty sections are reported too: a
# heading with nothing under it reads as though the question was answered.
required-adr-sections: ["## Context", "## Decision", "## Consequences"]
required-sd-sections: ["## Problem Statement", "## Proposed Design", "## Risks & Trade-offs"]

# Warn (never fail) when an ADR has been Proposed for longer than this.
proposed-stale-days: 90
`
