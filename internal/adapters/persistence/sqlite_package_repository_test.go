package persistence

import (
	"testing"

	"github.com/dirsouza/packages-npm/internal/core/domain"
	"github.com/dirsouza/packages-npm/internal/core/ports/outbound"
	"github.com/dirsouza/packages-npm/internal/infrastructure/database"
)

func newTestRepository(t *testing.T) *SQLitePackageRepository {
	t.Helper()
	conn, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	repo, err := NewSQLitePackageRepository(conn)
	if err != nil {
		t.Fatal(err)
	}
	return repo
}

func names(t *testing.T, repo *SQLitePackageRepository) []string {
	t.Helper()
	packages, err := repo.FindAll(domain.SortByID)
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	out := make([]string, len(packages))
	for i, p := range packages {
		out[i] = p.Name.Value()
	}
	return out
}

func TestFindSkipsInvalidRows(t *testing.T) {
	repo := newTestRepository(t)
	if _, err := repo.conn.DB.Exec(
		`INSERT INTO packages (id, display_name, package_name, version) VALUES
		 (1, 'Válido', 'typescript', ''),
		 (2, 'Nome legado', '-g', ''),
		 (3, '   ', 'yarn', ''),
		 (4, 'Versão legada', 'ntl', '1 2')`,
	); err != nil {
		t.Fatal(err)
	}

	if got := names(t, repo); len(got) != 1 || got[0] != "typescript" {
		t.Fatalf("FindAll: got %v, want [typescript]", got)
	}
	packages, err := repo.FindByIDs([]int64{1, 2, 3, 4})
	if err != nil {
		t.Fatalf("FindByIDs: %v", err)
	}
	if len(packages) != 1 || packages[0].Name.Value() != "typescript" {
		t.Fatalf("FindByIDs: got %d pacotes, want só typescript", len(packages))
	}
}

func TestReplaceAll(t *testing.T) {
	seed := func(t *testing.T, repo *SQLitePackageRepository) {
		t.Helper()
		if err := repo.ReplaceAll([]outbound.BackupEntry{{ID: 1, DisplayName: "Yarn", PackageName: "yarn"}}); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("substitui tudo mantendo os IDs", func(t *testing.T) {
		repo := newTestRepository(t)
		seed(t, repo)
		err := repo.ReplaceAll([]outbound.BackupEntry{
			{ID: 7, DisplayName: "TypeScript", PackageName: "typescript"},
			{ID: 9, DisplayName: "NestJS CLI", PackageName: "@nestjs/cli"},
		})
		if err != nil {
			t.Fatal(err)
		}
		pkg, err := repo.FindByID(9)
		if err != nil || pkg.Name.Value() != "@nestjs/cli" {
			t.Fatalf("FindByID(9) = %+v, %v", pkg, err)
		}
		if got := names(t, repo); len(got) != 2 {
			t.Fatalf("got %v, want 2 pacotes", got)
		}
	})

	for name, entries := range map[string][]outbound.BackupEntry{
		"entrada inválida": {
			{ID: 1, DisplayName: "TypeScript", PackageName: "typescript"},
			{ID: 2, DisplayName: "Ruim", PackageName: "x&calc"},
		},
		"nome duplicado": {
			{ID: 1, DisplayName: "TypeScript", PackageName: "typescript"},
			{ID: 2, DisplayName: "TypeScript 2", PackageName: "typescript"},
		},
	} {
		t.Run(name+" mantém os dados anteriores", func(t *testing.T) {
			repo := newTestRepository(t)
			seed(t, repo)
			if err := repo.ReplaceAll(entries); err == nil {
				t.Fatal("esperava erro")
			}
			if got := names(t, repo); len(got) != 1 || got[0] != "yarn" {
				t.Fatalf("got %v, want [yarn]", got)
			}
		})
	}
}
