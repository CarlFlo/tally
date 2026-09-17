package torrent

import (
	"crypto/sha1"
	"encoding/hex"
	"testing"
)

func TestParseTorrentMetadataSingleFile(t *testing.T) {
	info := []byte("d6:lengthi123e4:name8:show.mkv12:piece lengthi16384e6:pieces20:aaaaaaaaaaaaaaaaaaaae")
	data := append([]byte("d4:info"), info...)
	data = append(data, 'e')

	metadata, err := ParseTorrentMetadata(data)
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Name != "show.mkv" || metadata.TotalSize != 123 || len(metadata.Files) != 1 {
		t.Fatalf("unexpected metadata: %+v", metadata)
	}
	if metadata.Files[0].Path != "show.mkv" || metadata.Files[0].Size != 123 {
		t.Fatalf("unexpected file metadata: %+v", metadata.Files[0])
	}
	hash := sha1.Sum(info)
	if metadata.InfoHashV1 != hex.EncodeToString(hash[:]) {
		t.Fatalf("infohash = %q, want %q", metadata.InfoHashV1, hex.EncodeToString(hash[:]))
	}
	inspection := InspectTorrentPayload(metadata)
	if inspection.VideoFiles != 1 || inspection.SubtitleFiles != 0 || inspection.ExecutableFiles != 0 || inspection.MainVideoSize != 123 {
		t.Fatalf("unexpected inspection: %+v", inspection)
	}
}

func TestParseTorrentMetadataMultiFile(t *testing.T) {
	info := []byte("d5:filesld6:lengthi1000e4:pathl13:episode01.mkveed6:lengthi40e4:pathl4:subs11:English.srteee4:name4:Show12:piece lengthi16384e6:pieces20:aaaaaaaaaaaaaaaaaaaae")
	data := append([]byte("d4:info"), info...)
	data = append(data, 'e')

	metadata, err := ParseTorrentMetadata(data)
	if err != nil {
		t.Fatal(err)
	}
	if metadata.TotalSize != 1040 || len(metadata.Files) != 2 {
		t.Fatalf("unexpected metadata: %+v", metadata)
	}
	if metadata.Files[0].Path != "episode01.mkv" || metadata.Files[1].Path != "subs/English.srt" {
		t.Fatalf("unexpected file paths: %+v", metadata.Files)
	}
	inspection := InspectTorrentPayload(metadata)
	if inspection.VideoFiles != 1 || inspection.SubtitleFiles != 1 || inspection.ArchiveFiles != 0 || inspection.ExecutableFiles != 0 {
		t.Fatalf("unexpected inspection: %+v", inspection)
	}
}

func TestParseTorrentMetadataRejectsUnsafePath(t *testing.T) {
	info := []byte("d5:filesld6:lengthi10e4:pathl2:..7:bad.mkveee4:name4:Show12:piece lengthi16384e6:pieces20:aaaaaaaaaaaaaaaaaaaae")
	data := append([]byte("d4:info"), info...)
	data = append(data, 'e')
	if _, err := ParseTorrentMetadata(data); err == nil {
		t.Fatal("unsafe torrent path was accepted")
	}
}

func TestInspectTorrentPayloadFlagsBlockedTypes(t *testing.T) {
	metadata := TorrentMetadata{
		Files: []TorrentFile{
			{Path: "episode.mkv", Size: 1000},
			{Path: "setup.exe", Size: 100},
			{Path: "payload.rar", Size: 200},
		},
		TotalSize: 1300,
	}
	inspection := InspectTorrentPayload(metadata)
	if inspection.VideoFiles != 1 || inspection.ExecutableFiles != 1 || inspection.ArchiveFiles != 1 {
		t.Fatalf("unexpected inspection: %+v", inspection)
	}
}
