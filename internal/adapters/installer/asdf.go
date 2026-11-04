package installer

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// asdfDataDir returns the asdf data directory ($ASDF_DATA_DIR or ~/.asdf).
func asdfDataDir(home string) string {
	if dir := os.Getenv("ASDF_DATA_DIR"); dir != "" {
		return dir
	}
	return filepath.Join(home, ".asdf")
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

	matches, err := filepath.Glob(filepath.Join(installs, "*", "bin", "npm"))
	if err != nil || len(matches) == 0 {
		return "", fmt.Errorf("nenhuma versão do nodejs instalada via asdf")
	}
	newest := matches[0]
	for _, m := range matches[1:] {
		if compareVersions(versionOf(m), versionOf(newest)) > 0 {
			newest = m
		}
	}
	return newest, nil
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

// versionOf extracts "<version>" from ".../installs/nodejs/<version>/bin/npm".
func versionOf(npmPath string) string {
	return filepath.Base(filepath.Dir(filepath.Dir(npmPath)))
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
