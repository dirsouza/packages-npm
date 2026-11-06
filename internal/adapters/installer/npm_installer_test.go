package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNpmErrorSummary(t *testing.T) {
	tests := map[string]struct {
		output, want string
	}{
		"mantém só as linhas npm error": {
			output: "npm warn deprecated x\nnpm error code EEXIST\nnpm error path /bin/yarn\nnpm error EEXIST: file already exists\nnpm error extra\n",
			want:   "code EEXIST; path /bin/yarn; EEXIST: file already exists",
		},
		"formato npm ERR! do npm antigo": {
			output: "npm ERR! code EEXIST\nnpm ERR! path /bin/yarn\n",
			want:   "code EEXIST; path /bin/yarn",
		},
		"sem linhas npm error devolve a saída": {
			output: "  falha inesperada\n",
			want:   "falha inesperada",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := npmErrorSummary([]byte(tt.output)); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func eexistOutput(path string) []byte {
	return []byte("npm error code EEXIST\nnpm error path " + path + "\nnpm error EEXIST: file already exists\n")
}

func TestNpmFailureDetailCorepackHint(t *testing.T) {
	skipOnWindows(t)
	bin := t.TempDir()
	symlink := func(name, target string) string {
		t.Helper()
		path := filepath.Join(bin, name)
		if err := os.Symlink(target, path); err != nil {
			t.Fatal(err)
		}
		return path
	}
	script := func(name, content string) string {
		t.Helper()
		path := filepath.Join(bin, name)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}

	tests := map[string]struct {
		output []byte
		want   string
	}{
		"yarn do Corepack": {
			output: eexistOutput(symlink("yarn", "../lib/node_modules/corepack/dist/yarn.js")),
			want:   "`corepack disable yarn`",
		},
		"yarnpkg pertence ao yarn": {
			output: eexistOutput(symlink("yarnpkg", "../lib/node_modules/corepack/dist/yarnpkg.js")),
			want:   "`corepack disable yarn`",
		},
		"pnpx pertence ao pnpm": {
			output: eexistOutput(symlink("pnpx", "../lib/node_modules/corepack/dist/pnpx.js")),
			want:   "`corepack disable pnpm`",
		},
		"formato npm ERR! do npm antigo": {
			output: []byte("npm ERR! code EEXIST\nnpm ERR! path " + symlink("pnpm", "../lib/node_modules/corepack/dist/pnpm.js") + "\n"),
			want:   "`corepack disable pnpm`",
		},
		"script .cmd do Windows": {
			output: eexistOutput(script("pnpm.cmd", `@"%~dp0\node_modules\corepack\dist\pnpm.js" %*`)),
			want:   "`corepack disable pnpm`",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := npmFailureDetail(tt.output); !strings.Contains(got, tt.want) {
				t.Errorf("got %q, want conter %q", got, tt.want)
			}
		})
	}

	npmYarn := filepath.Join(t.TempDir(), "yarn")
	if err := os.Symlink("../lib/node_modules/yarn/bin/yarn.js", npmYarn); err != nil {
		t.Fatal(err)
	}

	noHint := map[string][]byte{
		"yarn instalado pelo npm":  eexistOutput(npmYarn),
		"binário fora do Corepack": eexistOutput(symlink("tsc", "../lib/node_modules/typescript/bin/tsc")),
		"outro código de erro":     []byte("npm error code E404\nnpm error path " + filepath.Join(bin, "yarn") + "\n"),
		"caminho inexistente":      eexistOutput(filepath.Join(bin, "ausente", "yarn")),
	}
	for name, output := range noHint {
		t.Run("sem dica: "+name, func(t *testing.T) {
			if got := npmFailureDetail(output); strings.Contains(got, "corepack disable") {
				t.Errorf("dica inesperada: %q", got)
			}
		})
	}
}
