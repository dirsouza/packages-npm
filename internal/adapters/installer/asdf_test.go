package installer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveAsdfNpm(t *testing.T) {
	skipOnWindows(t)
	t.Setenv("ASDF_NODEJS_VERSION", "")
	t.Setenv("ASDF_DEFAULT_TOOL_VERSIONS_FILENAME", "")

	t.Run("fallback ignora npm sem permissão de execução", func(t *testing.T) {
		home, data := t.TempDir(), t.TempDir()
		t.Setenv("ASDF_DATA_DIR", data)
		want := writeNpm(t, 0o700, data, "installs", "nodejs", "20.19.0", "bin", "npm")
		writeNpm(t, 0o600, data, "installs", "nodejs", "22.14.0", "bin", "npm")
		got, err := resolveAsdfNpm(home)
		if err != nil || got != want {
			t.Fatalf("got %q, %v; want %q", got, err, want)
		}
	})

	t.Run("usa a versão global do ~/.tool-versions", func(t *testing.T) {
		home, data := t.TempDir(), t.TempDir()
		t.Setenv("ASDF_DATA_DIR", data)
		want := writeNpm(t, 0o700, data, "installs", "nodejs", "22.14.0", "bin", "npm")
		writeNpm(t, 0o700, data, "installs", "nodejs", "24.21.0", "bin", "npm")
		toolVersions := []byte("golang 1.26.6\nnodejs 22.14.0 # global\n")
		if err := os.WriteFile(filepath.Join(home, ".tool-versions"), toolVersions, 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := resolveAsdfNpm(home)
		if err != nil || got != want {
			t.Fatalf("got %q, %v; want %q", got, err, want)
		}
	})

	t.Run("erro quando nenhuma versão é utilizável", func(t *testing.T) {
		home, data := t.TempDir(), t.TempDir()
		t.Setenv("ASDF_DATA_DIR", data)
		writeNpm(t, 0o600, data, "installs", "nodejs", "22.14.0", "bin", "npm")
		if got, err := resolveAsdfNpm(home); err == nil {
			t.Fatalf("esperava erro, got %q", got)
		}
	})
}

func TestAsdfDataDirNormalized(t *testing.T) {
	skipOnWindows(t)
	base := t.TempDir()
	t.Chdir(base)
	want := filepath.Join(base, "asdf")

	for _, env := range []string{want + "/", want + "//", "asdf", "./asdf/"} {
		t.Setenv("ASDF_DATA_DIR", env)
		if got := asdfDataDir(""); got != want {
			t.Errorf("ASDF_DATA_DIR=%q: got %q, want %q", env, got, want)
		}
	}
}

func TestAsdfNodeVersion(t *testing.T) {
	t.Setenv("ASDF_DEFAULT_TOOL_VERSIONS_FILENAME", "")

	tests := []struct {
		name, env, file, want string
	}{
		{name: "variável de ambiente tem prioridade", env: "24.21.0", file: "nodejs 22.14.0\n", want: "24.21.0"},
		{name: "lê o .tool-versions ignorando comentário", file: "nodejs 22.14.0 # global\n", want: "22.14.0"},
		{name: "system não aponta para o asdf", file: "nodejs system\n", want: ""},
		{name: "ref não aponta para uma pasta instalada", file: "nodejs ref:v1.0.2\n", want: ""},
		{name: "sem nodejs no arquivo", file: "golang 1.26.6\n", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("ASDF_NODEJS_VERSION", tt.env)
			if err := os.WriteFile(filepath.Join(home, ".tool-versions"), []byte(tt.file), 0o600); err != nil {
				t.Fatal(err)
			}
			if got := asdfNodeVersion(home); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
