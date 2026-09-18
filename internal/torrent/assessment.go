package torrent

import (
	"fmt"
	"math"
	"path"
	"strings"
)

type Confidence string

const (
	ConfidenceHigh     Confidence = "high"
	ConfidenceMedium   Confidence = "medium"
	ConfidenceLow      Confidence = "low"
	ConfidenceRejected Confidence = "rejected"
)

type VerificationState string

const (
	VerificationUnverified VerificationState = "unverified"
	VerificationVerified   VerificationState = "verified"
)

type AssessmentReasonCode string

const (
	ReasonShowMatch             AssessmentReasonCode = "show_match"
	ReasonEpisodeMatch          AssessmentReasonCode = "episode_match"
	ReasonHealthySwarm          AssessmentReasonCode = "healthy_swarm"
	ReasonExpectedSize          AssessmentReasonCode = "expected_size"
	ReasonSizeOutOfRange        AssessmentReasonCode = "size_out_of_range"
	ReasonVideoPayload          AssessmentReasonCode = "video_payload"
	ReasonNoBlockedPayload      AssessmentReasonCode = "no_blocked_payload"
	ReasonKnownBadInfoHash      AssessmentReasonCode = "known_bad_infohash"
	ReasonInfoHashMismatch      AssessmentReasonCode = "infohash_mismatch"
	ReasonWrongShow             AssessmentReasonCode = "wrong_show"
	ReasonWrongEpisode          AssessmentReasonCode = "wrong_episode"
	ReasonAmbiguousIdentity     AssessmentReasonCode = "ambiguous_identity"
	ReasonUnsupportedPack       AssessmentReasonCode = "unsupported_pack"
	ReasonMissingVideo          AssessmentReasonCode = "missing_video"
	ReasonSampleOnly            AssessmentReasonCode = "sample_only"
	ReasonBlockedPayload        AssessmentReasonCode = "blocked_payload"
	ReasonUnverifiableMagnet    AssessmentReasonCode = "unverifiable_magnet"
	ReasonTorrentMetadataFailed AssessmentReasonCode = "torrent_metadata_failed"
)

type AssessmentReason struct {
	Code   AssessmentReasonCode `json:"code"`
	Detail string               `json:"detail,omitempty"`
}

type ParsedRelease struct {
	Title        string `json:"title,omitempty"`
	Season       int    `json:"season,omitempty"`
	Episode      int    `json:"episode,omitempty"`
	Resolution   string `json:"resolution,omitempty"`
	Source       string `json:"source,omitempty"`
	Codec        string `json:"codec,omitempty"`
	Group        string `json:"group,omitempty"`
	MultiEpisode bool   `json:"multi_episode,omitempty"`
	SeasonPack   bool   `json:"season_pack,omitempty"`
}

type TorrentFile struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

type PayloadInspection struct {
	Files           []TorrentFile `json:"files,omitempty"`
	VideoFiles      int           `json:"video_files"`
	SubtitleFiles   int           `json:"subtitle_files"`
	ArchiveFiles    int           `json:"archive_files"`
	ExecutableFiles int           `json:"executable_files"`
	TotalSize       int64         `json:"total_size"`
	MainVideoSize   int64         `json:"main_video_size"`
}

type ReleaseAssessment struct {
	Confidence     Confidence         `json:"confidence"`
	Verification   VerificationState  `json:"verification"`
	Reasons        []AssessmentReason `json:"reasons,omitempty"`
	HardRejections []AssessmentReason `json:"hard_rejections,omitempty"`
	Parsed         ParsedRelease      `json:"parsed"`
	Payload        *PayloadInspection `json:"payload,omitempty"`
	InfoHash       string             `json:"info_hash,omitempty"`

	// Score is intentionally internal. User-facing UI should present the
	// confidence band and reasons rather than a misleading percentage.
	Score int `json:"-"`
}

func (a ReleaseAssessment) Rejected() bool {
	return a.Confidence == ConfidenceRejected || len(a.HardRejections) > 0
}

func (a ReleaseAssessment) EligibleForAutomaticDownload() bool {
	return a.Confidence == ConfidenceHigh && a.Verification == VerificationVerified && len(a.HardRejections) == 0
}

func VerifyTorrentBytes(base ReleaseAssessment, candidate SearchResult, data []byte) (ReleaseAssessment, error) {
	return VerifyTorrentForTarget(base, candidate, data, nil)
}

func VerifyTorrentForTarget(base ReleaseAssessment, candidate SearchResult, data []byte, target *EpisodeTarget) (ReleaseAssessment, error) {
	metadata, err := ParseTorrentMetadata(data)
	if err != nil {
		base.Verification = VerificationUnverified
		base.Reasons = appendReason(base.Reasons, AssessmentReason{Code: ReasonTorrentMetadataFailed})
		return base, fmt.Errorf("torrent metadata could not be inspected")
	}
	assessment := ApplyPayloadVerification(base, candidate, metadata)
	if target != nil && !assessment.Rejected() {
		assessment = ApplyPayloadIdentityVerification(assessment, metadata, *target)
	}
	return assessment, verificationError(assessment)
}


func VerifyResolvedFilesForTarget(base ReleaseAssessment, infohash, name string, files []TorrentFile, target EpisodeTarget) (ReleaseAssessment, error) {
	if len(files) == 0 || len(files) > maxTorrentFiles {
		return base, fmt.Errorf("resolved torrent metadata contains no usable files")
	}
	metadata := TorrentMetadata{Name: name, Files: append([]TorrentFile(nil), files...)}
	for _, file := range files {
		if file.Path == "" || file.Size < 0 || metadata.TotalSize > math.MaxInt64-file.Size {
			return base, fmt.Errorf("resolved torrent file metadata is invalid")
		}
		metadata.TotalSize += file.Size
	}
	base.InfoHash = normalizeInfoHash(infohash)
	assessment := ApplyPayloadVerification(base, SearchResult{InfoHash: base.InfoHash}, metadata)
	if !assessment.Rejected() {
		assessment = ApplyPayloadIdentityVerification(assessment, metadata, target)
	}
	return assessment, verificationError(assessment)
}

func ApplyPayloadVerification(base ReleaseAssessment, candidate SearchResult, metadata TorrentMetadata) ReleaseAssessment {
	inspection := InspectTorrentPayload(metadata)
	base.Verification = VerificationVerified
	base.Payload = &inspection
	if metadata.InfoHashV1 != "" {
		base.InfoHash = metadata.InfoHashV1
	} else if metadata.InfoHashV2 != "" {
		base.InfoHash = metadata.InfoHashV2
	}

	if candidate.InfoHash != "" && metadata.InfoHashV1 != "" && !strings.EqualFold(candidate.InfoHash, metadata.InfoHashV1) {
		base.HardRejections = appendReason(base.HardRejections, AssessmentReason{Code: ReasonInfoHashMismatch})
	}
	if inspection.VideoFiles == 0 {
		base.HardRejections = appendReason(base.HardRejections, AssessmentReason{Code: ReasonMissingVideo})
	} else {
		base.Reasons = appendReason(base.Reasons, AssessmentReason{Code: ReasonVideoPayload})
	}
	if inspection.ExecutableFiles > 0 {
		base.HardRejections = appendReason(base.HardRejections, AssessmentReason{Code: ReasonBlockedPayload, Detail: "executable_file"})
	} else {
		base.Reasons = appendReason(base.Reasons, AssessmentReason{Code: ReasonNoBlockedPayload})
	}
	if inspection.VideoFiles > 0 && payloadIsSampleOnly(metadata.Files) {
		base.HardRejections = appendReason(base.HardRejections, AssessmentReason{Code: ReasonSampleOnly})
	}
	if len(base.HardRejections) > 0 {
		base.Confidence = ConfidenceRejected
	}
	return base
}

func ApplyPayloadIdentityVerification(base ReleaseAssessment, metadata TorrentMetadata, target EpisodeTarget) ReleaseAssessment {
	exactEpisode := false
	explicitIdentity := false
	wrongShow := false
	wrongEpisode := false
	unsupportedPack := false
	identities := []string{metadata.Name}
	for _, file := range metadata.Files {
		if isVideoExtension(strings.ToLower(path.Ext(file.Path))) {
			identities = append(identities, strings.TrimSuffix(path.Base(file.Path), path.Ext(file.Path)))
		}
	}
	for _, identity := range identities {
		parsed := ParseReleaseName(identity)
		if parsed.MultiEpisode || parsed.SeasonPack {
			unsupportedPack = true
			continue
		}
		if parsed.Season == 0 && parsed.Episode == 0 && target.Season != 0 {
			continue
		}
		explicitIdentity = true
		if parsed.Season != target.Season || parsed.Episode != target.Episode {
			wrongEpisode = true
			continue
		}
		exactEpisode = true
		if parsed.Title != "" && !releaseTitleMatchesTarget(parsed.Title, target) {
			// Once the actual torrent payload exposes an explicit episode title,
			// a mismatching show name is strong enough evidence to reject it.
			wrongShow = true
		}
	}

	switch {
	case unsupportedPack:
		base.HardRejections = appendReason(base.HardRejections, AssessmentReason{Code: ReasonUnsupportedPack, Detail: "payload"})
	case wrongEpisode && !exactEpisode:
		base.HardRejections = appendReason(base.HardRejections, AssessmentReason{Code: ReasonWrongEpisode, Detail: "payload"})
	case wrongShow:
		base.HardRejections = appendReason(base.HardRejections, AssessmentReason{Code: ReasonWrongShow, Detail: "payload"})
	case exactEpisode:
		base.Reasons = appendReason(base.Reasons, AssessmentReason{Code: ReasonEpisodeMatch, Detail: "payload"})
	case !explicitIdentity:
		base.Reasons = appendReason(base.Reasons, AssessmentReason{Code: ReasonAmbiguousIdentity, Detail: "payload_episode_not_parsed"})
		if base.Confidence == ConfidenceHigh {
			base.Confidence = ConfidenceMedium
		}
	}
	if len(base.HardRejections) > 0 {
		base.Confidence = ConfidenceRejected
	}
	return base
}

func RejectKnownBadInfoHash(base ReleaseAssessment) ReleaseAssessment {
	base.HardRejections = appendReason(base.HardRejections, AssessmentReason{Code: ReasonKnownBadInfoHash})
	base.Confidence = ConfidenceRejected
	return base
}

func payloadIsSampleOnly(files []TorrentFile) bool {
	videos := 0
	for _, file := range files {
		if !isVideoExtension(strings.ToLower(path.Ext(file.Path))) {
			continue
		}
		videos++
		name := strings.ToLower(strings.TrimSuffix(path.Base(file.Path), path.Ext(file.Path)))
		if !strings.Contains(name, "sample") && !strings.Contains(name, "trailer") {
			return false
		}
	}
	return videos > 0
}

func appendReason(list []AssessmentReason, reason AssessmentReason) []AssessmentReason {
	for _, existing := range list {
		if existing.Code == reason.Code && existing.Detail == reason.Detail {
			return list
		}
	}
	return append(list, reason)
}

func verificationError(assessment ReleaseAssessment) error {
	if !assessment.Rejected() {
		return nil
	}
	for _, reason := range assessment.HardRejections {
		switch reason.Code {
		case ReasonInfoHashMismatch:
			return fmt.Errorf("torrent metadata did not match the selected result")
		case ReasonKnownBadInfoHash:
			return fmt.Errorf("torrent was previously marked bad")
		case ReasonBlockedPayload:
			return fmt.Errorf("torrent contains executable or script files")
		case ReasonMissingVideo:
			return fmt.Errorf("torrent contains no supported video files")
		case ReasonSampleOnly:
			return fmt.Errorf("torrent contains only sample or trailer video files")
		case ReasonSizeOutOfRange:
			return fmt.Errorf("torrent payload size is outside the configured MB per minute range")
		case ReasonWrongShow:
			return fmt.Errorf("torrent payload appears to contain a different show")
		case ReasonWrongEpisode:
			return fmt.Errorf("torrent payload appears to contain a different episode")
		case ReasonUnsupportedPack:
			return fmt.Errorf("torrent payload is a season or multi-episode pack")
		}
	}
	return fmt.Errorf("torrent failed inspection")
}
