package usecases

import (
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

	total := len(packages)
	for i, pkg := range packages {
		target := pkg.InstallTarget()
		installErr := s.installer.Install(target)
		onProgress(target, i+1, total, installErr)
	}

	return nil
}

// UninstallSelected remove globalmente os pacotes selecionados via npm.
// Usa o nome do pacote sem versão (e.g. "typescript"), pois o npm uninstall
// não aceita versão no argumento.
func (s *PackageService) UninstallSelected(ids []int64, onProgress outbound.ProgressFn) error {
	if len(ids) == 0 {
		return domain.ErrNoPackageSelected
	}

	packages, err := s.repo.FindByIDs(ids)
	if err != nil {
		return err
	}

	total := len(packages)
	for i, pkg := range packages {
		pkgName := pkg.Name.Value()
		uninstallErr := s.installer.Uninstall(pkgName)
		onProgress(pkgName, i+1, total, uninstallErr)
	}

	return nil
}

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
//   - ImportMerge:   mantém os existentes e adiciona apenas os ausentes (por package_name).
//
// Retorna o número de pacotes efetivamente inseridos.
func (s *PackageService) ImportBackup(path string, mode domain.ImportMode) (int, error) {
	entries, err := s.backup.Read(path)
	if err != nil {
		return 0, err
	}

	if mode == domain.ImportReplace {
		err := s.repo.ReplaceAll(entries)
		return len(entries), err
	}

	// ImportMerge: adiciona apenas os ausentes (compara por package_name).
	current, err := s.repo.FindAll(domain.SortByDisplayName)
	if err != nil {
		return 0, err
	}
	existing := make(map[string]struct{}, len(current))
	for _, p := range current {
		existing[p.Name.Value()] = struct{}{}
	}

	imported := 0
	for _, e := range entries {
		if _, found := existing[e.PackageName]; found {
			continue
		}
		dto := inbound.PackageDTO{
			DisplayName: e.DisplayName,
			PackageName: e.PackageName,
			Version:     e.Version,
		}
		if _, err := s.Add(dto); err != nil {
			continue // ignora duplicatas/inválidos sem interromper
		}
		imported++
	}
	return imported, nil
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
