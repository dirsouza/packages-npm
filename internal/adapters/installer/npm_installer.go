package installer

import (
	"fmt"

	"github.com/dirsouza/packages-npm/internal/core/domain"
	"github.com/dirsouza/packages-npm/internal/core/ports/outbound"
)

// NpmInstaller implements outbound.PackageInstaller using the system npm CLI.
type NpmInstaller struct{}

func NewNpmInstaller() *NpmInstaller { return &NpmInstaller{} }

// Compile-time assertion.
var _ outbound.PackageInstaller = (*NpmInstaller)(nil)

// Install runs `npm install -g <pkgTarget>` and returns a descriptive error on failure.
func (n *NpmInstaller) Install(pkgTarget string) error {
	cmd, err := newNpmCmd("install", "-g", pkgTarget)
	if err != nil {
		return fmt.Errorf("%w: %s — %s", domain.ErrInstallFailed, pkgTarget, err.Error())
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s — %s", domain.ErrInstallFailed, pkgTarget, string(output))
	}
	return nil
}

// Uninstall runs `npm uninstall -g <pkgName>` and returns a descriptive error on failure.
// Note: pkgName should be the bare package name (e.g. "typescript"), without version.
func (n *NpmInstaller) Uninstall(pkgName string) error {
	cmd, err := newNpmCmd("uninstall", "-g", pkgName)
	if err != nil {
		return fmt.Errorf("%w: %s — %s", domain.ErrUninstallFailed, pkgName, err.Error())
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s — %s", domain.ErrUninstallFailed, pkgName, string(output))
	}
	return nil
}
