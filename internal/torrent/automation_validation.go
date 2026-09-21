package torrent

import "fmt"

func ValidateAutomationFeedback(reason, note string) error {
	if !validFeedbackReason(reason) {
		return fmt.Errorf("invalid bad-run reason")
	}
	if len(note) > 500 {
		return fmt.Errorf("bad-run note is too long")
	}
	return nil
}

func ValidateShowPolicy(policy string) error {
	switch policy {
	case "default", "auto", "never":
		return nil
	default:
		return fmt.Errorf("invalid show automation policy")
	}
}

func ValidateShowMediaProfile(mode string) error {
	switch mode {
	case MediaProfileAuto, MediaProfileLive, MediaProfileAnimated:
		return nil
	default:
		return fmt.Errorf("invalid show media profile")
	}
}
