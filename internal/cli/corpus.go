package cli

import (
	"time"

	"github.com/codcod/jerry/internal/config"
	"github.com/codcod/jerry/internal/doc"
)

// openCorpus is the single seam every command that reads documents starts
// from, so a missing or broken config fails the same way everywhere.
func openCorpus(g *globals) (*doc.Corpus, *config.Config, error) {
	cfg, err := config.Load(g.configPath)
	if err != nil {
		return nil, nil, err
	}
	corpus, err := doc.Load(cfg.Root, cfg.Layout())
	if err != nil {
		return nil, nil, err
	}
	return corpus, cfg, nil
}

// now is a package variable so tests can pin the clock the staleness rule
// reads without threading a time through every command constructor.
var now = time.Now
