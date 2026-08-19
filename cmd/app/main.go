package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/dirsouza/packages-npm/internal/adapters/backup"
	"github.com/dirsouza/packages-npm/internal/adapters/installer"
	"github.com/dirsouza/packages-npm/internal/adapters/persistence"
	"github.com/dirsouza/packages-npm/internal/core/usecases"
	"github.com/dirsouza/packages-npm/internal/infrastructure/database"
	"github.com/dirsouza/packages-npm/ui"
)

func main() {
	dbPath := resolveDBPath()

	conn, packageService, err := initializeApp(dbPath)
	if err != nil {
		log.Fatalf("falha ao inicializar aplicação: %v", err)
	}
	defer closeConnection(conn)

	ui.Run(packageService, Version, Build)
}

func initializeApp(dbPath string) (*database.Connection, *usecases.PackageService, error) {
	conn, err := database.Open(dbPath)
	if err != nil {
		return nil, nil, err
	}

	repo, err := persistence.NewSQLitePackageRepository(conn)
	if err != nil {
		closeConnection(conn)
		return nil, nil, err
	}
	if err := seedDefaultPackages(repo); err != nil {
		closeConnection(conn)
		return nil, nil, err
	}

	npmInstaller := installer.NewNpmInstaller()
	npmChecker := installer.NewNpmPackageChecker()
	csvBackup := backup.NewCSVBackupStorage()
	packageService := usecases.NewPackageService(repo, npmInstaller, npmChecker, csvBackup)

	return conn, packageService, nil
}

func closeConnection(conn *database.Connection) {
	if err := conn.Close(); err != nil {
		log.Printf("aviso: falha ao fechar banco de dados: %v", err)
	}
}

// resolveDBPath returns the user config directory path for the database file.
func resolveDBPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	dir := filepath.Join(configDir, "packages-npm")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("aviso: não foi possível criar diretório de dados: %v", err)
		return "packages.db"
	}
	return filepath.Join(dir, "packages.db")
}
