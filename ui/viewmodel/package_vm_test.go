package viewmodel

import (
	"testing"

	"github.com/dirsouza/packages-npm/internal/core/domain"
)

func TestResolveInstallStatus(t *testing.T) {
	tests := []struct {
		version, installed string
		isInstalled        bool
		want               InstallStatus
	}{
		{version: "", installed: "5.9.3", isInstalled: true, want: StatusInstalled},
		{version: "5.9.3", installed: "5.9.3", isInstalled: true, want: StatusInstalled},
		{version: "5.0.0", installed: "5.9.3", isInstalled: true, want: StatusMismatch},
		{version: "v5.0.0", installed: "5.0.0", isInstalled: true, want: StatusInstalled},
		{version: "v5.0.0", installed: "5.9.3", isInstalled: true, want: StatusMismatch},
		{version: "^5.0.0", installed: "5.9.3", isInstalled: true, want: StatusInstalled},
		{version: "1.x", installed: "1.4.0", isInstalled: true, want: StatusInstalled},
		{version: "latest", installed: "5.9.3", isInstalled: true, want: StatusInstalled},
		{version: "5.0.0", isInstalled: false, want: StatusMissing},
	}
	for _, tt := range tests {
		v, err := domain.NewVersion(tt.version)
		if err != nil {
			t.Fatal(err)
		}
		name, _ := domain.NewPackageName("typescript")
		vm := FromDomain(&domain.Package{DisplayName: "TypeScript", Name: name, Version: v})

		vm.ResolveInstallStatus(tt.installed, tt.isInstalled)
		if vm.InstallStatus != tt.want {
			t.Errorf("versão %q, instalada %q: status = %v, want %v", tt.version, tt.installed, vm.InstallStatus, tt.want)
		}
	}
}
