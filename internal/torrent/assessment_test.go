package torrent

import "testing"

func TestAutomaticDownloadEligibilityRequiresHighVerifiedConfidence(t *testing.T) {
	tests := []struct {
		name       string
		assessment ReleaseAssessment
		eligible   bool
	}{
		{
			name: "high verified",
			assessment: ReleaseAssessment{
				Confidence:   ConfidenceHigh,
				Verification: VerificationVerified,
			},
			eligible: true,
		},
		{
			name: "high unverified magnet",
			assessment: ReleaseAssessment{
				Confidence:   ConfidenceHigh,
				Verification: VerificationUnverified,
				Reasons:      []AssessmentReason{{Code: ReasonUnverifiableMagnet}},
			},
		},
		{
			name: "medium verified",
			assessment: ReleaseAssessment{
				Confidence:   ConfidenceMedium,
				Verification: VerificationVerified,
			},
		},
		{
			name: "high verified but hard rejected",
			assessment: ReleaseAssessment{
				Confidence:     ConfidenceHigh,
				Verification:   VerificationVerified,
				HardRejections: []AssessmentReason{{Code: ReasonBlockedPayload}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.assessment.EligibleForAutomaticDownload(); got != test.eligible {
				t.Fatalf("EligibleForAutomaticDownload() = %v, want %v", got, test.eligible)
			}
		})
	}
}

func TestReleaseAssessmentRejected(t *testing.T) {
	if !(ReleaseAssessment{Confidence: ConfidenceRejected}).Rejected() {
		t.Fatal("rejected confidence must be rejected")
	}
	if !(ReleaseAssessment{Confidence: ConfidenceHigh, HardRejections: []AssessmentReason{{Code: ReasonKnownBadInfoHash}}}).Rejected() {
		t.Fatal("hard rejection must reject assessment")
	}
	if (ReleaseAssessment{Confidence: ConfidenceHigh}).Rejected() {
		t.Fatal("high confidence without hard rejection must not be rejected")
	}
}

func TestApplyPayloadVerificationRejectsExecutablePayload(t *testing.T) {
	base := ReleaseAssessment{Confidence: ConfidenceHigh, Verification: VerificationUnverified}
	candidate := SearchResult{InfoHash: "abc"}
	metadata := TorrentMetadata{
		InfoHashV1: "abc",
		Files: []TorrentFile{
			{Path: "episode.mkv", Size: 1000},
			{Path: "setup.exe", Size: 50},
		},
		TotalSize: 1050,
	}
	assessment := ApplyPayloadVerification(base, candidate, metadata)
	if assessment.Verification != VerificationVerified || assessment.Confidence != ConfidenceRejected {
		t.Fatalf("unexpected assessment: %+v", assessment)
	}
	if !assessment.Rejected() || assessment.EligibleForAutomaticDownload() {
		t.Fatal("blocked payload must not be eligible for automatic download")
	}
	if assessment.Payload == nil || assessment.Payload.VideoFiles != 1 || assessment.Payload.ExecutableFiles != 1 {
		t.Fatalf("unexpected payload inspection: %+v", assessment.Payload)
	}
}

func TestApplyPayloadVerificationRejectsInfoHashMismatch(t *testing.T) {
	assessment := ApplyPayloadVerification(
		ReleaseAssessment{Confidence: ConfidenceHigh},
		SearchResult{InfoHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		TorrentMetadata{
			InfoHashV1: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			Files:      []TorrentFile{{Path: "episode.mkv", Size: 1000}},
			TotalSize:  1000,
		},
	)
	if !assessment.Rejected() || assessment.Confidence != ConfidenceRejected {
		t.Fatalf("infohash mismatch was not rejected: %+v", assessment)
	}
}

func TestResolvedFileVerificationPreservesKnownMagnetInfoHash(t *testing.T) {
	hash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	base := ReleaseAssessment{
		Confidence: ConfidenceHigh,
		Verification: VerificationUnverified,
		InfoHash: hash,
		Parsed: ParsedRelease{Title: "example show", Season: 1, Episode: 2},
	}
	files := []TorrentFile{{Path: "Example.Show.S01E02.mkv", Size: 2 * 1024 * 1024 * 1024}}
	assessment, err := VerifyResolvedFilesForTarget(base, hash, "Example.Show.S01E02", files, EpisodeTarget{
		ShowTitle: "Example Show", Season: 1, Episode: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if assessment.InfoHash != hash {
		t.Fatalf("resolved verification lost known magnet infohash: %q", assessment.InfoHash)
	}
}

func TestApplyPayloadIdentityVerificationStillRejectsDifferentShow(t *testing.T) {
	assessment := ApplyPayloadIdentityVerification(
		ReleaseAssessment{Confidence: ConfidenceHigh, Verification: VerificationVerified},
		TorrentMetadata{
			Name:  "Different.Show.S01E02",
			Files: []TorrentFile{{Path: "Different.Show.S01E02.mkv", Size: 1000}},
		},
		EpisodeTarget{ShowTitle: "Example Show", Season: 1, Episode: 2},
	)
	if !assessment.Rejected() || !hasReason(assessment.HardRejections, ReasonWrongShow) {
		t.Fatalf("explicit payload show mismatch was not rejected: %+v", assessment)
	}
}

