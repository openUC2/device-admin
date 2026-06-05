package main

import (
	"context"
	"log"
	"os"
	"runtime/debug"

	"github.com/carlmjohnson/versioninfo"
	"github.com/urfave/cli/v3"
)

func main() {
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

var cmd = &cli.Command{
	Name:    "machine-admin",
	Version: toolVersion,
	Usage:   "Provides a web browser interface for system administration",
	Commands: []*cli.Command{
		serverCmd,
		sidecarCmd,
	},
}

// Versioning

const (
	// fallbackVersion is the version reported which the Forklift tool reports itself as if its actual
	// version is unknown.
	fallbackVersion = "dev"
)

var (
	toolVersion = determineVersion(buildSummary, fallbackVersion)
	// buildSummary should be overridden by ldflags, such as with GoReleaser's "Summary".
	buildSummary = ""
)

// determineVersion returns either a semver, a pseudoversion, or a Git hash based on information
// available from Go's `debug.ReadBuildInfo()`.
func determineVersion(override, fallback string) string {
	if override != "" {
		return override
	}

	const dirtySuffix = "-dirty"
	// Determine any version tags, if available
	if info, ok := debug.ReadBuildInfo(); ok &&
		info.Main.Version != "" && info.Main.Version != "(devel)" {
		v := info.Main.Version
		if versioninfo.DirtyBuild {
			v += dirtySuffix
		}
		return v
	}
	if v := versioninfo.Version; v != "unknown" && v != "(devel)" {
		if versioninfo.DirtyBuild {
			v += dirtySuffix
		}
		return v
	}

	// Fall back to whatever is available
	if r := versioninfo.Revision; r != "unknown" && r != "" {
		if versioninfo.DirtyBuild {
			r += dirtySuffix
		}
		return r
	}
	return fallback
}
