package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestDownloadToTempFileStreamsAndEnforcesLimit(t *testing.T) {
	c := control(t)
	payload := strings.Repeat("x", 1024*1024)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte(payload))
	}))
	defer server.Close()

	dir := t.TempDir()
	response, err := c.DownloadToTempFile(context.Background(), Request{
		Provider: "download",
		URL:      server.URL,
		MaxBytes: 2 << 20,
	}, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(response.Path)
	info, err := os.Stat(response.Path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != int64(len(payload)) || response.Size != int64(len(payload)) || response.Status != http.StatusOK {
		t.Fatalf("unexpected streamed response: size=%d reported=%d status=%d", info.Size(), response.Size, response.Status)
	}

	overflowDir := t.TempDir()
	if _, err = c.DownloadToTempFile(context.Background(), Request{
		Provider: "download-limit",
		URL:      server.URL,
		MaxBytes: 1024,
	}, overflowDir); err == nil {
		t.Fatal("oversized provider download was accepted")
	}
	entries, err := os.ReadDir(overflowDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatal("failed provider download left a temporary file behind")
	}
}
