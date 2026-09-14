package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func (s *Service) Retain(ctx context.Context) error {
	keep, e := s.retentionCount(ctx)
	if e != nil {
		return e
	}
	rows, e := s.DB.Rows(ctx, "SELECT id,filename FROM backup_records WHERE kind='auto' AND verified=1 ORDER BY created_at DESC,id DESC LIMIT -1 OFFSET ?", keep)
	if e != nil {
		return e
	}
	for _, r := range rows {
		name := r["filename"].(string)
		if filepath.Base(name) != name {
			return fmt.Errorf("unsafe backup filename")
		}
		if e = os.Remove(filepath.Join(s.Path, name)); e != nil && !os.IsNotExist(e) {
			return e
		}
		if _, e = s.DB.ExecContext(ctx, "DELETE FROM backup_records WHERE id=?", r["id"]); e != nil {
			return e
		}
	}
	return nil
}

func ListFiles(path string) ([]string, error) {
	entries, e := os.ReadDir(path)
	if e != nil {
		return nil, e
	}
	out := []string{}
	for _, v := range entries {
		if strings.HasSuffix(v.Name(), ".zip") {
			out = append(out, v.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}
