package domain

import (
	"regexp"
	"strings"
)

const maxPackageNameLength = 214

var (
	versionRegex      = regexp.MustCompile(`^$|^\*$|^[~^><]=?\d[\w.*-]*$|^\d[\w.*-]*$|^[a-zA-Z][a-zA-Z0-9._-]*$`)
	exactVersionRegex = regexp.MustCompile(`^v?(\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?)$`)
	packageNameRegex  = regexp.MustCompile(`^(@[a-zA-Z0-9~][a-zA-Z0-9._~-]*/)?[a-zA-Z0-9~][a-zA-Z0-9._~-]*$`)
)

// PackageName is an npm package identifier following npm naming rules
// (URL-safe characters and an optional @scope/). It rejects flags, URLs,
// paths and shell metacharacters, so it is safe as an npm argument.
type PackageName struct {
	value string
}

func NewPackageName(raw string) (PackageName, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return PackageName{}, ErrEmptyPackageName
	}
	if len(trimmed) > maxPackageNameLength || !packageNameRegex.MatchString(trimmed) {
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
func (v Version) IsDefined() bool { return v.value != "" }

// ExactRelease returns the pinned release without the "v" prefix that npm also
// accepts ("v5.0.0" → "5.0.0"), and false for ranges, wildcards and dist-tags.
func (v Version) ExactRelease() (string, bool) {
	match := exactVersionRegex.FindStringSubmatch(v.value)
	if match == nil {
		return "", false
	}
	return match[1], true
}
func (v Version) Equals(o Version) bool { return v.value == o.value }
