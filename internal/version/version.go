package version

import "runtime/debug"

// Version is injected at release time via -ldflags
// "-X sprout/internal/version.Version=<tag>" (set by GoReleaser).
var Version = ""

// String returns the display version, prefixed with "v".
// Priority: ldflags-injected tag → Go module build info → "dev".
func String() string {
	v := Version
	if v == "" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
			v = info.Main.Version
		}
	}
	if v == "" {
		return "dev"
	}
	if v[0] != 'v' {
		v = "v" + v
	}
	return v
}
