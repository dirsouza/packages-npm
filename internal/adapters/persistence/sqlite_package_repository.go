package persistence

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/dirsouza/packages-npm/internal/core/domain"
	"github.com/dirsouza/packages-npm/internal/core/ports/outbound"
	"github.com/dirsouza/packages-npm/internal/infrastructure/database"
)

const schema = `
CREATE TABLE IF NOT EXISTS packages (
  id           INTEGER  PRIMARY KEY AUTOINCREMENT,
  display_name TEXT     NOT NULL,
  package_name TEXT     NOT NULL UNIQUE,
  version      TEXT     NOT NULL DEFAULT '',
  created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);`

// migrations adicionam colunas a tabelas já existentes sem recriar os dados.
var migrations = []string{
	// SQLite não permite NOT NULL DEFAULT CURRENT_TIMESTAMP em ALTER TABLE.
	// Colunas são adicionadas como NULL; Save/Update preenchem sempre.
	`ALTER TABLE packages ADD COLUMN created_at DATETIME`,
	`ALTER TABLE packages ADD COLUMN updated_at DATETIME`,
}

// SQLitePackageRepository implements outbound.PackageRepository using SQLite.
type SQLitePackageRepository struct {
	conn *database.Connection
}

// NewSQLitePackageRepository constructs the repository and applies schema + migrations.
func NewSQLitePackageRepository(conn *database.Connection) (*SQLitePackageRepository, error) {
	if _, err := conn.DB.Exec(schema); err != nil {
		return nil, fmt.Errorf("applying schema: %w", err)
	}
	if err := runMigrations(conn.DB); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}
	return &SQLitePackageRepository{conn: conn}, nil
}

// runMigrations executa cada ALTER TABLE ignorando erros de "duplicate column"
// (SQLite retorna erro ao tentar adicionar uma coluna que já existe).
func runMigrations(db *sql.DB) error {
	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			// "duplicate column name" significa que a migração já foi aplicada.
			if !strings.Contains(err.Error(), "duplicate column name") {
				return fmt.Errorf("migration %q: %w", m, err)
			}
		}
	}
	return nil
}

// Compile-time assertion.
var _ outbound.PackageRepository = (*SQLitePackageRepository)(nil)

func (r *SQLitePackageRepository) FindAll(order domain.SortOrder) ([]*domain.Package, error) {
	q := fmt.Sprintf(
		`SELECT id, display_name, package_name, version, created_at, updated_at FROM packages ORDER BY %s`, sortColumn(order))
	rows, err := r.conn.DB.Query(q)
	if err != nil {
		return nil, fmt.Errorf("FindAll: %w", err)
	}
	defer rows.Close()
	return r.scan(rows)
}

func (r *SQLitePackageRepository) FindByID(id int64) (*domain.Package, error) {
	row := r.conn.DB.QueryRow(
		`SELECT id, display_name, package_name, version, created_at, updated_at FROM packages WHERE id = ?`, id)
	pkg, err := r.scanOne(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrPackageNotFound
	}
	return pkg, err
}

func (r *SQLitePackageRepository) FindByIDs(ids []int64) ([]*domain.Package, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]

	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	q := fmt.Sprintf(
		`SELECT id, display_name, package_name, version, created_at, updated_at FROM packages WHERE id IN (%s) ORDER BY display_name`, placeholders)
	rows, err := r.conn.DB.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("FindByIDs: %w", err)
	}
	defer rows.Close()
	return r.scan(rows)
}

func (r *SQLitePackageRepository) Save(pkg *domain.Package) (*domain.Package, error) {
	now := time.Now().UTC()
	result, err := r.conn.DB.Exec(
		`INSERT INTO packages (display_name, package_name, version, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		pkg.DisplayName, pkg.Name.Value(), pkg.Version.Value(), now, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil, domain.ErrDuplicatePackage
		}
		return nil, fmt.Errorf("Save: %w", err)
	}
	pkg.ID, _ = result.LastInsertId()
	pkg.CreatedAt = now
	pkg.UpdatedAt = now
	return pkg, nil
}

func (r *SQLitePackageRepository) Update(pkg *domain.Package) error {
	now := time.Now().UTC()
	_, err := r.conn.DB.Exec(
		`UPDATE packages SET display_name = ?, package_name = ?, version = ?, updated_at = ? WHERE id = ?`,
		pkg.DisplayName, pkg.Name.Value(), pkg.Version.Value(), now, pkg.ID,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return domain.ErrDuplicatePackage
		}
		return fmt.Errorf("Update: %w", err)
	}
	pkg.UpdatedAt = now
	return nil
}

func (r *SQLitePackageRepository) Delete(id int64) error {
	_, err := r.conn.DB.Exec(`DELETE FROM packages WHERE id = ?`, id)
	return err
}

func (r *SQLitePackageRepository) DeleteAll() error {
	_, err := r.conn.DB.Exec(`DELETE FROM packages`)
	return err
}

func (r *SQLitePackageRepository) ReplaceAll(entries []outbound.BackupEntry) (err error) {
	packages, err := packagesFromBackupEntries(entries)
	if err != nil {
		return err
	}

	tx, err := r.conn.DB.Begin()
	if err != nil {
		return fmt.Errorf("ReplaceAll begin: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.Exec(`DELETE FROM packages`); err != nil {
		return fmt.Errorf("ReplaceAll delete: %w", err)
	}

	stmt, err := tx.Prepare(
		`INSERT INTO packages (id, display_name, package_name, version, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("ReplaceAll prepare: %w", err)
	}
	defer stmt.Close()

	now := time.Now().UTC()
	for i, pkg := range packages {
		if _, err = stmt.Exec(entries[i].ID, pkg.DisplayName, pkg.Name.Value(), pkg.Version.Value(), now, now); err != nil {
			return fmt.Errorf("ReplaceAll exec %q: %w", pkg.Name.Value(), err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("ReplaceAll commit: %w", err)
	}
	return nil
}

// Restore inserts entries with their original IDs (used by ImportReplace).
// Uses INSERT OR IGNORE to skip conflicts silently.
func (r *SQLitePackageRepository) Restore(entries []outbound.BackupEntry) error {
	packages, err := packagesFromBackupEntries(entries)
	if err != nil {
		return err
	}

	stmt, err := r.conn.DB.Prepare(
		`INSERT OR IGNORE INTO packages (id, display_name, package_name, version, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("Restore prepare: %w", err)
	}
	defer stmt.Close()

	now := time.Now().UTC()
	for i, pkg := range packages {
		if _, err := stmt.Exec(entries[i].ID, pkg.DisplayName, pkg.Name.Value(), pkg.Version.Value(), now, now); err != nil {
			return fmt.Errorf("Restore exec %q: %w", pkg.Name.Value(), err)
		}
	}
	return nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

const timeLayout = "2006-01-02 15:04:05"

func parseTime(s string) time.Time {
	t, err := time.Parse(timeLayout, s)
	if err != nil {
		// Tenta formato alternativo com timezone
		t, _ = time.Parse(time.RFC3339, s)
	}
	return t
}

func (r *SQLitePackageRepository) scan(rows *sql.Rows) ([]*domain.Package, error) {
	var packages []*domain.Package
	for rows.Next() {
		var id int64
		var displayName, packageName, version string
		var createdAt, updatedAt sql.NullString
		if err := rows.Scan(&id, &displayName, &packageName, &version, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		pkg, err := buildPackage(id, displayName, packageName, version, createdAt.String, updatedAt.String)
		if err != nil {
			return nil, err
		}
		packages = append(packages, pkg)
	}
	return packages, rows.Err()
}

func (r *SQLitePackageRepository) scanOne(row *sql.Row) (*domain.Package, error) {
	var id int64
	var displayName, packageName, version string
	var createdAt, updatedAt sql.NullString
	if err := row.Scan(&id, &displayName, &packageName, &version, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	return buildPackage(id, displayName, packageName, version, createdAt.String, updatedAt.String)
}

func buildPackage(id int64, displayName, packageName, version, createdAt, updatedAt string) (*domain.Package, error) {
	name, err := domain.NewPackageName(packageName)
	if err != nil {
		return nil, err
	}
	ver, err := domain.NewVersion(version)
	if err != nil {
		return nil, err
	}
	return &domain.Package{
		ID:          id,
		DisplayName: displayName,
		Name:        name,
		Version:     ver,
		CreatedAt:   parseTime(createdAt),
		UpdatedAt:   parseTime(updatedAt),
	}, nil
}

func sortColumn(order domain.SortOrder) string {
	if order == domain.SortByID {
		return "id"
	}
	return "display_name"
}

func packagesFromBackupEntries(entries []outbound.BackupEntry) ([]*domain.Package, error) {
	packages := make([]*domain.Package, len(entries))
	for i, entry := range entries {
		pkg, err := packageFromBackupEntry(entry)
		if err != nil {
			return nil, err
		}
		packages[i] = pkg
	}
	return packages, nil
}

func packageFromBackupEntry(entry outbound.BackupEntry) (*domain.Package, error) {
	name, err := domain.NewPackageName(entry.PackageName)
	if err != nil {
		return nil, fmt.Errorf("backup entry %q inválido: %w", entry.PackageName, err)
	}
	version, err := domain.NewVersion(entry.Version)
	if err != nil {
		return nil, fmt.Errorf("backup entry %q inválido: %w", entry.PackageName, err)
	}
	pkg := &domain.Package{
		ID:          entry.ID,
		DisplayName: entry.DisplayName,
		Name:        name,
		Version:     version,
	}
	if err := pkg.Validate(); err != nil {
		return nil, fmt.Errorf("backup entry %q inválido: %w", entry.PackageName, err)
	}
	return pkg, nil
}
