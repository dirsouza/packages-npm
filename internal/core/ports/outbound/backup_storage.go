package outbound

// BackupEntry is a portable, serializable snapshot of a single package record.
// ID is included so that an ImportReplace can restore the original row identifiers.
type BackupEntry struct {
	ID          int64  `csv:"id"`
	DisplayName string `csv:"display_name"`
	PackageName string `csv:"package_name"`
	Version     string `csv:"version"`
}

// BackupStorage is the outbound port for reading and writing backups.
// Implementations are free to use any format/transport (CSV, JSON, S3, etc.).
type BackupStorage interface {
	// Write serialises entries to the given path, creating or overwriting it.
	Write(path string, entries []BackupEntry) error
	// Read deserialises entries from the given path.
	Read(path string) ([]BackupEntry, error)
}
