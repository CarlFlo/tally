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
