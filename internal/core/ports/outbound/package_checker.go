package outbound

// InstalledPackage carries the globally installed npm package name and version.
type InstalledPackage struct {
	Name    string
	Version string
}

// PackageChecker is the outbound port for querying globally installed npm packages.
// Implementations must not block the caller for longer than needed.
type PackageChecker interface {
	ListGlobalInstalled() (map[string]InstalledPackage, error)
}
