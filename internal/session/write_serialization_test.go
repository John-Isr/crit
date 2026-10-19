package session

import (
	"os"
	"testing"
	"time"
)

func TestWriteFilesWaitsForInFlightSave(t *testing.T) {
	s := &Session{RepoRoot: t.TempDir()}
	comment := s.AddReviewComment("delete me", "", "")
	s.WriteFiles()
	// Model a background save holding writeMu. A direct flush must wait for it.
	s.writeMu.Lock()
	done := make(chan struct{})
	go func() { s.WriteFiles(); close(done) }()
	premature := false
	select {
	case <-done:
		premature = true
	case <-time.After(100 * time.Millisecond):
	}
	s.DeleteReviewComment(comment.ID)
	s.writeMu.Unlock()
	<-done
	quiesceSession(t, s)
	if premature {
		t.Fatal("WriteFiles bypassed the in-flight save")
	}
	if _, err := os.Stat(ReviewPathsFor(s.critJSONPath()).Review); !os.IsNotExist(err) {
		t.Fatalf("deleted comment's review file remains: %v", err)
	}
}
