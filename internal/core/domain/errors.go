package domain

import "errors"

var (
	ErrEmptyDisplayName  = errors.New("display name cannot be empty")
	ErrPackageNotFound   = errors.New("package not found")
	ErrDuplicatePackage  = errors.New("a package with this name already exists")
	ErrInstallFailed     = errors.New("package installation failed")
	ErrUninstallFailed   = errors.New("package uninstallation failed")
	ErrNoPackageSelected = errors.New("no packages selected for installation")
)
