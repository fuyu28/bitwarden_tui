package repository

import "github.com/fuyu28/bitwarden_tui/internal/model"

type VaultRepository interface {
	IsUnlocked() (bool, error)
	Unlock(password string) error
	List() ([]model.ListItem, error)
	GetDetail(id string, itemType model.ItemType) (*model.Item, error)
	Sync() error
}
