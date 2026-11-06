package backup

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/dirsouza/packages-npm/internal/core/ports/outbound"
)

const utf8BOM = "\ufeff"

var csvHeader = []string{"id", "display_name", "package_name", "version"}

// CSVBackupStorage implements outbound.BackupStorage using RFC 4180 CSV files.
// It uses only stdlib packages (encoding/csv, os, strconv) → cross-platform.
type CSVBackupStorage struct{}

// NewCSVBackupStorage constructs the adapter (stateless, no dependencies).
func NewCSVBackupStorage() *CSVBackupStorage { return &CSVBackupStorage{} }

// Compile-time assertion.
var _ outbound.BackupStorage = (*CSVBackupStorage)(nil)

// Write serialises entries to a CSV file at path, ordered by ID.
// Creates or overwrites the file; compatible with macOS, Linux and Windows.
func (c *CSVBackupStorage) Write(path string, entries []outbound.BackupEntry) (err error) {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("backup: criar arquivo: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("backup: fechar arquivo: %w", closeErr))
		}
	}()

	w := csv.NewWriter(f)

	if err := w.Write(csvHeader); err != nil {
		return fmt.Errorf("backup: gravar cabeçalho: %w", err)
	}

	for _, e := range entries {
		row := []string{
			strconv.FormatInt(e.ID, 10),
			e.DisplayName,
			e.PackageName,
			e.Version,
		}
		if err := w.Write(row); err != nil {
			return fmt.Errorf("backup: gravar linha %q: %w", e.PackageName, err)
		}
	}

	w.Flush()
	return w.Error()
}

// Read deserialises a CSV backup file previously written by Write.
func (c *CSVBackupStorage) Read(path string) ([]outbound.BackupEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("backup: abrir arquivo: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.TrimLeadingSpace = true

	rows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("backup: ler CSV: %w", err)
	}

	if len(rows) < 1 {
		return nil, errors.New("backup: arquivo CSV vazio")
	}
	if err := checkHeader(rows[0]); err != nil {
		return nil, err
	}

	entries := make([]outbound.BackupEntry, 0, len(rows)-1)
	for i, row := range rows[1:] {
		if len(row) < 4 {
			return nil, fmt.Errorf("backup: linha %d incompleta (esperado 4 colunas, encontrado %d)", i+2, len(row))
		}
		id, parseErr := strconv.ParseInt(row[0], 10, 64)
		if parseErr != nil {
			return nil, fmt.Errorf("backup: linha %d — ID inválido %q: %w", i+2, row[0], parseErr)
		}
		entries = append(entries, outbound.BackupEntry{
			ID:          id,
			DisplayName: row[1],
			PackageName: row[2],
			Version:     row[3],
		})
	}
	return entries, nil
}

// checkHeader rejects files whose columns do not match csvHeader, so a file
// without header or with reordered columns is not imported with shifted data.
func checkHeader(header []string) error {
	got := slices.Clone(header)
	if len(got) > 0 {
		got[0] = strings.TrimPrefix(got[0], utf8BOM)
	}
	if !slices.Equal(got, csvHeader) {
		return fmt.Errorf("backup: cabeçalho inválido %v (esperado %v)", header, csvHeader)
	}
	return nil
}
