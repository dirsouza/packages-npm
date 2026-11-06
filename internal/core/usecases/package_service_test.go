package usecases

import (
	"errors"
	"testing"

	"github.com/dirsouza/packages-npm/internal/core/domain"
	"github.com/dirsouza/packages-npm/internal/core/ports/inbound"
	"github.com/dirsouza/packages-npm/internal/core/ports/outbound"
)

type fakeRepository struct {
	outbound.PackageRepository
	packages []*domain.Package
}

func (r *fakeRepository) FindAll(domain.SortOrder) ([]*domain.Package, error) { return r.packages, nil }

func (r *fakeRepository) FindByIDs([]int64) ([]*domain.Package, error) { return r.packages, nil }

func (r *fakeRepository) Save(pkg *domain.Package) (*domain.Package, error) {
	r.packages = append(r.packages, pkg)
	return pkg, nil
}

type fakeInstaller struct {
	installed, uninstalled []string
}

func (f *fakeInstaller) Install(target string) error {
	f.installed = append(f.installed, target)
	return nil
}

func (f *fakeInstaller) Uninstall(name string) error {
	f.uninstalled = append(f.uninstalled, name)
	return nil
}

type fakeBackup struct {
	outbound.BackupStorage
	entries []outbound.BackupEntry
}

func (f *fakeBackup) Read(string) ([]outbound.BackupEntry, error) { return f.entries, nil }

func mustPackage(t *testing.T, name, version string) *domain.Package {
	t.Helper()
	pkg, err := packageFromDTO(inbound.PackageDTO{DisplayName: name, PackageName: name, Version: version})
	if err != nil {
		t.Fatal(err)
	}
	return pkg
}

func TestRunOnSelected(t *testing.T) {
	repo := &fakeRepository{packages: []*domain.Package{mustPackage(t, "typescript", "5.0.0")}}
	installer := &fakeInstaller{}
	service := NewPackageService(repo, installer, nil, nil)

	if err := service.InstallSelected(nil, nil); !errors.Is(err, domain.ErrNoPackageSelected) {
		t.Fatalf("sem IDs: err = %v, want %v", err, domain.ErrNoPackageSelected)
	}
	if err := service.InstallSelected([]int64{1}, nil); err != nil {
		t.Fatalf("InstallSelected sem onProgress: %v", err)
	}
	if err := service.UninstallSelected([]int64{1}, nil); err != nil {
		t.Fatalf("UninstallSelected sem onProgress: %v", err)
	}
	if installer.installed[0] != "typescript@5.0.0" || installer.uninstalled[0] != "typescript" {
		t.Fatalf("installed=%v uninstalled=%v", installer.installed, installer.uninstalled)
	}

	empty := NewPackageService(&fakeRepository{}, installer, nil, nil)
	if err := empty.UninstallSelected([]int64{99}, nil); !errors.Is(err, domain.ErrPackageNotFound) {
		t.Fatalf("IDs inexistentes: err = %v, want %v", err, domain.ErrPackageNotFound)
	}
}

func TestImportBackupMergeCountsSkipped(t *testing.T) {
	repo := &fakeRepository{packages: []*domain.Package{mustPackage(t, "yarn", "")}}
	backup := &fakeBackup{entries: []outbound.BackupEntry{
		{DisplayName: "Yarn", PackageName: "yarn"},
		{DisplayName: "TypeScript", PackageName: "typescript"},
		{DisplayName: "TypeScript de novo", PackageName: "typescript"},
		{DisplayName: "Ruim", PackageName: "x&calc"},
	}}
	service := NewPackageService(repo, nil, nil, backup)

	got, err := service.ImportBackup("backup.csv", domain.ImportMerge)
	if err != nil {
		t.Fatal(err)
	}
	if want := (inbound.ImportResult{Imported: 1, Skipped: 3}); got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
