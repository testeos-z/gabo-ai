package main

import "runtime/debug"

// resolveVersion returns the ldflags version when set, else the module
// version Go records in the build info: v<x.y.z> for `go install
// <module>@v<x.y.z>`, a pseudo-version for a source build inside a git
// checkout (init rejects it as not x.y.z). "(devel)" and missing build info
// resolve to "".
func resolveVersion(build string, read func() (*debug.BuildInfo, bool)) string {
	if build != "" {
		return build
	}
	if info, ok := read(); ok && info != nil && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return ""
}
