package jobs

import "context"

// runResult keeps job outcomes named at the call site and persistence boundary.
type runResult struct {
	Candidates, Processed, Skipped, Changes, Attempt int
}

func (s *Service) run(ctx context.Context, id, kind, trigger, show string) (runResult, error) {
	switch kind {
	case "backup":
		return s.runBackup(ctx, trigger)
	case "maintenance":
		return s.runMaintenance(ctx)
	case "metadata":
		return s.runMetadata(ctx, id, trigger, show)
	case "torrent_automation":
		if s.TorrentAutomation == nil {
			return runResult{Attempt: 1}, nil
		}
		processed, err := s.TorrentAutomation.Run(ctx)
		return runResult{Processed: processed, Changes: processed, Attempt: 1}, err
	default:
		return runResult{Attempt: 1}, nil
	}
}
