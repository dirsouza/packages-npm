package installer

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const shimSniffBytes = 4096

// corepackHint explains an npm EEXIST failure caused by a Corepack shim
// (yarn, yarnpkg, pnpm, pnpx): npm refuses to overwrite a binary it does not
// own, and `corepack disable` is what releases it. It returns "" otherwise.
func corepackHint(output []byte) string {
	code, path := npmErrorField(output, "code"), npmErrorField(output, "path")
	if code != "EEXIST" || path == "" {
		return ""
	}
	command, ok := corepackCommand(path)
	if !ok || !isCorepackShim(path) {
		return ""
	}
	return fmt.Sprintf("%q é gerenciado pelo Corepack; rode `corepack disable %s` e tente de novo", filepath.Base(path), command)
}

func npmErrorField(output []byte, key string) string {
	for line := range strings.Lines(string(output)) {
		if rest, ok := cutNpmErrorPrefix(line); ok {
			if value, ok := strings.CutPrefix(rest, key+" "); ok {
				return strings.TrimSpace(value)
			}
		}
	}
	return ""
}

// corepackCommand maps a shim file to the package manager name that
// `corepack disable` expects (yarnpkg belongs to yarn, pnpx to pnpm).
func corepackCommand(path string) (string, bool) {
	name := strings.ToLower(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	switch name {
	case "yarn", "yarnpkg":
		return "yarn", true
	case "pnpm", "pnpx":
		return "pnpm", true
	}
	return "", false
}

// isCorepackShim checks the file itself: on Unix the shim is a symlink into
// corepack/dist; on Windows it is a .cmd/.ps1 script that calls Corepack.
func isCorepackShim(path string) bool {
	if target, err := os.Readlink(path); err == nil {
		return strings.Contains(filepath.ToSlash(target), "corepack/")
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	head, err := io.ReadAll(io.LimitReader(f, shimSniffBytes))
	return err == nil && bytes.Contains(bytes.ToLower(head), []byte("corepack"))
}
