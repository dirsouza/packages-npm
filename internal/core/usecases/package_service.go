package usecases

import (
	"errors"

	"github.com/dirsouza/packages-npm/internal/core/domain"
	"github.com/dirsouza/packages-npm/internal/core/ports/inbound"
	"github.com/dirsouza/packages-npm/internal/core/ports/outbound"
)

// PackageService implements inbound.PackageUseCase.
// It orchestrates all package-related use cases and delegates
// persistence and installation to outbound ports (Dependency Inversion).
type PackageService struct {
	repo      outbound.PackageRepository
	installer outbound.PackageInstaller
	checker   outbound.PackageChecker
	backup    outbound.BackupStorage
}

// NewPackageService constructs the service with its required dependencies.
// Constructor injection ensures all invariants are satisfied at wiring time.
func NewPackageService(
	repo outbound.PackageRepository,
	installer outbound.PackageInstaller,
	checker outbound.PackageChecker,
	backup outbound.BackupStorage,
) *PackageService {
	return &PackageService{repo: repo, installer: installer, checker: checker, backup: backup}
}

// Compile-time assertion: PackageService must implement PackageUseCase.
var _ inbound.PackageUseCase = (*PackageService)(nil)

func (s *PackageService) ListAll(order domain.SortOrder) ([]*domain.Package, error) {
	return s.repo.FindAll(order)
}

func (s *PackageService) Add(dto inbound.PackageDTO) (*domain.Package, error) {
	pkg, err := packageFromDTO(dto)
	if err != nil {
		return nil, err
	}
	return s.repo.Save(pkg)
}

func (s *PackageService) Update(dto inbound.PackageDTO) error {
	existing, err := s.repo.FindByID(dto.ID)
	if err != nil {
		return err
	}

	pkg, err := packageFromDTO(dto)
	if err != nil {
		return err
	}

	existing.DisplayName = pkg.DisplayName
	existing.Name = pkg.Name
	existing.Version = pkg.Version
	return s.repo.Update(existing)
}

func (s *PackageService) Delete(id int64) error {
	return s.repo.Delete(id)
}

func (s *PackageService) InstallSelected(ids []int64, onProgress outbound.ProgressFn) error {
	return s.runOnSelected(ids, onProgress, (*domain.Package).InstallTarget, s.installer.Install)
}

// UninstallSelected uses the bare package name: npm uninstall does not accept a version.
func (s *PackageService) UninstallSelected(ids []int64, onProgress outbound.ProgressFn) error {
	return s.runOnSelected(ids, onProgress, packageNameOf, s.installer.Uninstall)
}

func (s *PackageService) runOnSelected(
	ids []int64,
	onProgress outbound.ProgressFn,
	argOf func(*domain.Package) string,
	op func(string) error,
) error {
	if len(ids) == 0 {
		return domain.ErrNoPackageSelected
	}
	if onProgress == nil {
		onProgress = func(string, int, int, error) {}
	}

	packages, err := s.repo.FindByIDs(ids)
	if err != nil {
		return err
	}
	if len(packages) == 0 {
		return domain.ErrPackageNotFound
	}

	for i, pkg := range packages {
		arg := argOf(pkg)
		onProgress(arg, i+1, len(packages), op(arg))
	}
	return nil
}

func packageNameOf(pkg *domain.Package) string { return pkg.Name.Value() }

// CheckInstalled cross-references all persisted packages against the globally
// installed npm packages, returning installation info keyed by package name.
//
// Version matching rules:
//   - No version defined → installed if present at any version.
//   - Version defined    → installed correctly only if versions match exactly.
func (s *PackageService) CheckInstalled() (map[string]inbound.InstalledInfo, error) {
	globalInstalled, err := s.checker.ListGlobalInstalled()
	if err != nil {
		return nil, err
	}

	packages, err := s.repo.FindAll(domain.SortByDisplayName)
	if err != nil {
		return nil, err
	}

	result := make(map[string]inbound.InstalledInfo, len(packages))
	for _, pkg := range packages {
		name := pkg.Name.Value()
		info := inbound.InstalledInfo{PackageName: name}

		if npmPkg, found := globalInstalled[name]; found {
			info.IsInstalled = true
			info.InstalledVersion = npmPkg.Version
		}

		result[name] = info
	}
	return result, nil
}

// CheckRuntime returns the Node.js and npm versions used to install and check packages.
func (s *PackageService) CheckRuntime() (inbound.RuntimeInfo, error) {
	versions, err := s.checker.CheckRuntime()
	if err != nil {
		return inbound.RuntimeInfo{}, err
	}
	return inbound.RuntimeInfo{NodeVersion: versions.Node, NpmVersion: versions.Npm}, nil
}

// ExportBackup serializa todos os pacotes cadastrados em um arquivo CSV portável,
// ordenados pelo ID de inserção.
func (s *PackageService) ExportBackup(path string) error {
	packages, err := s.repo.FindAll(domain.SortByID)
	if err != nil {
		return err
	}

	entries := make([]outbound.BackupEntry, len(packages))
	for i, p := range packages {
		entries[i] = outbound.BackupEntry{
			ID:          p.ID,
			DisplayName: p.DisplayName,
			PackageName: p.Name.Value(),
			Version:     p.Version.Value(),
		}
	}
	return s.backup.Write(path, entries)
}

// ImportBackup restaura pacotes a partir de um arquivo CSV exportado por ExportBackup.
//
// Modos:
//   - ImportReplace: apaga todos os registros e restaura com os IDs originais.
//   - ImportMerge:   mantém os existentes e adiciona apenas os ausentes (por package_name);
//     entradas já cadastradas ou inválidas são contadas em Skipped. Qualquer outra falha
//     (ex.: banco indisponível) interrompe a importação e volta junto com o resultado parcial.
func (s *PackageService) ImportBackup(path string, mode domain.ImportMode) (inbound.ImportResult, error) {
	entries, err := s.backup.Read(path)
	if err != nil {
		return inbound.ImportResult{}, err
	}

	if mode == domain.ImportReplace {
		if err := s.repo.ReplaceAll(entries); err != nil {
			return inbound.ImportResult{}, err
		}
		return inbound.ImportResult{Imported: len(entries)}, nil
	}

	current, err := s.repo.FindAll(domain.SortByDisplayName)
	if err != nil {
		return inbound.ImportResult{}, err
	}
	existing := make(map[string]struct{}, len(current))
	for _, p := range current {
		existing[p.Name.Value()] = struct{}{}
	}

	var result inbound.ImportResult
	for _, e := range entries {
		if _, found := existing[e.PackageName]; found {
			result.Skipped++
			continue
		}
		dto := inbound.PackageDTO{DisplayName: e.DisplayName, PackageName: e.PackageName, Version: e.Version}
		if _, err := s.Add(dto); err != nil {
			if !isRejectedEntry(err) {
				return result, err
			}
			result.Skipped++
			continue
		}
		existing[e.PackageName] = struct{}{}
		result.Imported++
	}
	return result, nil
}

func isRejectedEntry(err error) bool {
	for _, rejected := range []error{
		domain.ErrDuplicatePackage,
		domain.ErrEmptyDisplayName,
		domain.ErrEmptyPackageName,
		domain.ErrInvalidPackageName,
		domain.ErrInvalidVersion,
	} {
		if errors.Is(err, rejected) {
			return true
		}
	}
	return false
}

func packageFromDTO(dto inbound.PackageDTO) (*domain.Package, error) {
	name, err := domain.NewPackageName(dto.PackageName)
	if err != nil {
		return nil, err
	}
	version, err := domain.NewVersion(dto.Version)
	if err != nil {
		return nil, err
	}
	pkg := &domain.Package{
		ID:          dto.ID,
		DisplayName: dto.DisplayName,
		Name:        name,
		Version:     version,
	}
	if err := pkg.Validate(); err != nil {
		return nil, err
	}
	return pkg, nil
}
