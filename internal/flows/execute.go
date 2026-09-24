package flows

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/CarlFlo/tally/internal/torrent"
)

type Event struct {
	SourceRunID string `json:"source_run_id,omitempty"`
	ShowID      string `json:"show_id"`
	EpisodeID   string `json:"episode_id"`
	ShowName    string `json:"show_name"`
	Season      int    `json:"season"`
	Episode     int    `json:"episode"`
	TriggeredAt int64  `json:"triggered_at"`
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
type value struct {
	event     Event
	text      string
	results   []torrent.SearchResult
	candidate torrent.SearchResult
	trace     any
}
type executor func(context.Context, Node, value, Event, SearchFunc) (map[string]value, string, error)

var registry = map[string]NodeKind{
	"trigger.episode":   {Type: "trigger.episode", Outputs: []Port{{"event", "event"}}, execute: executeTrigger},
	"trigger.manual":    {Type: "trigger.manual", Outputs: []Port{{"event", "event"}}, execute: executeTrigger},
	"query.build":       {Type: "query.build", Inputs: []Port{{"event", "event"}}, Outputs: []Port{{"query", "text"}}, Config: []string{"prefix"}, execute: executeQuery},
	"text.replace":      {Type: "text.replace", Inputs: []Port{{"text", "text"}}, Outputs: []Port{{"text", "text"}}, Config: []string{"find", "replace"}, execute: executeReplace},
	"jackett.search":    {Type: "jackett.search", Inputs: []Port{{"query", "text"}}, Outputs: []Port{{"results", "results"}, {"no_results", "results"}, {"error", "results"}}, execute: executeSearch},
	"torrent.filter":    {Type: "torrent.filter", Inputs: []Port{{"results", "results"}}, Outputs: []Port{{"results", "results"}}, Config: []string{"min_seeders"}, execute: executeFilter},
	"logic.has_results": {Type: "logic.has_results", Inputs: []Port{{"results", "results"}}, Outputs: []Port{{"true", "results"}, {"false", "results"}}, execute: executeHasResults},
	"torrent.best":      {Type: "torrent.best", Inputs: []Port{{"results", "results"}}, Outputs: []Port{{"candidate", "candidate"}, {"no_candidate", "results"}}, execute: executeBest},
	"action.download":   {Type: "action.download", Inputs: []Port{{"candidate", "candidate"}}, execute: executeDownload},
	"action.log":        {Type: "action.log", Inputs: []Port{{"candidate", "candidate"}}, execute: executeLog},
	"action.stop":       {Type: "action.stop", Inputs: []Port{{"results", "results"}}, execute: executeStop},
}

func config(node Node, key string) string {
	var v string
	_ = json.Unmarshal(node.Config[key], &v)
	return v
}
func executeTrigger(_ context.Context, _ Node, _ value, event Event, _ SearchFunc) (map[string]value, string, error) {
	return map[string]value{"event": {event: event, trace: event}}, "Trigger received", nil
}
func executeQuery(_ context.Context, node Node, _ value, event Event, _ SearchFunc) (map[string]value, string, error) {
	query := strings.TrimSpace(config(node, "prefix") + " " + fmt.Sprintf("%s S%02dE%02d", event.ShowName, event.Season, event.Episode))
	return map[string]value{"query": {text: query, trace: query}}, query, nil
}
func executeReplace(_ context.Context, node Node, in value, _ Event, _ SearchFunc) (map[string]value, string, error) {
	find := config(node, "find")
	if find == "" {
		return nil, "", fmt.Errorf("find text is required")
	}
	result := strings.ReplaceAll(in.text, find, config(node, "replace"))
	return map[string]value{"text": {text: result, trace: result}}, result, nil
}
func executeSearch(ctx context.Context, _ Node, in value, _ Event, search SearchFunc) (map[string]value, string, error) {
	if search == nil {
		return nil, "", fmt.Errorf("Jackett is unavailable")
	}
	results, err := search(ctx, in.text)
	if err != nil {
		return map[string]value{"error": {trace: Results{Candidates: []Candidate{}, Total: 0}}}, "Jackett search failed", err
	}
	if len(results) > 100 {
		results = results[:100]
	}
	port := "results"
	if len(results) == 0 {
		port = "no_results"
	}
	return map[string]value{port: {results: results, trace: summarize(results, nil)}}, fmt.Sprintf("%d results", len(results)), nil
}
func summarize(results []torrent.SearchResult, rejected []Rejection) Results {
	out := Results{Candidates: make([]Candidate, 0, min(len(results), 20)), Rejected: rejected, Total: len(results)}
	for _, result := range results {
		if len(out.Candidates) >= 20 {
			break
		}
		out.Candidates = append(out.Candidates, Candidate{Name: result.Name, Seeders: result.Seeders, Size: result.Size, Provider: result.Provider})
	}
	return out
}
func executeFilter(_ context.Context, node Node, in value, event Event, _ SearchFunc) (map[string]value, string, error) {
	minimum := 1
	if raw := config(node, "min_seeders"); raw != "" {
		n, e := strconv.Atoi(raw)
		if e != nil || n < 0 || n > 10000 {
			return nil, "", fmt.Errorf("min seeders must be 0–10000")
		}
		minimum = n
	}
	target := torrent.EpisodeTarget{ShowTitle: event.ShowName, Season: event.Season, Episode: event.Episode}
	accepted := make([]torrent.SearchResult, 0, len(in.results))
	rejected := make([]Rejection, 0)
	counts := make(map[string]int)
	for _, result := range in.results {
		reasons := make([]string, 0)
		if result.Seeders < minimum {
			reasons = append(reasons, "below_min_seeders")
		}
		assessment := torrent.EvaluateSearchCandidate(result, target)
		for _, reason := range assessment.HardRejections {
			reasons = append(reasons, string(reason.Code))
		}
		if assessment.Confidence != torrent.ConfidenceHigh && len(reasons) == 0 {
			reasons = append(reasons, "confidence_too_low")
		}
		if len(reasons) > 0 {
			for _, reason := range reasons {
				counts[reason]++
			}
			if len(rejected) < 20 {
				rejected = append(rejected, Rejection{Name: result.Name, Reasons: reasons})
			}
			continue
		}
		accepted = append(accepted, result)
	}
	trace := summarize(accepted, rejected)
	trace.RejectionCounts = counts
	for i := range trace.Candidates {
		trace.Candidates[i].Confidence = torrent.ConfidenceHigh
	}
	return map[string]value{"results": {results: accepted, trace: trace}}, fmt.Sprintf("%d of %d acceptable", len(accepted), len(in.results)), nil
}
func executeHasResults(_ context.Context, _ Node, in value, _ Event, _ SearchFunc) (map[string]value, string, error) {
	port := "false"
	if len(in.results) > 0 {
		port = "true"
	}
	return map[string]value{port: in}, port, nil
}
func executeBest(_ context.Context, _ Node, in value, _ Event, _ SearchFunc) (map[string]value, string, error) {
	if len(in.results) == 0 {
		return map[string]value{"no_candidate": in}, "No candidate", nil
	}
	items := append([]torrent.SearchResult(nil), in.results...)
	sort.SliceStable(items, func(i, j int) bool { return items[i].Seeders > items[j].Seeders })
	chosen := items[0]
	candidate := Candidate{Name: chosen.Name, Seeders: chosen.Seeders, Size: chosen.Size, Provider: chosen.Provider, Confidence: torrent.ConfidenceHigh}
	return map[string]value{"candidate": {candidate: chosen, trace: candidate}}, candidate.Name, nil
}
func executeDownload(_ context.Context, _ Node, in value, _ Event, _ SearchFunc) (map[string]value, string, error) {
	return nil, "Would inspect and submit if verified: " + in.candidate.Name, nil
}
func executeLog(_ context.Context, _ Node, in value, _ Event, _ SearchFunc) (map[string]value, string, error) {
	return nil, "Would log " + in.candidate.Name, nil
}
func executeStop(_ context.Context, _ Node, _ value, _ Event, _ SearchFunc) (map[string]value, string, error) {
	return nil, "Stopped", nil
}

// Execute is bounded even after graph validation. It never performs side effects.
func Execute(ctx context.Context, flow Flow, event Event, search SearchFunc) Run {
	start := time.Now()
	run := Run{FlowID: flow.ID, FlowRevision: flow.Revision, SourceRunID: event.SourceRunID, Event: event, Definition: flow.Definition, Steps: []Step{}, Status: "completed", StartedAt: start.Unix()}
	if err := Validate(flow.Name, flow.Definition); err != nil {
		run.Status = "failed"
		run.Steps = append(run.Steps, Step{Status: "failed", Summary: err.Error()})
		return run
	}
	type pending struct {
		node  string
		input value
	}
	queue := make([]pending, 0, 40)
	for _, node := range flow.Definition.Nodes {
		if strings.HasPrefix(node.Type, "trigger.") {
			queue = append(queue, pending{node: node.ID})
		}
	}
	nodes := map[string]Node{}
	for _, node := range flow.Definition.Nodes {
		nodes[node.ID] = node
	}
	visited := map[string]bool{}
	for len(queue) > 0 && len(run.Steps) < 40 {
		if ctx.Err() != nil {
			run.Status = "failed"
			break
		}
		current := queue[0]
		queue = queue[1:]
		if visited[current.node] {
			continue
		}
		visited[current.node] = true
		node := nodes[current.node]
		started := time.Now()
		outputs, summary, err := registry[node.Type].execute(ctx, node, current.input, event, search)
		step := Step{NodeID: node.ID, Status: "successful", Input: current.input.trace, Summary: summary, DurationMS: time.Since(started).Milliseconds()}
		if err != nil {
			step.Status = "failed"
			if node.Type == "jackett.search" {
				step.Summary = "Jackett search failed"
			} else {
				step.Summary = err.Error()
			}
			run.Status = "failed"
		}
		if node.Type == "action.stop" {
			step.Status = "stopped"
			if run.Status != "failed" {
				run.Status = "stopped"
			}
		}
		for _, output := range outputs {
			step.Output = output.trace
			break
		}
		run.Steps = append(run.Steps, step)
		for _, edge := range flow.Definition.Edges {
			if edge.Source == node.ID {
				if output, ok := outputs[edge.SourcePort]; ok {
					queue = append(queue, pending{node: edge.Target, input: output})
				}
			}
		}
	}
	if len(queue) > 0 {
		run.Status = "failed"
		run.Steps = append(run.Steps, Step{Status: "failed", Summary: "execution limit reached"})
	}
	if run.Status == "completed" {
		for _, step := range run.Steps {
			if strings.HasPrefix(nodes[step.NodeID].Type, "torrent.best") && step.Summary == "No candidate" {
				run.Status = "no_candidate"
			}
			if nodes[step.NodeID].Type == "jackett.search" && step.Summary == "0 results" {
				run.Status = "no_candidate"
			}
		}
	}
	for _, node := range flow.Definition.Nodes {
		if !visited[node.ID] {
			run.Steps = append(run.Steps, Step{NodeID: node.ID, Status: "skipped"})
		}
	}
	run.DurationMS = time.Since(start).Milliseconds()
	return run
}
