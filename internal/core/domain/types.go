package domain

// SortOrder define o critério de ordenação de listas de pacotes.
type SortOrder uint8

const (
	SortByDisplayName SortOrder = iota
	SortByID
)

// ImportMode define como um backup é aplicado ao estado atual.
type ImportMode uint8

const (
	ImportMerge ImportMode = iota
	ImportReplace
)
