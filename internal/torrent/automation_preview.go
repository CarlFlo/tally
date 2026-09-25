package torrent

import (
	"time"

	"github.com/CarlFlo/tally/internal/settings"
)

type PreviewRejection struct {
	Name    string
	Reasons []string
}

// PreviewAutomationCandidates shares the scheduler's preliminary rule checks
// and ranking. It deliberately does not fetch torrent payloads or submit them.
func PreviewAutomationCandidates(results []SearchResult, target EpisodeTarget, config settings.TorrentAutomation, profile string, runtimeMinutes int, now time.Time) ([]SearchResult, []PreviewRejection) {
	valid := make([]automationCandidate, 0, len(results))
	rejected := make([]PreviewRejection, 0)
	for _, result := range results {
		reasons := []string{}
		if result.Seeders < config.MinSeeders {
			reasons = append(reasons, "below_min_seeders")
		}
		if !automationKeywordsMatch(result.Name, config) {
			reasons = append(reasons, "keyword_filtered")
		}
		assessment := EvaluateSearchCandidate(result, target)
		for _, reason := range assessment.HardRejections {
			reasons = append(reasons, string(reason.Code))
		}
		if !automationGroupAllowed(assessment.Parsed.Group, config) {
			reasons = append(reasons, "group_filtered")
		}
		if !automationUploaderAllowed(result.Uploader, config) {
			reasons = append(reasons, "uploader_filtered")
		}
		if assessment.Confidence != ConfidenceHigh {
			reasons = append(reasons, "confidence_too_low")
		}
		size := EvaluateSizeProfile(result.Size, runtimeMinutes, config, profile)
		if size.Known && !size.InActiveRange {
			reasons = append(reasons, "size_rate_filtered")
		}
		if len(reasons) > 0 {
			rejected = append(rejected, PreviewRejection{result.Name, reasons})
			continue
		}
		valid = append(valid, automationCandidate{Result: result, Assessment: assessment, SizeProfile: size, ReleaseAge: EvaluateReleaseAge(result.Published, now, time.Duration(config.ReleaseDelayMinutes)*time.Minute)})
	}
	if !ReleaseDelayWindowOpen(candidateReleaseAges(valid)) {
		ready := valid[:0]
		for _, candidate := range valid {
			if candidate.ReleaseAge.Known {
				rejected = append(rejected, PreviewRejection{candidate.Result.Name, []string{"release_too_new"}})
			} else {
				ready = append(ready, candidate)
			}
		}
		valid = ready
	}
	rankAutomationCandidates(valid, config)
	if len(valid) > config.MaxCandidates {
		valid = valid[:config.MaxCandidates]
	}
	selected := make([]SearchResult, 0, len(valid))
	for _, candidate := range valid {
		selected = append(selected, candidate.Result)
	}
	return selected, rejected
}
