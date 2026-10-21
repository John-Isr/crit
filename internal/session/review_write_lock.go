package session

import (
	"fmt"
	"os"
)

// WithReviewWriteLock serializes read-modify-write transactions on review.json.
// Writers must read their disk state after acquiring it. Keep it separate from
// share.lock, which spans network calls and can contain review-file writes.
func WithReviewWriteLock(identity string, fn func() error) error {
	if err := ensureReviewFolder(identity); err != nil {
		return fmt.Errorf("ensuring review folder: %w", err)
	}
	paths := ReviewPathsFor(identity)
	if err := os.MkdirAll(paths.Folder, 0o700); err != nil {
		return fmt.Errorf("creating review folder: %w", err)
	}
	f, err := os.OpenFile(paths.ReviewLock, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("opening review lock: %w", err)
	}
	defer func() {
		_ = funlock(f)
		_ = f.Close()
	}()
	if err := flockExclusive(f); err != nil {
		return fmt.Errorf("locking review: %w", err)
	}
	return fn()
}
