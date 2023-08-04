package trpc

import "fmt"

// rule of trpc version
// 1. MAJOR version when you make incompatible API changes；
// 2. MINOR version when you add functionality in a backwards-compatible manner；
// 3. PATCH version when you make backwards-compatible bug fixes；
// 4. Additional labels for pre-release and build metadata are available as extensions to the MAJOR.MINOR.PATCH format；
// alpha             0.1.0-alpha
// beta              0.1.0-beta
// release candidate 0.1.0-rc
// release           0.1.0
const (
	MajorVersion  = 0
	MinorVersion  = 14
	PatchVersion  = 0
	VersionSuffix = "-dev" // -alpha -alpha.1 -beta -rc -rc.1
)

// Version returns the version of trpc.
func Version() string {
	return fmt.Sprintf("v%d.%d.%d%s", MajorVersion, MinorVersion, PatchVersion, VersionSuffix)
}
