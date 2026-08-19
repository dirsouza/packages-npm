package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// npmLocation holds the resolved npm binary path and its parent bin directory.
// The binDir is injected into PATH so that `node` (called internally by npm)
// is also accessible — critical for GUI apps that do not inherit the shell PATH.
type npmLocation struct {
	path   string // absolute path to npm executable
	binDir string // parent directory (contains node, npx, etc.)
}

// resolveNpm locates the npm executable and its bin directory.
//
// GUI apps do not inherit the shell PATH on macOS, Windows, and some Linux
// desktop environments. This function tries common install locations per OS.
func resolveNpm() (npmLocation, error) {
	found := func(p string) (npmLocation, error) {
		return npmLocation{path: p, binDir: filepath.Dir(p)}, nil
	}

	// 1. Process PATH — works in terminal launches and most Linux setups.
	if p, err := exec.LookPath("npm"); err == nil {
		return found(p)
	}

	if runtime.GOOS == "windows" {
		return resolveNpmWindows(found)
	}
	return resolveNpmUnix(found)
}

// resolveNpmUnix searches macOS and Linux specific locations.
func resolveNpmUnix(found func(string) (npmLocation, error)) (npmLocation, error) {
	fixed := []string{
		"/usr/local/bin/npm",    // Homebrew (Intel Mac) / Linux npm
		"/opt/homebrew/bin/npm", // Homebrew (Apple Silicon)
		"/usr/bin/npm",          // Linux system package
		"/usr/local/n/bin/npm",  // n (node version manager)
	}
	for _, p := range fixed {
		if isExecutable(p) {
			return found(p)
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return npmLocation{}, errNotFound()
	}

	// Volta
	if p := filepath.Join(home, ".volta", "bin", "npm"); isExecutable(p) {
		return found(p)
	}

	// fnm — newest version installed
	if p, err := globLast(filepath.Join(home, ".local/share/fnm/node-versions/*/installation/bin/npm")); err == nil {
		return found(p)
	}

	// nvm — resolve default alias first for exact version match
	nvmAlias := filepath.Join(home, ".nvm", "alias", "default")
	if data, err := os.ReadFile(nvmAlias); err == nil {
		ver := strings.TrimSpace(string(data))
		if !strings.HasPrefix(ver, "v") {
			ver = "v" + ver
		}
		p := filepath.Join(home, ".nvm", "versions", "node", ver, "bin", "npm")
		if isExecutable(p) {
			return found(p)
		}
	}

	// nvm — fallback: lexicographically last (newest) installed version
	if p, err := globLast(filepath.Join(home, ".nvm/versions/node/*/bin/npm")); err == nil {
		return found(p)
	}

	return npmLocation{}, errNotFound()
}

// resolveNpmWindows searches Windows-specific npm locations.
func resolveNpmWindows(found func(string) (npmLocation, error)) (npmLocation, error) {
	// Fixed paths from common Windows installers (npm.cmd is the wrapper script).
	fixed := []string{
		`C:\Program Files\nodejs\npm.cmd`,
		`C:\Program Files (x86)\nodejs\npm.cmd`,
	}
	for _, p := range fixed {
		if isExecutable(p) {
			return found(p)
		}
	}

	home, _ := os.UserHomeDir()
	appData := os.Getenv("APPDATA")
	localAppData := os.Getenv("LOCALAPPDATA")

	// Volta (~\AppData\Roaming\Volta\bin\npm.cmd or ~\.volta\bin\npm.cmd)
	for _, p := range []string{
		filepath.Join(home, ".volta", "bin", "npm.cmd"),
		filepath.Join(appData, "Volta", "bin", "npm.cmd"),
	} {
		if isExecutable(p) {
			return found(p)
		}
	}

	// nvm-windows (~\AppData\Roaming\nvm\<version>\npm.cmd)
	if p, err := globLast(filepath.Join(appData, `nvm\v*\npm.cmd`)); err == nil {
		return found(p)
	}

	// fnm on Windows (~\AppData\Local\fnm\node-versions\*\installation\npm.cmd)
	if p, err := globLast(filepath.Join(localAppData, `fnm\node-versions\*\installation\npm.cmd`)); err == nil {
		return found(p)
	}

	// Scoop (~\scoop\apps\nodejs\current\npm.cmd)
	if p := filepath.Join(home, `scoop\apps\nodejs\current\npm.cmd`); isExecutable(p) {
		return found(p)
	}

	// Chocolatey
	if p := `C:\ProgramData\chocolatey\bin\npm.cmd`; isExecutable(p) {
		return found(p)
	}

	return npmLocation{}, errNotFound()
}

// newNpmCmd creates an exec.Cmd for npm with the resolved binary and the npm
// bin directory prepended to PATH, ensuring `node` is also accessible.
func newNpmCmd(args ...string) (*exec.Cmd, error) {
	loc, err := resolveNpm()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(loc.path, args...)

	// Prepend the node bin dir so npm can invoke `node` even in GUI contexts.
	// Use the OS-specific path list separator (: on Unix, ; on Windows).
	current := os.Getenv("PATH")
	if current == "" {
		current = "/usr/bin:/bin"
	}
	sep := string(os.PathListSeparator)
	cmd.Env = append(os.Environ(), "PATH="+loc.binDir+sep+current)

	return cmd, nil
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		ext := strings.ToLower(filepath.Ext(path))
		return ext == ".exe" || ext == ".cmd" || ext == ".bat"
	}
	return info.Mode()&0o111 != 0
}

// globLast returns the lexicographically last match for pattern (newest
// semver-named directory when sorted by name).
func globLast(pattern string) (string, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return "", fmt.Errorf("nenhum match para %s", pattern)
	}
	return matches[len(matches)-1], nil
}

func errNotFound() error {
	return fmt.Errorf("npm não encontrado; instale Node.js: https://nodejs.org")
}
