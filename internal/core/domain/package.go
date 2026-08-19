package domain

import "time"

// Package is the core entity of the application.
// It represents an npm package that can be installed globally.
type Package struct {
	ID          int64
	DisplayName string
	Name        PackageName
	Version     Version
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// InstallTarget returns the npm install argument (e.g. "typescript@5.0.0" or "typescript").
func (p *Package) InstallTarget() string {
	if p.Version.IsDefined() {
		return p.Name.Value() + "@" + p.Version.Value()
	}
	return p.Name.Value()
}

// Validate checks business rules on the entity.
func (p *Package) Validate() error {
	if p.DisplayName == "" {
		return ErrEmptyDisplayName
	}
	return nil
}
