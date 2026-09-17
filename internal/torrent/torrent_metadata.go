package torrent

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"path"
	"sort"
	"strconv"
	"strings"
)

const (
	maxBencodeDepth = 64
	maxTorrentFiles = 10000
)

type TorrentMetadata struct {
	Name       string        `json:"name"`
	InfoHashV1 string        `json:"info_hash_v1,omitempty"`
	InfoHashV2 string        `json:"info_hash_v2,omitempty"`
	Files      []TorrentFile `json:"files"`
	TotalSize  int64         `json:"total_size"`
}

type bencodeNode struct {
	kind    byte
	integer int64
	bytes   []byte
	list    []bencodeNode
	dict    map[string]bencodeNode
	start   int
	end     int
}

type bencodeDecoder struct {
	data []byte
	pos  int
}

func ParseTorrentMetadata(data []byte) (TorrentMetadata, error) {
	if len(data) < 2 || data[0] != 'd' {
		return TorrentMetadata{}, fmt.Errorf("torrent metadata is not a bencoded dictionary")
	}
	decoder := bencodeDecoder{data: data}
	root, err := decoder.value(0)
	if err != nil {
		return TorrentMetadata{}, err
	}
	if decoder.pos != len(data) {
		return TorrentMetadata{}, fmt.Errorf("torrent metadata has trailing data")
	}
	if root.kind != 'd' {
		return TorrentMetadata{}, fmt.Errorf("torrent metadata root is not a dictionary")
	}
	info, ok := root.dict["info"]
	if !ok || info.kind != 'd' || info.start < 0 || info.end > len(data) || info.start >= info.end {
		return TorrentMetadata{}, fmt.Errorf("torrent metadata is missing the info dictionary")
	}

	metadata := TorrentMetadata{Name: torrentString(info.dict, "name.utf-8", "name")}
	if metadata.Name == "" {
		return TorrentMetadata{}, fmt.Errorf("torrent metadata is missing a name")
	}

	rawInfo := data[info.start:info.end]
	if _, ok := info.dict["pieces"]; ok {
		hash := sha1.Sum(rawInfo)
		metadata.InfoHashV1 = hex.EncodeToString(hash[:])
	}
	if version, ok := info.dict["meta version"]; ok && version.kind == 'i' && version.integer == 2 {
		hash := sha256.Sum256(rawInfo)
		metadata.InfoHashV2 = hex.EncodeToString(hash[:])
	}

	if files, ok := info.dict["files"]; ok {
		if err := collectV1Files(files, &metadata); err != nil {
			return TorrentMetadata{}, err
		}
	} else if tree, ok := info.dict["file tree"]; ok {
		if err := collectV2Files(tree, nil, &metadata); err != nil {
			return TorrentMetadata{}, err
		}
	} else {
		length, ok := info.dict["length"]
		if !ok || length.kind != 'i' || length.integer < 0 {
			return TorrentMetadata{}, fmt.Errorf("torrent metadata has no valid file length")
		}
		if err := appendTorrentFile(&metadata, metadata.Name, length.integer); err != nil {
			return TorrentMetadata{}, err
		}
	}

	if len(metadata.Files) == 0 {
		return TorrentMetadata{}, fmt.Errorf("torrent metadata contains no files")
	}
	return metadata, nil
}

func collectV1Files(node bencodeNode, metadata *TorrentMetadata) error {
	if node.kind != 'l' {
		return fmt.Errorf("torrent files metadata is not a list")
	}
	for _, file := range node.list {
		if file.kind != 'd' {
			return fmt.Errorf("torrent file entry is not a dictionary")
		}
		length, ok := file.dict["length"]
		if !ok || length.kind != 'i' || length.integer < 0 {
			return fmt.Errorf("torrent file entry has no valid length")
		}
		pathNode, ok := file.dict["path.utf-8"]
		if !ok {
			pathNode, ok = file.dict["path"]
		}
		if !ok || pathNode.kind != 'l' || len(pathNode.list) == 0 {
			return fmt.Errorf("torrent file entry has no valid path")
		}
		segments := make([]string, 0, len(pathNode.list))
		for _, segment := range pathNode.list {
			if segment.kind != 's' {
				return fmt.Errorf("torrent file path contains a non-string segment")
			}
			value := string(segment.bytes)
			if !validTorrentPathSegment(value) {
				return fmt.Errorf("torrent file path contains an unsafe segment")
			}
			segments = append(segments, value)
		}
		if err := appendTorrentFile(metadata, strings.Join(segments, "/"), length.integer); err != nil {
			return err
		}
	}
	return nil
}

func collectV2Files(node bencodeNode, prefix []string, metadata *TorrentMetadata) error {
	if node.kind != 'd' {
		return fmt.Errorf("torrent v2 file tree entry is not a dictionary")
	}
	keys := make([]string, 0, len(node.dict))
	for key := range node.dict {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		child := node.dict[key]
		if key == "" {
			if len(prefix) == 0 || child.kind != 'd' {
				return fmt.Errorf("torrent v2 file tree has an invalid leaf")
			}
			length, ok := child.dict["length"]
			if !ok || length.kind != 'i' || length.integer < 0 {
				return fmt.Errorf("torrent v2 file entry has no valid length")
			}
			if err := appendTorrentFile(metadata, strings.Join(prefix, "/"), length.integer); err != nil {
				return err
			}
			continue
		}
		if !validTorrentPathSegment(key) {
			return fmt.Errorf("torrent v2 file path contains an unsafe segment")
		}
		next := append(append([]string(nil), prefix...), key)
		if err := collectV2Files(child, next, metadata); err != nil {
			return err
		}
	}
	return nil
}

func appendTorrentFile(metadata *TorrentMetadata, filePath string, size int64) error {
	if len(metadata.Files) >= maxTorrentFiles {
		return fmt.Errorf("torrent metadata contains too many files")
	}
	if filePath == "" || size < 0 || metadata.TotalSize > math.MaxInt64-size {
		return fmt.Errorf("torrent file metadata is invalid")
	}
	metadata.Files = append(metadata.Files, TorrentFile{Path: filePath, Size: size})
	metadata.TotalSize += size
	return nil
}

func validTorrentPathSegment(value string) bool {
	return value != "" && value != "." && value != ".." && !strings.ContainsAny(value, "/\\\x00")
}

func torrentString(dict map[string]bencodeNode, keys ...string) string {
	for _, key := range keys {
		if node, ok := dict[key]; ok && node.kind == 's' && len(node.bytes) > 0 {
			return string(node.bytes)
		}
	}
	return ""
}

func InspectTorrentPayload(metadata TorrentMetadata) PayloadInspection {
	inspection := PayloadInspection{Files: append([]TorrentFile(nil), metadata.Files...), TotalSize: metadata.TotalSize}
	for _, file := range metadata.Files {
		ext := strings.ToLower(path.Ext(file.Path))
		switch {
		case isVideoExtension(ext):
			inspection.VideoFiles++
			if file.Size > inspection.MainVideoSize {
				inspection.MainVideoSize = file.Size
			}
		case isSubtitleExtension(ext):
			inspection.SubtitleFiles++
		case isArchiveExtension(ext):
			inspection.ArchiveFiles++
		case isExecutableExtension(ext):
			inspection.ExecutableFiles++
		}
	}
	return inspection
}

func isVideoExtension(ext string) bool {
	switch ext {
	case ".mkv", ".mp4", ".m4v", ".avi", ".mov", ".ts", ".m2ts", ".webm":
		return true
	default:
		return false
	}
}

func isSubtitleExtension(ext string) bool {
	switch ext {
	case ".srt", ".ass", ".ssa", ".sub", ".vtt":
		return true
	default:
		return false
	}
}

func isArchiveExtension(ext string) bool {
	switch ext {
	case ".rar", ".zip", ".7z", ".tar", ".gz", ".bz2", ".xz":
		return true
	default:
		return false
	}
}

func isExecutableExtension(ext string) bool {
	switch ext {
	case ".exe", ".msi", ".com", ".scr", ".bat", ".cmd", ".ps1", ".vbs", ".js", ".jar", ".apk", ".dmg", ".pkg", ".deb", ".rpm", ".appimage", ".sh":
		return true
	default:
		return false
	}
}

func (d *bencodeDecoder) value(depth int) (bencodeNode, error) {
	if depth > maxBencodeDepth || d.pos >= len(d.data) {
		return bencodeNode{}, fmt.Errorf("invalid bencoded torrent metadata")
	}
	start := d.pos
	switch d.data[d.pos] {
	case 'i':
		d.pos++
		end := d.pos
		for end < len(d.data) && d.data[end] != 'e' {
			end++
		}
		if end == len(d.data) || end == d.pos {
			return bencodeNode{}, fmt.Errorf("invalid bencoded integer")
		}
		value, err := strconv.ParseInt(string(d.data[d.pos:end]), 10, 64)
		if err != nil {
			return bencodeNode{}, fmt.Errorf("invalid bencoded integer")
		}
		d.pos = end + 1
		return bencodeNode{kind: 'i', integer: value, start: start, end: d.pos}, nil
	case 'l':
		d.pos++
		node := bencodeNode{kind: 'l', start: start}
		for {
			if d.pos >= len(d.data) {
				return bencodeNode{}, fmt.Errorf("unterminated bencoded list")
			}
			if d.data[d.pos] == 'e' {
				d.pos++
				node.end = d.pos
				return node, nil
			}
			child, err := d.value(depth + 1)
			if err != nil {
				return bencodeNode{}, err
			}
			node.list = append(node.list, child)
		}
	case 'd':
		d.pos++
		node := bencodeNode{kind: 'd', dict: map[string]bencodeNode{}, start: start}
		for {
			if d.pos >= len(d.data) {
				return bencodeNode{}, fmt.Errorf("unterminated bencoded dictionary")
			}
			if d.data[d.pos] == 'e' {
				d.pos++
				node.end = d.pos
				return node, nil
			}
			key, err := d.stringNode()
			if err != nil {
				return bencodeNode{}, err
			}
			name := string(key.bytes)
			if _, exists := node.dict[name]; exists {
				return bencodeNode{}, fmt.Errorf("duplicate bencoded dictionary key")
			}
			value, err := d.value(depth + 1)
			if err != nil {
				return bencodeNode{}, err
			}
			node.dict[name] = value
		}
	default:
		return d.stringNode()
	}
}

func (d *bencodeDecoder) stringNode() (bencodeNode, error) {
	start := d.pos
	colon := d.pos
	for colon < len(d.data) && d.data[colon] != ':' {
		if d.data[colon] < '0' || d.data[colon] > '9' {
			return bencodeNode{}, fmt.Errorf("invalid bencoded string length")
		}
		colon++
	}
	if colon == len(d.data) || colon == d.pos {
		return bencodeNode{}, fmt.Errorf("invalid bencoded string")
	}
	length, err := strconv.ParseInt(string(d.data[d.pos:colon]), 10, 64)
	if err != nil || length < 0 || length > int64(len(d.data)) {
		return bencodeNode{}, fmt.Errorf("invalid bencoded string length")
	}
	valueStart := colon + 1
	if length > int64(len(d.data)-valueStart) {
		return bencodeNode{}, fmt.Errorf("bencoded string exceeds torrent metadata")
	}
	d.pos = valueStart + int(length)
	return bencodeNode{kind: 's', bytes: d.data[valueStart:d.pos], start: start, end: d.pos}, nil
}
