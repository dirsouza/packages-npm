package main

import (
	"errors"

	"github.com/dirsouza/packages-npm/internal/core/domain"
	"github.com/dirsouza/packages-npm/internal/core/ports/outbound"
)

func seedDefaultPackages(repo outbound.PackageRepository) error {
	packages, err := repo.FindAll(domain.SortByID)
	if err != nil || len(packages) > 0 {
		return err
	}

	defaults := []struct {
		displayName string
		packageName string
		version     string
	}{
		{displayName: "TypeScript", packageName: "typescript"},
		{displayName: "NestJS CLI", packageName: "@nestjs/cli"},
		{displayName: "NPM Check", packageName: "npm-check"},
		{displayName: "Yarn", packageName: "yarn"},
		{displayName: "Yarn Check", packageName: "yarn-check"},
		{displayName: "NTL - Node Task List", packageName: "ntl"},
		{displayName: "HTTP Server", packageName: "http-server"},
		{displayName: "Serverless Offline", packageName: "serverless-offline"},
	}

	for _, item := range defaults {
		name, err := domain.NewPackageName(item.packageName)
		if err != nil {
			return err
		}
		version, err := domain.NewVersion(item.version)
		if err != nil {
			return err
		}
		pkg := &domain.Package{
			DisplayName: item.displayName,
			Name:        name,
			Version:     version,
		}
		if err := pkg.Validate(); err != nil {
			return err
		}
		if _, err := repo.Save(pkg); err != nil && !errors.Is(err, domain.ErrDuplicatePackage) {
			return err
		}
	}

	return nil
}
