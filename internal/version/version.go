package version

import "runtime/debug"

// Build information is injected via -ldflags at build time.
var (
	Version   = "0.1.0-dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// String returns the human-readable version line.
func String() string {
	return "fyrwall " + Version + " (" + Commit + ", " + BuildDate + ")"
}

// Info returns detailed build info, falling back to VCS info embedded by go build.
func Info() map[string]string {
	m := map[string]string{
		"version":    Version,
		"commit":     Commit,
		"build_date": BuildDate,
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				if Commit == "unknown" {
					m["commit"] = s.Value
				}
			case "vcs.time":
				if BuildDate == "unknown" {
					m["build_date"] = s.Value
				}
			}
		}
	}
	return m
}
