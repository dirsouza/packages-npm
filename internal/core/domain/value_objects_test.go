package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestNewPackageName(t *testing.T) {
	tests := []struct {
		raw     string
		want    string
		wantErr error
	}{
		{raw: "typescript", want: "typescript"},
		{raw: "  @nestjs/cli  ", want: "@nestjs/cli"},
		{raw: "JSONStream", want: "JSONStream"},
		{raw: "lodash.merge", want: "lodash.merge"},
		{raw: "   ", wantErr: ErrEmptyPackageName},
		{raw: "--registry=http://evil.example", wantErr: ErrInvalidPackageName},
		{raw: "-g", wantErr: ErrInvalidPackageName},
		{raw: "typescript&calc", wantErr: ErrInvalidPackageName},
		{raw: "a|powershell", wantErr: ErrInvalidPackageName},
		{raw: "%PATH%", wantErr: ErrInvalidPackageName},
		{raw: "https://evil.example/ts.tgz", wantErr: ErrInvalidPackageName},
		{raw: "file:../x", wantErr: ErrInvalidPackageName},
		{raw: "git+ssh://host/repo", wantErr: ErrInvalidPackageName},
		{raw: "foo bar", wantErr: ErrInvalidPackageName},
		{raw: "@scope", wantErr: ErrInvalidPackageName},
		{raw: strings.Repeat("a", maxPackageNameLength+1), wantErr: ErrInvalidPackageName},
	}
	for _, tt := range tests {
		got, err := NewPackageName(tt.raw)
		if !errors.Is(err, tt.wantErr) {
			t.Errorf("NewPackageName(%q) err = %v, want %v", tt.raw, err, tt.wantErr)
			continue
		}
		if err == nil && got.Value() != tt.want {
			t.Errorf("NewPackageName(%q) = %q, want %q", tt.raw, got.Value(), tt.want)
		}
	}
}

func TestNewVersion(t *testing.T) {
	for _, raw := range []string{"", "5.0.0", "^5", "~1.2", ">=2", "1.x", "*", "latest", "1.2.3-beta.1"} {
		if _, err := NewVersion(raw); err != nil {
			t.Errorf("NewVersion(%q) err = %v", raw, err)
		}
	}
	for _, raw := range []string{"5 0", "1.0&calc", "-1"} {
		if _, err := NewVersion(raw); !errors.Is(err, ErrInvalidVersion) {
			t.Errorf("NewVersion(%q) err = %v, want %v", raw, err, ErrInvalidVersion)
		}
	}
}

func TestVersionIsExact(t *testing.T) {
	tests := map[string]bool{
		"5.0.0":        true,
		"1.2.3-beta.1": true,
		"":             false,
		"^5.0.0":       false,
		"~1.2":         false,
		"1.x":          false,
		"*":            false,
		"latest":       false,
	}
	for raw, want := range tests {
		v, err := NewVersion(raw)
		if err != nil {
			t.Fatalf("NewVersion(%q): %v", raw, err)
		}
		if got := v.IsExact(); got != want {
			t.Errorf("Version(%q).IsExact() = %v, want %v", raw, got, want)
		}
	}
}

func TestPackageValidateRejectsBlankDisplayName(t *testing.T) {
	for _, name := range []string{"", "   ", "\t"} {
		if err := (&Package{DisplayName: name}).Validate(); !errors.Is(err, ErrEmptyDisplayName) {
			t.Errorf("Validate(%q) err = %v, want %v", name, err, ErrEmptyDisplayName)
		}
	}
}
