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
		{raw: "payload.tgz", wantErr: ErrInvalidPackageName},
		{raw: "PAYLOAD.TGZ", wantErr: ErrInvalidPackageName},
		{raw: "payload.tar", wantErr: ErrInvalidPackageName},
		{raw: "payload.tar.gz", wantErr: ErrInvalidPackageName},
		{raw: "payload.TAR.GZ", wantErr: ErrInvalidPackageName},
		{raw: "@scope/payload.tgz", want: "@scope/payload.tgz"},
		{raw: "payload-tgz", want: "payload-tgz"},
		{raw: "payload.tgz.js", want: "payload.tgz.js"},
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

func TestVersionExactRelease(t *testing.T) {
	tests := map[string]string{
		"5.0.0":         "5.0.0",
		"v5.0.0":        "5.0.0",
		"1.2.3-beta.1":  "1.2.3-beta.1",
		"v1.2.3-beta.1": "1.2.3-beta.1",
		"":              "",
		"^5.0.0":        "",
		"~1.2":          "",
		"1.x":           "",
		"*":             "",
		"latest":        "",
		"vnext":         "",
	}
	for raw, want := range tests {
		v, err := NewVersion(raw)
		if err != nil {
			t.Fatalf("NewVersion(%q): %v", raw, err)
		}
		got, ok := v.ExactRelease()
		if got != want || ok != (want != "") {
			t.Errorf("Version(%q).ExactRelease() = (%q, %v), want (%q, %v)", raw, got, ok, want, want != "")
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
