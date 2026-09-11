// Package update: server-side version-check service that surfaces new
// releases as a persistent notification for the web admin. Off by
// default (no-telemetry policy); enabled with updates.check_enabled.
package update

import (
	"context"
	"fmt"
	"time"

	"github.com/PotenFYR-Studios/FYRwall/internal/version"
)

// Notifier persists an update-available notification through the
// standard notification pipeline (dedup by fingerprint).
type Notifier interface {
	UpsertUpdateNotification(latest, url, notes string) error
}

// Service runs periodic checks.
type Service struct {
	ManifestURL string
	Interval    time.Duration
	Notifier    Notifier
}

// Run blocks until ctx is done, checking on the configured interval.
// Failures are silent-by-design (offline hosts must not spam logs):
// a failed check simply means "no update info this round".
func (s *Service) Run(ctx context.Context) {
	if s.Interval < time.Hour {
		s.Interval = 24 * time.Hour
	}
	t := time.NewTicker(s.Interval)
	defer t.Stop()
	s.once(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.once(ctx)
		}
	}
}

func (s *Service) once(ctx context.Context) {
	res, err := Check(ctx, s.ManifestURL, version.Version)
	if err != nil || !res.UpdateAvailable {
		return
	}
	_ = s.Notifier.UpsertUpdateNotification(res.Latest, res.URL, res.Notes)
}

// String renders the human-facing summary used in the notification.
func UpdateSummary(res Result) string {
	return fmt.Sprintf("FYRwall %s is available (installed: %s). %s", res.Latest, res.Current, res.Notes)
}
