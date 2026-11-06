package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
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
		resolvedPath, err := absolutePath(p)
		if err != nil {
			return npmLocation{}, err
		}
		return npmLocation{path: resolvedPath, binDir: filepath.Dir(resolvedPath)}, nil
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

	// asdf
	if p, err := resolveAsdfNpm(home); err == nil {
		return found(p)
	}

	// fnm — newest version installed (.../node-versions/<version>/installation/bin/npm)
	if p, err := globNewest(filepath.Join(home, ".local", "share", "fnm", "node-versions", "*", "installation", "bin", "npm"), 3); err == nil {
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

	// nvm — fallback: newest installed version (.../node/<version>/bin/npm)
	if p, err := globNewest(filepath.Join(home, ".nvm", "versions", "node", "*", "bin", "npm"), 2); err == nil {
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
	if p, err := globNewest(filepath.Join(appData, "nvm", "v*", "npm.cmd"), 1); err == nil {
		return found(p)
	}

	// fnm on Windows (~\AppData\Local\fnm\node-versions\<version>\installation\npm.cmd)
	if p, err := globNewest(filepath.Join(localAppData, "fnm", "node-versions", "*", "installation", "npm.cmd"), 2); err == nil {
		return found(p)
	}

	// Scoop (~\scoop\apps\nodejs\current\npm.cmd)
	if p := filepath.Join(home, "scoop", "apps", "nodejs", "current", "npm.cmd"); isExecutable(p) {
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

	cmd := &exec.Cmd{
		Path: loc.path,
		Args: append([]string{loc.path}, args...),
	}

	// Prepend the node bin dir so npm can invoke `node` even in GUI contexts.
	// Use the OS-specific path list separator (: on Unix, ; on Windows).
	current := os.Getenv("PATH")
	if current == "" {
		if runtime.GOOS == "windows" {
			if sysRoot := os.Getenv("SystemRoot"); sysRoot != "" {
				current = filepath.Join(sysRoot, "System32")
			}
		} else {
			current = "/usr/bin:/bin"
		}
	}
	sep := string(os.PathListSeparator)
	cmd.Env = append(os.Environ(), "PATH="+loc.binDir+sep+current)

	return cmd, nil
}

func absolutePath(path string) (string, error) {
	cleaned := filepath.Clean(path)
	if filepath.IsAbs(cleaned) {
		return cleaned, nil
	}
	return filepath.Abs(cleaned)
}

func isExecutable(path string) bool {
	cleaned, err := absolutePath(path)
	if err != nil {
		return false
	}

	// #nosec G703 -- candidate executables are normalized to absolute paths before inspection.
	info, err := os.Stat(cleaned)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		ext := strings.ToLower(filepath.Ext(cleaned))
		return ext == ".exe" || ext == ".cmd" || ext == ".bat"
	}
	return info.Mode()&0o111 != 0
}

// globNewest returns the executable match of pattern with the highest version.
// versionDepth is how many levels above the executable the version directory
// sits (e.g. 2 for ".../<version>/bin/npm"); a leading "v" is ignored.
func globNewest(pattern string, versionDepth int) (string, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("padrão inválido %s: %w", pattern, err)
	}
	newest, newestVer := "", ""
	for _, m := range matches {
		if !isExecutable(m) {
			continue
		}
		ver := versionDir(m, versionDepth)
		if newest == "" || compareVersions(ver, newestVer) > 0 {
			newest, newestVer = m, ver
		}
	}
	if newest == "" {
		return "", fmt.Errorf("nenhum npm executável para %s", pattern)
	}
	return newest, nil
}

// versionDir returns the name of the directory versionDepth levels above path,
// without a leading "v" (".../v24.1.0/bin/npm", 2 → "24.1.0").
func versionDir(path string, versionDepth int) string {
	for range versionDepth {
		path = filepath.Dir(path)
	}
	return strings.TrimPrefix(filepath.Base(path), "v")
}

// compareVersions compares dotted numeric versions segment by segment
// (24.14.0 > 24.9.0); non-numeric segments compare as strings.
func compareVersions(a, b string) int {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(as) && i < len(bs); i++ {
		ai, aErr := strconv.Atoi(as[i])
		bi, bErr := strconv.Atoi(bs[i])
		if aErr == nil && bErr == nil {
			if ai != bi {
				return ai - bi
			}
			continue
		}
		if c := strings.Compare(as[i], bs[i]); c != 0 {
			return c
		}
	}
	return len(as) - len(bs)
}

func errNotFound() error {
	return fmt.Errorf("npm não encontrado; instale Node.js: https://nodejs.org")
}
