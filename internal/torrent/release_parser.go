package torrent

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var (
	releaseEpisodePattern = regexp.MustCompile(`(?i)(?:^|[ ._\-\[])(?:s([0-9]{1,2})e([0-9]{1,3})|([0-9]{1,2})x([0-9]{1,3}))`)
	releaseMultiPattern   = regexp.MustCompile(`(?i)s[0-9]{1,2}e[0-9]{1,3}e[0-9]{1,3}`)
	releaseSeasonPattern  = regexp.MustCompile(`(?i)(?:^|[ ._\-\[])(?:s([0-9]{1,2})|season[ ._\-]*([0-9]{1,2}))(?:$|[ ._\-\]])`)
	releaseResolution     = regexp.MustCompile(`(?i)(2160p|1080p|720p|576p|480p)`)
	releaseGroup          = regexp.MustCompile(`-([A-Za-z0-9][A-Za-z0-9._]{1,31})$`)
)

func ParseReleaseName(name string) ParsedRelease {
	parsed := ParsedRelease{}
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return parsed
	}

	match := releaseEpisodePattern.FindStringSubmatchIndex(trimmed)
	if match != nil {
		seasonText, episodeText := "", ""
		if match[2] >= 0 && match[3] >= 0 {
			seasonText = trimmed[match[2]:match[3]]
			episodeText = trimmed[match[4]:match[5]]
		} else if match[6] >= 0 && match[7] >= 0 {
			seasonText = trimmed[match[6]:match[7]]
			episodeText = trimmed[match[8]:match[9]]
		}
		parsed.Season, _ = strconv.Atoi(seasonText)
		parsed.Episode, _ = strconv.Atoi(episodeText)
		parsed.Title = normalizeReleaseTitle(trimmed[:match[0]])
		parsed.MultiEpisode = releaseMultiPattern.MatchString(trimmed)
	} else if seasonMatch := releaseSeasonPattern.FindStringSubmatchIndex(trimmed); seasonMatch != nil {
		seasonText := ""
		if seasonMatch[2] >= 0 && seasonMatch[3] >= 0 {
			seasonText = trimmed[seasonMatch[2]:seasonMatch[3]]
		} else if seasonMatch[4] >= 0 && seasonMatch[5] >= 0 {
			seasonText = trimmed[seasonMatch[4]:seasonMatch[5]]
		}
		parsed.Season, _ = strconv.Atoi(seasonText)
		parsed.Title = normalizeReleaseTitle(trimmed[:seasonMatch[0]])
		parsed.SeasonPack = true
	} else {
		parsed.Title = normalizeReleaseTitle(trimmed)
	}

	if resolution := releaseResolution.FindString(trimmed); resolution != "" {
		parsed.Resolution = strings.ToLower(resolution)
	}
	lower := strings.ToLower(trimmed)
	parsed.Source = detectReleaseSource(lower)
	parsed.Codec = detectReleaseCodec(lower)
	if group := releaseGroup.FindStringSubmatch(trimmed); len(group) == 2 {
		parsed.Group = group[1]
	}
	return parsed
}

func normalizeReleaseTitle(value string) string {
	var b strings.Builder
	lastSpace := true
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastSpace = false
			continue
		}
		if !lastSpace {
			b.WriteByte(' ')
			lastSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}

func detectReleaseSource(lower string) string {
	switch {
	case strings.Contains(lower, "web-dl"), strings.Contains(lower, "web.dl"), strings.Contains(lower, "web dl"):
		return "web-dl"
	case strings.Contains(lower, "webrip"), strings.Contains(lower, "web-rip"), strings.Contains(lower, "web.rip"):
		return "webrip"
	case strings.Contains(lower, "bluray"), strings.Contains(lower, "blu-ray"), strings.Contains(lower, "bdrip"):
		return "bluray"
	case strings.Contains(lower, "hdtv"):
		return "hdtv"
	default:
		return ""
	}
}

func detectReleaseCodec(lower string) string {
	switch {
	case strings.Contains(lower, "x265"), strings.Contains(lower, "h265"), strings.Contains(lower, "h.265"), strings.Contains(lower, "hevc"):
		return "h265"
	case strings.Contains(lower, "x264"), strings.Contains(lower, "h264"), strings.Contains(lower, "h.264"), strings.Contains(lower, "avc"):
		return "h264"
	case strings.Contains(lower, "av1"):
		return "av1"
	default:
		return ""
	}
}
