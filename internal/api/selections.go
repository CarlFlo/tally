package api

import (
	"sync"
	"time"
)

type selection struct {
	Profile string
	Data    []byte
	Expires time.Time
}

type selectionStore struct{ sync.Map }

func (s *selectionStore) prune() {
	count := 0
	s.Range(func(k, v any) bool {
		count++
		if time.Now().After(v.(selection).Expires) || count > 10000 {
			s.Delete(k)
		}
		return true
	})
}
