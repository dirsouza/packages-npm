package installer

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// writeNpm creates a fake npm file at dir/<rel...> with the given mode.
func writeNpm(t *testing.T, mode os.FileMode, elem ...string) string {
	t.Helper()
	p := filepath.Join(elem...)
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"), mode); err != nil {
		t.Fatal(err)
	}
	return p
}

func skipOnWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("usa permissão de execução Unix")
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int // sinal esperado
	}{
		{"24.14.0", "24.9.0", 1},
		{"9.11.2", "24.1.0", -1},
		{"24.21.0", "24.21.0", 0},
		{"24.21", "24.21.0", -1},
	}
	for _, tt := range tests {
		got := compareVersions(tt.a, tt.b)
		if sign(got) != tt.want {
			t.Errorf("compareVersions(%q, %q) = %d, quer sinal %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func sign(n int) int {
	switch {
	case n > 0:
		return 1
	case n < 0:
		return -1
	}
	return 0
}

func TestGlobNewest(t *testing.T) {
	skipOnWindows(t)

	t.Run("nvm: compara numericamente e ignora o prefixo v", func(t *testing.T) {
		root := t.TempDir()
		writeNpm(t, 0o700, root, "v9.11.2", "bin", "npm")
		want := writeNpm(t, 0o700, root, "v24.1.0", "bin", "npm")
		got, err := globNewest(filepath.Join(root, "*", "bin", "npm"), 2)
		if err != nil || got != want {
			t.Fatalf("got %q, %v; want %q", got, err, want)
		}
	})

	t.Run("fnm: versão três níveis acima e npm sem execução ignorado", func(t *testing.T) {
		root := t.TempDir()
		want := writeNpm(t, 0o700, root, "v20.19.0", "installation", "bin", "npm")
		writeNpm(t, 0o600, root, "v22.14.0", "installation", "bin", "npm")
		got, err := globNewest(filepath.Join(root, "*", "installation", "bin", "npm"), 3)
		if err != nil || got != want {
			t.Fatalf("got %q, %v; want %q", got, err, want)
		}
	})

	t.Run("erro quando nenhum candidato é executável", func(t *testing.T) {
		root := t.TempDir()
		writeNpm(t, 0o600, root, "v22.14.0", "bin", "npm")
		if got, err := globNewest(filepath.Join(root, "*", "bin", "npm"), 2); err == nil {
			t.Fatalf("esperava erro, got %q", got)
		}
	})
}
