package version

import (
	"fmt"
	"runtime/debug"
	"time"
)

// These variables can be set at build time via ldflags:
//
//	go build -ldflags="-X github.com/muurk/smartap/internal/version.Version=v1.2.3 \
//	                   -X github.com/muurk/smartap/internal/version.Commit=abc123"
//
// If not set, they will be populated from git info at runtime (if available),
// or fall back to "dev" with a timestamp.
var (
	// Version is the semantic version of the application
	Version = ""
	// Commit is the git commit hash
	Commit = ""
)

func init() {
	// If version wasn't set via ldflags, try to get it from build info
	if Version == "" || Commit == "" {
		populateFromBuildInfo()
	}

	// Final fallback if we still don't have values
	if Version == "" {
		Version = fmt.Sprintf("dev-%s", time.Now().Format("20060102-150405"))
	}
	if Commit == "" {
		Commit = "unknown"
	}
}

// populateFromBuildInfo attempts to read version info from Go's build info
// This includes VCS information when built from a git repository
func populateFromBuildInfo() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	// Look for VCS settings in build info
	var vcsRevision, vcsModified, vcsTime string
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			vcsRevision = setting.Value
		case "vcs.modified":
			vcsModified = setting.Value
		case "vcs.time":
			vcsTime = setting.Value
		}
	}

	// Set commit from VCS revision if we have it
	if Commit == "" && vcsRevision != "" {
		// Use short hash (first 7 characters)
		if len(vcsRevision) > 7 {
			Commit = vcsRevision[:7]
		} else {
			Commit = vcsRevision
		}
		// Mark as dirty if modified
		if vcsModified == "true" {
			Commit += "-dirty"
		}
	}

	// For version, we don't have git tags in build info, so use a dev version
	// with the commit time if available
	if Version == "" {
		if vcsTime != "" {
			// Parse and format the VCS time
			if t, err := time.Parse(time.RFC3339, vcsTime); err == nil {
				Version = fmt.Sprintf("dev-%s", t.Format("20060102"))
			}
		}
	}
}

// Full returns the full version string including commit
func Full() string {
	return fmt.Sprintf("%s (commit: %s)", Version, Commit)
}
