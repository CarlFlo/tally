package flows

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/CarlFlo/tally/internal/automationchain"
	"github.com/CarlFlo/tally/internal/settings"
	"github.com/CarlFlo/tally/internal/torrent"
)

type Event struct {
	Kind           string `json:"kind"`
	SourceRunID    string `json:"source_run_id,omitempty"`
	ShowID         string `json:"show_id"`
	EpisodeID      string `json:"episode_id"`
	ShowName       string `json:"show_name"`
	Season         int    `json:"season"`
	Episode        int    `json:"episode"`
	TriggeredAt    int64  `json:"triggered_at"`
	RuntimeMinutes int    `json:"runtime_minutes,omitempty"`
	MediaProfile   string `json:"media_profile,omitempty"`
}
type Candidate struct {
	Name       string             `json:"name"`
	Seeders    int                `json:"seeders"`
	Size       int64              `json:"size"`
	Provider   string             `json:"provider"`
	Confidence torrent.Confidence `json:"confidence,omitempty"`
}
type Rejection struct {
	Name    string   `json:"name"`
	Reasons []string `json:"reasons"`
}
type Results struct {
	Candidates      []Candidate    `json:"candidates"`
	Rejected        []Rejection    `json:"rejected,omitempty"`
	RejectionCounts map[string]int `json:"rejection_counts,omitempty"`
	Total           int            `json:"total"`
}
type Step struct {
	NodeID     string `json:"node_id"`
	Status     string `json:"status"`
	Input      any    `json:"input,omitempty"`
	Output     any    `json:"output,omitempty"`
	Summary    string `json:"summary,omitempty"`
	DurationMS int64  `json:"duration_ms"`
}
type Run struct {
	ID           string     `json:"id"`
	FlowID       string     `json:"flow_id"`
	FlowRevision int        `json:"flow_revision"`
	SourceRunID  string     `json:"source_run_id,omitempty"`
	Event        Event      `json:"event"`
	Definition   Definition `json:"definition"`
	Steps        []Step     `json:"steps"`
	Status       string     `json:"status"`
	StartedAt    int64      `json:"started_at"`
	DurationMS   int64      `json:"duration_ms"`
}
type SearchFunc func(context.Context, string) ([]torrent.SearchResult, error)

func Query(d Definition, event Event) string {
	c := d.Blocks[0].Config
	parts := []string{}
	if prefix := strings.TrimSpace(c["prefix"]); prefix != "" {
		parts = append(parts, prefix)
	}
	showName := strings.TrimSpace(c["title_override"])
	if showName == "" {
		showName = event.ShowName
	}
	parts = append(parts, showName, fmt.Sprintf("S%02dE%02d", event.Season, event.Episode))
	if suffix := strings.TrimSpace(c["suffix"]); suffix != "" {
		parts = append(parts, suffix)
	}
	return strings.Join(parts, " ")
}
func MinSeeders(d Definition, fallback int) int {
	if v := d.Blocks[2].Config["min_seeders"]; v != "" {
		n, _ := strconv.Atoi(v)
		if n > fallback {
			return n
		}
	}
	return fallback
}
func summarize(results []torrent.SearchResult) Results {
	out := Results{Candidates: []Candidate{}, Total: len(results)}
	for _, r := range results {
		if len(out.Candidates) == 20 {
			break
		}
		out.Candidates = append(out.Candidates, Candidate{Name: r.Name, Seeders: r.Seeders, Size: r.Size, Provider: r.Provider})
	}
	return out
}

// Execute previews the same ordered stages. It never contacts the downloader.
func Execute(ctx context.Context, flow Flow, event Event, search SearchFunc) Run {
	return ExecuteWithSettings(ctx, flow, event, settings.DefaultTorrentAutomation(), search)
}

func ExecuteWithSettings(ctx context.Context, flow Flow, event Event, base settings.TorrentAutomation, search SearchFunc) Run {
	start := time.Now()
	run := Run{FlowID: flow.ID, FlowRevision: flow.Revision, SourceRunID: event.SourceRunID, Event: event, Definition: flow.Definition, Steps: []Step{}, Status: "completed", StartedAt: start.Unix()}
	defer func() { run.DurationMS = time.Since(start).Milliseconds() }()
	if err := Validate(flow.Name, flow.Definition); err != nil {
		run.Status = "failed"
		run.Steps = append(run.Steps, Step{Status: "failed", Summary: err.Error()})
		return run
	}
	profile := event.MediaProfile
	if profile != "animated" {
		profile = "live"
	}
	config, err := automationchain.Apply(base, flow.Definition.Blocks[2].Config, flow.Definition.Blocks[3].Config, profile)
	if err != nil {
		run.Status = "failed"
		run.Steps = append(run.Steps, Step{Status: "failed", Summary: err.Error()})
		return run
	}
	var input any = event
	var results []torrent.SearchResult
	query := ""
	for i, block := range flow.Definition.Blocks {
		if ctx.Err() != nil {
			run.Status = "failed"
			break
		}
		began := time.Now()
		step := Step{NodeID: block.ID, Status: "successful", Input: input}
		switch i {
		case 0:
			query = Query(flow.Definition, event)
			step.Output = query
			step.Summary = query
			input = query
		case 1:
			if search == nil {
				step.Status = "failed"
				step.Summary = "Jackett is unavailable"
				run.Status = "failed"
				break
			}
			found, err := search(ctx, query)
			if err != nil {
				step.Status = "failed"
				step.Summary = "Jackett search failed"
				run.Status = "failed"
				break
			}
			if len(found) > 100 {
				found = found[:100]
			}
			results = found
			step.Output = summarize(results)
			step.Summary = fmt.Sprintf("%d results", len(results))
			input = step.Output
		case 2:
			originalCount := len(results)
			targetTitle := strings.TrimSpace(flow.Definition.Blocks[0].Config["title_override"])
			if targetTitle == "" {
				targetTitle = event.ShowName
			}
			accepted, removed := torrent.PreviewAutomationCandidates(results, torrent.EpisodeTarget{ShowTitle: targetTitle, Season: event.Season, Episode: event.Episode}, config, profile, event.RuntimeMinutes, time.Now())
			rejected := []Rejection{}
			counts := map[string]int{}
			for _, item := range removed {
				for _, reason := range item.Reasons {
					counts[reason]++
				}
				if len(rejected) < 20 {
					rejected = append(rejected, Rejection{item.Name, item.Reasons})
				}
			}
			results = accepted
			summary := summarize(results)
			summary.Rejected = rejected
			summary.RejectionCounts = counts
			for i := range summary.Candidates {
				summary.Candidates[i].Confidence = torrent.ConfidenceHigh
			}
			step.Output = summary
			step.Summary = fmt.Sprintf("%d of %d shortlisted", len(results), originalCount)
			input = summary
		case 3:
			if len(results) == 0 {
				step.Status = "no_candidate"
				step.Summary = "No candidate"
				run.Status = "no_candidate"
				break
			}
			r := results[0]
			candidate := Candidate{Name: r.Name, Seeders: r.Seeders, Size: r.Size, Provider: r.Provider, Confidence: torrent.ConfidenceHigh}
			step.Output = candidate
			step.Summary = r.Name
			input = candidate
		case 4:
			step.Summary = "Would inspect and submit if verified: " + results[0].Name
		}
		step.DurationMS = time.Since(began).Milliseconds()
		run.Steps = append(run.Steps, step)
		if run.Status != "completed" {
			for _, remaining := range flow.Definition.Blocks[i+1:] {
				run.Steps = append(run.Steps, Step{NodeID: remaining.ID, Status: "skipped"})
			}
			break
		}
	}
	run.DurationMS = time.Since(start).Milliseconds()
	return run
}
