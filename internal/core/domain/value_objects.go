package domain

import (
	"regexp"
	"strings"
)

var versionRegex = regexp.MustCompile(`^$|^\*$|^[~^><]=?\d[\w.*-]*$|^\d[\w.*-]*$|^[a-zA-Z][a-zA-Z0-9._-]*$`)

// PackageName represents a non-empty npm package identifier (trimmed).
// It never starts with "-", so it cannot be read as an npm flag.
// Value Object: immutable, equality by value.
type PackageName struct {
	value string
}

func NewPackageName(raw string) (PackageName, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return PackageName{}, ErrEmptyPackageName
	}
	if strings.HasPrefix(trimmed, "-") {
		return PackageName{}, ErrInvalidPackageName
	}
	return PackageName{value: trimmed}, nil
}

func (p PackageName) Value() string             { return p.value }
func (p PackageName) String() string            { return p.value }
func (p PackageName) Equals(o PackageName) bool { return p.value == o.value }

// Version represents an optional npm semver specifier.
// Empty value means "latest" (no version pinned).
type Version struct {
	value string
}

func NewVersion(raw string) (Version, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return Version{}, nil
	}
	if !versionRegex.MatchString(trimmed) {
		return Version{}, ErrInvalidVersion
	}
	return Version{value: trimmed}, nil
}

func (v Version) Value() string { return v.value }
func (v Version) String() string {
	if v.value == "" {
		return "latest"
	}
	return v.value
}
func (v Version) IsDefined() bool       { return v.value != "" }
func (v Version) Equals(o Version) bool { return v.value == o.value }
