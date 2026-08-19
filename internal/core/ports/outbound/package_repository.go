package outbound

import "github.com/dirsouza/packages-npm/internal/core/domain"

// PackageRepository is the outbound port for package persistence.
type PackageRepository interface {
	FindAll(order domain.SortOrder) ([]*domain.Package, error)
	FindByID(id int64) (*domain.Package, error)
	FindByIDs(ids []int64) ([]*domain.Package, error)
	Save(pkg *domain.Package) (*domain.Package, error)
	Update(pkg *domain.Package) error
	Delete(id int64) error
	DeleteAll() error
	// Restore inserts entries with their original IDs, used by ImportReplace.
	Restore(entries []BackupEntry) error
	ReplaceAll(entries []BackupEntry) error
}
