package database

import (
	"context"
)

func (s *Store) Rows(ctx context.Context, query string, args ...any) ([]map[string]any, error) {
	rows, e := s.QueryContext(ctx, query, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	cols, e := rows.Columns()
	if e != nil {
		return nil, e
	}
	out := make([]map[string]any, 0)
	for rows.Next() {
		v := make([]any, len(cols))
		ptr := make([]any, len(cols))
		for i := range v {
			ptr[i] = &v[i]
		}
		if e = rows.Scan(ptr...); e != nil {
			return nil, e
		}
		r := map[string]any{}
		for i, k := range cols {
			if b, ok := v[i].([]byte); ok {
				r[k] = string(b)
			} else {
				r[k] = v[i]
			}
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
