package installer

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// asdfDataDir returns the asdf data directory ($ASDF_DATA_DIR or ~/.asdf) as
// a clean absolute path, so it can be compared with resolved npm paths.
func asdfDataDir(home string) string {
	dir := os.Getenv("ASDF_DATA_DIR")
	if dir == "" {
		dir = filepath.Join(home, ".asdf")
	}
	if abs, err := absolutePath(dir); err == nil {
		return abs
	}
	return filepath.Clean(dir)
}

// resolveAsdfNpm locates npm inside an asdf nodejs installation.
//
// asdf shims are bash scripts that call the `asdf` binary, which GUI apps
// cannot find. Instead, the version is resolved the same way asdf does for a
// process without a project directory (ASDF_NODEJS_VERSION, then the global
// ~/.tool-versions) and the real install directory is used. Falls back to the
// newest installed version.
func resolveAsdfNpm(home string) (string, error) {
	installs := filepath.Join(asdfDataDir(home), "installs", "nodejs")

	if ver := asdfNodeVersion(home); ver != "" {
		if p := filepath.Join(installs, ver, "bin", "npm"); isExecutable(p) {
			return p, nil
		}
	}

	return globNewest(filepath.Join(installs, "*", "bin", "npm"), 2)
}

// asdfNodeVersion returns the globally selected nodejs version, or "" when
// none is set (or it is "system", "ref:..." or "path:...").
func asdfNodeVersion(home string) string {
	if ver := os.Getenv("ASDF_NODEJS_VERSION"); ver != "" {
		return usableAsdfVersion(ver)
	}

	filename := os.Getenv("ASDF_DEFAULT_TOOL_VERSIONS_FILENAME")
	if filename == "" {
		filename = ".tool-versions"
	}
	f, err := os.Open(filepath.Join(home, filepath.Base(filename)))
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line, _, _ := strings.Cut(scanner.Text(), "#")
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "nodejs" {
			return usableAsdfVersion(fields[1])
		}
	}
	return ""
}

func usableAsdfVersion(ver string) string {
	if ver == "system" || strings.Contains(ver, ":") {
		return ""
	}
	return ver
}

// reshimAsdf regenerates asdf shims after a global install/uninstall so new
// package binaries become available in the shell. It is a no-op when npm does
// not come from asdf, and failures are ignored: the npm operation already
// succeeded and `asdf reshim nodejs` can be run manually.
func reshimAsdf(npmPath string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	dataDir := asdfDataDir(home)
	if !strings.HasPrefix(npmPath, dataDir+string(os.PathSeparator)) {
		return
	}

	asdfBin, err := resolveAsdfBin(home)
	if err != nil {
		return
	}
	cmd := &exec.Cmd{
		Path: asdfBin,
		Args: []string{asdfBin, "reshim", "nodejs"},
		Env:  append(os.Environ(), "ASDF_DATA_DIR="+dataDir),
	}
	_ = cmd.Run()
}

// resolveAsdfBin locates the asdf executable (Go binary in asdf >= 0.16,
// installed by Homebrew or manually; legacy installs keep it in ~/.asdf/bin).
func resolveAsdfBin(home string) (string, error) {
	if p, err := exec.LookPath("asdf"); err == nil {
		return absolutePath(p)
	}
	for _, p := range []string{
		"/opt/homebrew/bin/asdf",
		"/usr/local/bin/asdf",
		filepath.Join(asdfDataDir(home), "bin", "asdf"),
		filepath.Join(home, ".local", "bin", "asdf"),
		filepath.Join(home, "go", "bin", "asdf"),
	} {
		if isExecutable(p) {
			return p, nil
		}
	}
	return "", fmt.Errorf("asdf não encontrado")
}
