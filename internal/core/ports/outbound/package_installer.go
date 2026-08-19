package outbound

// ProgressFn é o callback invocado após cada tentativa de operação num pacote.
// pkgTarget é o argumento npm usado (e.g. "typescript" ou "typescript@5.0.0").
// done / total permitem rastrear progresso. opErr é não-nil se a operação falhou.
type ProgressFn func(pkgTarget string, done, total int, opErr error)

// PackageInstaller é o outbound port para operações npm de instalação/desinstalação.
type PackageInstaller interface {
	Install(pkgTarget string) error
	Uninstall(pkgName string) error
}
