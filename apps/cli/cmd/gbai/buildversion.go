package main

import "runtime/debug"

// resolveVersion returns the ldflags version when set, else the module
// version Go records for `go install <module>@v<x.y.z>` builds; source builds
// ("(devel)") and binaries without build info resolve to "".
func resolveVersion(build string, read func() (*debug.BuildInfo, bool)) string {
	if build != "" {
		return build
	}
	if info, ok := read(); ok && info != nil && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return ""
}
