package backup

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/CarlFlo/tally/internal/database"
)

func readManifest(dir string, seen map[string]bool) (m Manifest, err error) {
	data, e := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if e != nil {
		return m, e
	}
	if e = json.Unmarshal(data, &m); e != nil || m.Format != 1 || m.Schema < 1 || m.Schema > database.Version {
		return m, fmt.Errorf("unsupported backup manifest or schema")
	}
	if _, ok := m.Files["app.db"]; !ok {
		return m, fmt.Errorf("database missing from manifest")
	}
	if len(m.Files)+1 != len(seen) {
		return m, fmt.Errorf("manifest does not cover archive")
	}
	for name, want := range m.Files {
		if !seen[name] || name == "manifest.json" {
			return m, fmt.Errorf("manifest entry missing")
		}
		f, e := os.Open(filepath.Join(dir, filepath.FromSlash(name)))
		if e != nil {
			return m, e
		}
		hash := sha256.New()
		_, e = io.Copy(hash, f)
		f.Close()
		if e != nil || hex.EncodeToString(hash.Sum(nil)) != want {
			return m, fmt.Errorf("backup checksum mismatch")
		}
	}

	return m, nil
}
