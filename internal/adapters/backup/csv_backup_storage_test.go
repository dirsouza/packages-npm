package backup

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/dirsouza/packages-npm/internal/core/ports/outbound"
)

func writeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "backup.csv")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCSVBackupStorageRoundTrip(t *testing.T) {
	storage := NewCSVBackupStorage()
	path := filepath.Join(t.TempDir(), "backup.csv")
	want := []outbound.BackupEntry{
		{ID: 1, DisplayName: "TypeScript", PackageName: "typescript"},
		{ID: 2, DisplayName: "HTTP, Server", PackageName: "http-server", Version: "14.1.1"},
	}

	if err := storage.Write(path, want); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := storage.Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestCSVBackupStorageReadHeader(t *testing.T) {
	storage := NewCSVBackupStorage()

	for name, content := range map[string]string{
		"BOM do Excel":                  utf8BOM + "id,display_name,package_name,version\n1,TypeScript,typescript,\n",
		"BOM com cabeçalho entre aspas": utf8BOM + `"id","display_name","package_name","version"` + "\n" + `1,"TypeScript","typescript",""` + "\n",
	} {
		t.Run("aceita "+name, func(t *testing.T) {
			got, err := storage.Read(writeFile(t, content))
			if err != nil || len(got) != 1 || got[0].PackageName != "typescript" {
				t.Fatalf("got %+v, %v", got, err)
			}
		})
	}

	for name, content := range map[string]string{
		"sem cabeçalho":       "1,TypeScript,typescript,\n",
		"colunas reordenadas": "id,package_name,display_name,version\n1,typescript,TypeScript,\n",
		"arquivo vazio":       "",
	} {
		t.Run(name, func(t *testing.T) {
			if got, err := storage.Read(writeFile(t, content)); err == nil {
				t.Fatalf("esperava erro, got %+v", got)
			}
		})
	}
}
