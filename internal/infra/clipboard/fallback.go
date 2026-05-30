package clipboard

import "github.com/atotto/clipboard"

type Fallback struct{}

func NewFallback() *Fallback {
	return &Fallback{}
}

func (f *Fallback) Copy(value string) error {
	return clipboard.WriteAll(value)
}

func (f *Fallback) CopySensitive(value string) error {
	return clipboard.WriteAll(value)
}

func (f *Fallback) SupportsHiddenCopy() bool {
	return false
}
