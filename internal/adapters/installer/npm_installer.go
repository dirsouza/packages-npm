package installer

import (
	"fmt"
	"strings"

	"github.com/dirsouza/packages-npm/internal/core/domain"
	"github.com/dirsouza/packages-npm/internal/core/ports/outbound"
)

const maxErrorLines = 3

// NpmInstaller implements outbound.PackageInstaller using the system npm CLI.
type NpmInstaller struct{}

func NewNpmInstaller() *NpmInstaller { return &NpmInstaller{} }

var _ outbound.PackageInstaller = (*NpmInstaller)(nil)

// Install runs `npm install -g <pkgTarget>`.
func (n *NpmInstaller) Install(pkgTarget string) error {
	return runGlobal(domain.ErrInstallFailed, "install", pkgTarget)
}

// Uninstall runs `npm uninstall -g <pkgName>`; pkgName must not carry a version.
func (n *NpmInstaller) Uninstall(pkgName string) error {
	return runGlobal(domain.ErrUninstallFailed, "uninstall", pkgName)
}

func runGlobal(failure error, command, arg string) error {
	cmd, err := newNpmCmd(command, "-g", arg)
	if err != nil {
		return fmt.Errorf("%w: %s — %w", failure, arg, err)
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s — %s", failure, arg, npmFailureDetail(output))
	}
	reshimAsdf(cmd.Path)
	return nil
}

func npmFailureDetail(output []byte) string {
	detail := npmErrorSummary(output)
	if hint := corepackHint(output); hint != "" {
		detail += "\n\n" + hint
	}
	return detail
}

// cutNpmErrorPrefix strips the error prefix of an npm log line: "npm error"
// in current npm, "npm ERR!" in older releases.
func cutNpmErrorPrefix(line string) (string, bool) {
	for _, prefix := range [...]string{"npm error", "npm ERR!"} {
		if rest, ok := strings.CutPrefix(line, prefix); ok {
			return strings.TrimSpace(rest), true
		}
	}
	return "", false
}

// npmErrorSummary keeps the first npm error lines, which carry the cause of
// the failure, instead of the whole npm log.
func npmErrorSummary(output []byte) string {
	var lines []string
	for line := range strings.Lines(string(output)) {
		if cause, ok := cutNpmErrorPrefix(line); ok {
			lines = append(lines, cause)
			if len(lines) == maxErrorLines {
				break
			}
		}
	}
	if len(lines) == 0 {
		return strings.TrimSpace(string(output))
	}
	return strings.Join(lines, "; ")
}
