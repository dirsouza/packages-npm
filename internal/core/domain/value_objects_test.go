package domain

import (
	"errors"
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
		{raw: "   ", wantErr: ErrEmptyPackageName},
		{raw: "--registry=http://evil.example", wantErr: ErrInvalidPackageName},
		{raw: " -g", wantErr: ErrInvalidPackageName},
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
	if _, err := NewVersion("5 0"); !errors.Is(err, ErrInvalidVersion) {
		t.Errorf("NewVersion(%q) err = %v, want %v", "5 0", err, ErrInvalidVersion)
	}
}
