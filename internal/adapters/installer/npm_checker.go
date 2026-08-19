package installer

import (
	"encoding/json"
	"fmt"

	"github.com/dirsouza/packages-npm/internal/core/ports/outbound"
)

// npmListOutput mirrors the JSON structure returned by `npm list -g --depth=0 --json`.
type npmListOutput struct {
	Dependencies map[string]npmDep `json:"dependencies"`
}

type npmDep struct {
	Version string `json:"version"`
}

// NpmPackageChecker implements outbound.PackageChecker using the system npm CLI.
type NpmPackageChecker struct{}

func NewNpmPackageChecker() *NpmPackageChecker { return &NpmPackageChecker{} }

// Compile-time assertion.
var _ outbound.PackageChecker = (*NpmPackageChecker)(nil)

// ListGlobalInstalled runs `npm list -g --depth=0 --json` and returns a map of
// package name → InstalledPackage. npm may exit non-zero for peer-dep warnings,
// so we ignore the exit code and parse whatever JSON was written to stdout.
func (c *NpmPackageChecker) ListGlobalInstalled() (map[string]outbound.InstalledPackage, error) {
	cmd, err := newNpmCmd("list", "-g", "--depth=0", "--json")
	if err != nil {
		return nil, err
	}
	stdout, runErr := cmd.Output()

	if len(stdout) == 0 {
		if runErr != nil {
			return nil, fmt.Errorf("npm list falhou: %w", runErr)
		}
		return map[string]outbound.InstalledPackage{}, nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stdout, &raw); err != nil {
		return nil, fmt.Errorf("npm list: saída inválida: %w", err)
	}
	if runErr != nil {
		if _, ok := raw["dependencies"]; !ok {
			return nil, fmt.Errorf("npm list falhou: %w", runErr)
		}
	}

	var result npmListOutput
	if err := json.Unmarshal(stdout, &result); err != nil {
		return nil, fmt.Errorf("npm list: saída inválida: %w", err)
	}

	installed := make(map[string]outbound.InstalledPackage, len(result.Dependencies))
	for name, dep := range result.Dependencies {
		installed[name] = outbound.InstalledPackage{Name: name, Version: dep.Version}
	}
	return installed, nil
}
