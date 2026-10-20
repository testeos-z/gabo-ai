// Package version reports the build version of the gbai binary.
package version

// DefaultVersion is reported when no version was injected at build time.
const DefaultVersion = "dev"

// Info describes the running build.
type Info struct {
	Version string
}

// UseCase reports build information.
type UseCase struct {
	version string
}

// NewUseCase returns a use case reporting the given version, or DefaultVersion when empty.
func NewUseCase(version string) *UseCase {
	if version == "" {
		version = DefaultVersion
	}
	return &UseCase{version: version}
}

// Show returns the build information.
func (u *UseCase) Show() Info {
	return Info{Version: u.version}
}
