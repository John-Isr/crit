package session

import (
	"fmt"
	"os"
	"path/filepath"
)

// WithReviewWriteLock serializes read-modify-write transactions on review.json.
// Writers must read their disk state after acquiring it. Keep it separate from
// share.lock, which spans network calls and can contain review-file writes.
func WithReviewWriteLock(identity string, fn func() error) error {
	if err := ensureReviewFolder(identity); err != nil {
		return err
	}
	if err := os.MkdirAll(identity, 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(identity, "review.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("opening review lock: %w", err)
	}
	defer f.Close()
	if err := flockExclusive(f); err != nil {
		return fmt.Errorf("locking review: %w", err)
	}
	defer funlock(f)
	return fn()
}
