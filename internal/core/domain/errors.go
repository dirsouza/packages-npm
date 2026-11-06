package domain

import "errors"

var (
	ErrEmptyDisplayName  = errors.New("display name cannot be empty")
	ErrEmptyPackageName  = errors.New("package name cannot be empty")
	ErrInvalidVersion    = errors.New("invalid version format: use a semver, range, wildcard, dist-tag or leave it empty")
	ErrPackageNotFound   = errors.New("package not found")
	ErrDuplicatePackage  = errors.New("a package with this name already exists")
	ErrInstallFailed     = errors.New("package installation failed")
	ErrUninstallFailed   = errors.New("package uninstallation failed")
	ErrNoPackageSelected = errors.New("no packages selected for installation")
)
