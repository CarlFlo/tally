package flows

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/CarlFlo/tally/internal/automationchain"
	"github.com/CarlFlo/tally/internal/settings"
)

type Block struct {
	ID     string            `json:"id"`
	Type   string            `json:"type"`
	Config map[string]string `json:"config"`
}
type Definition struct {
	Blocks []Block `json:"blocks"`
}
type Flow struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	ShowID     string     `json:"show_id,omitempty"`
	Revision   int        `json:"revision"`
	Definition Definition `json:"definition"`
	CreatedAt  int64      `json:"created_at"`
	UpdatedAt  int64      `json:"updated_at"`
}

// The stages are deliberately fixed. The scheduler has one guarded search and
// submission path, so a saved chain can configure it without bypassing checks.
var stages = []string{"query.build", "jackett.search", "torrent.filter", "torrent.best", "action.download"}

type BlockKind struct {
	Type   string   `json:"type"`
	Config []string `json:"config"`
}

func Kinds() []BlockKind {
	return []BlockKind{{stages[0], []string{"title_override", "prefix", "suffix"}}, {stages[1], []string{}}, {stages[2], []string{"min_seeders", "include_keywords", "exclude_keywords", "allowed_groups", "allowed_uploaders", "min_mb_per_minute", "max_mb_per_minute", "release_delay_minutes"}}, {stages[3], []string{"preferred_quality", "prefer_smaller", "preferred_groups", "preferred_uploaders", "preferred_providers", "max_candidates"}}, {stages[4], []string{}}}
}
func DefaultDefinition() Definition {
	blocks := make([]Block, len(stages))
	for i, stage := range stages {
		blocks[i] = Block{ID: fmt.Sprintf("step-%d", i+1), Type: stage, Config: map[string]string{}}
	}
	return Definition{Blocks: blocks}
}

func DefaultProfileDefinition(profile string, config settings.TorrentAutomation) Definition {
	d := DefaultDefinition()
	config = config.Effective()
	minimum, maximum := config.LiveMinMBPerMinute, config.LiveMaxMBPerMinute
	if profile == "animated" {
		minimum, maximum = config.AnimatedMinMBPerMinute, config.AnimatedMaxMBPerMinute
	}
	d.Blocks[2].Config = map[string]string{
		"min_seeders":           fmt.Sprint(config.MinSeeders),
		"include_keywords":      config.IncludeKeywords,
		"exclude_keywords":      config.ExcludeKeywords,
		"allowed_groups":        automationchain.Join(config.AllowedGroups),
		"allowed_uploaders":     automationchain.Join(config.AllowedUploaders),
		"min_mb_per_minute":     fmt.Sprint(minimum),
		"max_mb_per_minute":     fmt.Sprint(maximum),
		"release_delay_minutes": fmt.Sprint(config.ReleaseDelayMinutes),
	}
	d.Blocks[3].Config = map[string]string{
		"preferred_quality":   config.PreferredQuality,
		"prefer_smaller":      fmt.Sprint(config.PreferSmaller),
		"preferred_groups":    automationchain.Join(config.PreferredGroups),
		"preferred_uploaders": automationchain.Join(config.PreferredUploaders),
		"preferred_providers": automationchain.Join(config.PreferredProviders),
		"max_candidates":      fmt.Sprint(config.MaxCandidates),
	}
	return d
}
func Validate(name string, definition Definition) error {
	if len(strings.TrimSpace(name)) < 1 || len(name) > 100 {
		return errors.New("chain name must be 1–100 characters")
	}
	if len(definition.Blocks) != len(stages) {
		return errors.New("chain requires the five automation stages")
	}
	ids := map[string]bool{}
	for i, block := range definition.Blocks {
		if block.Type != stages[i] || !validID(block.ID) || ids[block.ID] {
			return fmt.Errorf("invalid block at step %d", i+1)
		}
		ids[block.ID] = true
		allowed := map[string]bool{}
		for _, key := range Kinds()[i].Config {
			allowed[key] = true
		}
		for key, value := range block.Config {
			limit := 200
			if key == "include_keywords" || key == "exclude_keywords" {
				limit = 500
			}
			if strings.HasSuffix(key, "_groups") || strings.HasSuffix(key, "_uploaders") || key == "preferred_providers" {
				limit = 8192
			}
			if !allowed[key] || len(value) > limit {
				return fmt.Errorf("invalid configuration for step %d", i+1)
			}
		}
		if block.Type == "torrent.filter" {
			v := block.Config["min_seeders"]
			if v != "" {
				var n int
				if _, err := fmt.Sscanf(v, "%d", &n); err != nil || n < 0 || n > 1_000_000 || fmt.Sprint(n) != v {
					return errors.New("min seeders must be 0–1000000")
				}
			}
		}
	}
	for _, profile := range []string{"live", "animated"} {
		if _, err := automationchain.Apply(settings.DefaultTorrentAutomation(), definition.Blocks[2].Config, definition.Blocks[3].Config, profile); err != nil {
			return err
		}
	}
	return nil
}
func validID(id string) bool {
	if len(id) < 1 || len(id) > 64 {
		return false
	}
	for _, c := range id {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

// Older saved editor definitions are converted at the storage boundary. The
// fixed stages retain the settings that can safely affect scheduled runs.
func decodeDefinition(raw string) (Definition, error) {
	var d Definition
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		return d, err
	}
	if len(d.Blocks) > 0 {
		return d, nil
	}
	var old struct {
		Nodes []struct {
			Type   string            `json:"type"`
			Config map[string]string `json:"config"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal([]byte(raw), &old); err != nil {
		return d, err
	}
	d = DefaultDefinition()
	for _, node := range old.Nodes {
		for i := range d.Blocks {
			if d.Blocks[i].Type == node.Type {
				for _, key := range Kinds()[i].Config {
					if value, ok := node.Config[key]; ok {
						d.Blocks[i].Config[key] = value
					}
				}
				break
			}
		}
	}
	return d, nil
}
