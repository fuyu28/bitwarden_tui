package repository

type ClipboardRepository interface {
	Copy(value string) error
	CopySensitive(value string) error
	SupportsHiddenCopy() bool
}
