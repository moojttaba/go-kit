// Package buildinfo exposes the running binary's version/build metadata and a
// helper to log it once at startup, so every service reports "what am I" in a
// uniform structured line. Shared here (not copied per service) per RULES §4.
//
// Version resolution, in order:
//  1. the linker-injected `version` var (CI/Docker sets a real semver), else
//  2. the VCS revision Go embeds automatically on `go build` (short commit SHA,
//     plus "-dirty" when the tree had uncommitted changes), else
//  3. "dev".
//
// To inject a version at build time:
//
//	go build -ldflags "-X github.com/moojttaba/go-kit/buildinfo.version=v1.2.3"
package buildinfo

import (
	"log/slog"
	"runtime/debug"
)

// version is intentionally unexported and settable only via -ldflags.
var version = ""

// Info is resolved build metadata for the running binary.
type Info struct {
	Version  string // semver (ldflags) or short VCS revision or "dev"
	Revision string // full VCS commit SHA, "" if unavailable
	Time     string // VCS commit time (RFC 3339), "" if unavailable
	Dirty    bool   // build had uncommitted changes
	Go       string // Go toolchain version
}

// Read resolves build metadata from ldflags and the embedded build info.
func Read() Info {
	info := Info{Version: version}

	if bi, ok := debug.ReadBuildInfo(); ok {
		info.Go = bi.GoVersion
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				info.Revision = s.Value
			case "vcs.time":
				info.Time = s.Value
			case "vcs.modified":
				info.Dirty = s.Value == "true"
			}
		}
	}

	if info.Version == "" {
		switch {
		case info.Revision != "":
			short := info.Revision
			if len(short) > 12 {
				short = short[:12]
			}
			info.Version = short
			if info.Dirty {
				info.Version += "-dirty"
			}
		default:
			info.Version = "dev"
		}
	}

	return info
}

// Log emits a single structured "build info" line for the service and returns
// the resolved Info (so callers can reuse it, e.g. in a health payload).
func Log(service string) Info {
	i := Read()
	slog.Info("build info",
		"service", service,
		"version", i.Version,
		"revision", i.Revision,
		"build_time", i.Time,
		"dirty", i.Dirty,
		"go", i.Go,
	)
	return i
}
