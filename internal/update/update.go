// Package update implements the optional new-version check. Per the
// no-telemetry policy (spec section 93) it is disabled by default and
// only contacts the single configured manifest URL.
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Manifest is the release index format served by the project (or a local
// file in air-gapped setups).
type Manifest struct {
	Latest string `json:"latest"`
	URL    string `json:"url"`
	Notes  string `json:"notes"`
}

// Result of a check.
type Result struct {
	UpdateAvailable bool
	Current         string
	Latest          string
	URL             string
	Notes           string
}

// HTTPTimeout bounds the manifest fetch.
const HTTPTimeout = 10 * time.Second

// Check fetches the manifest and compares against current.
func Check(ctx context.Context, manifestURL, current string) (Result, error) {
	res := Result{Current: current}
	ctx2, cancel := context.WithTimeout(ctx, HTTPTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx2, http.MethodGet, manifestURL, nil)
	if err != nil {
		return res, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return res, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return res, fmt.Errorf("manifest fetch: HTTP %d", resp.StatusCode)
	}
	// Cap read at 64 KiB; a manifest is tiny.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return res, err
	}
	var m Manifest
	if err := json.Unmarshal(body, &m); err != nil {
		return res, err
	}
	res.Latest = m.Latest
	res.URL = m.URL
	res.Notes = m.Notes
	res.UpdateAvailable = CompareVersions(current, m.Latest) < 0
	return res, nil
}

// CompareVersions compares dotted numeric versions (-1, 0, 1). A
// pre-release suffix (e.g. -dev) sorts older than the bare number;
// missing segments are equivalent to zero (1.0 == 1.0.0).
func CompareVersions(a, b string) int {
	aNum, aPre := parseVer(a)
	bNum, bPre := parseVer(b)
	for i := 0; i < 3; i++ {
		if aNum[i] != bNum[i] {
			if aNum[i] < bNum[i] {
				return -1
			}
			return 1
		}
	}
	switch {
	case aPre && !bPre:
		return -1
	case bPre && !aPre:
		return 1
	}
	return 0
}

// parseVer splits a version into three numeric segments plus a
// pre-release flag.
func parseVer(v string) ([3]int, bool) {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	pre := false
	if i := strings.IndexByte(v, '-'); i >= 0 {
		v = v[:i]
		pre = true
	}
	var out [3]int
	for i, part := range strings.SplitN(v, ".", 3) {
		out[i], _ = strconv.Atoi(part)
	}
	return out, pre
}
