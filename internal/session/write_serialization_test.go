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

// An empty flush with no review folder must still clear pendingWrite, or the
// watcher ignores review.json written by another process (crit comment).
func TestEmptyWriteWithoutFolderClearsPendingWrite(t *testing.T) {
	s := &Session{RepoRoot: t.TempDir()}
	comment := s.AddReviewComment("gone", "", "")
	s.DeleteReviewComment(comment.ID)
	s.WriteFiles()
	quiesceSession(t, s)
	if s.PendingWriteForTest() {
		t.Fatal("pendingWrite still set after empty write")
	}
	critPath := s.critJSONPath()
	if _, err := os.Stat(critPath); !os.IsNotExist(err) {
		t.Fatalf("empty write created the review folder: %v", err)
	}
	external := CritJSON{ReviewComments: []Comment{{ID: "r_ext", Body: "external"}}}
	if err := SaveCritJSON(critPath, external); err != nil {
		t.Fatal(err)
	}
	if !s.MergeExternalCritJSONForTest() {
		t.Fatal("external review comment was not merged")
	}
}

// A write after the review folder was deleted externally clears the session
// without recreating the folder for the lock file.
func TestWriteAfterExternalDeleteKeepsFolderDeleted(t *testing.T) {
	s := &Session{RepoRoot: t.TempDir()}
	s.AddReviewComment("x", "", "")
	s.WriteFiles()
	critPath := s.critJSONPath()
	if err := os.RemoveAll(critPath); err != nil {
		t.Fatal(err)
	}
	s.AddReviewComment("y", "", "")
	s.WriteFiles()
	quiesceSession(t, s)
	if _, err := os.Stat(critPath); !os.IsNotExist(err) {
		t.Fatalf("review folder recreated after external delete: %v", err)
	}
}
