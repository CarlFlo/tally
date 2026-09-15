package api

import (
	"testing"
	"time"
)

func TestSelectionsBelongToOneServerAndExpire(t *testing.T) {
	first, second := &Server{}, &Server{}
	first.selections.Store("choice", selection{Profile: "profile-admin", Expires: time.Now().Add(time.Minute)})
	if _, ok := second.selections.Load("choice"); ok {
		t.Fatal("a different server inherited a torrent selection")
	}
	first.selections.Store("expired", selection{Profile: "profile-admin", Expires: time.Now().Add(-time.Minute)})
	first.selections.prune()
	if _, ok := first.selections.Load("expired"); ok {
		t.Fatal("expired selection survived pruning")
	}
	if _, ok := first.selections.Load("choice"); !ok {
		t.Fatal("pruning removed a current selection")
	}
}
