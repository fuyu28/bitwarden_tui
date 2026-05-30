package rbw

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/fuyu28/bitwarden_tui/internal/model"
)


type Client struct{}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) IsUnlocked() (bool, error) {
	cmd := exec.Command("rbw", "unlocked")
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		}
		return false, fmt.Errorf("rbw unlocked: %w", err)
	}
	return true, nil
}

func (c *Client) Unlock(password string) error {
	cmd := exec.Command("rbw", "unlock")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("rbw unlock start: %w", err)
	}
	if _, err := fmt.Fprintln(stdin, password); err != nil {
		return fmt.Errorf("write password: %w", err)
	}
	stdin.Close()
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("rbw unlock: %w", err)
	}
	return nil
}

type listRawItem struct {
	ID     string          `json:"id"`
	Name   string          `json:"name"`
	User   *string         `json:"user"`
	Folder *string         `json:"folder"`
	URIs   json.RawMessage `json:"uris"`
	Type   string          `json:"type"`
}

func (c *Client) List() ([]model.ListItem, error) {
	out, err := exec.Command("rbw", "list", "--raw").Output()
	if err != nil {
		return nil, fmt.Errorf("rbw list --raw: %w", err)
	}

	var raws []listRawItem
	if err := json.Unmarshal(out, &raws); err != nil {
		return nil, fmt.Errorf("parse list: %w", err)
	}

	items := make([]model.ListItem, 0, len(raws))
	for _, raw := range raws {
		item := model.ListItem{
			ID:   raw.ID,
			Name: raw.Name,
			Type: model.ItemType(raw.Type),
		}
		if raw.User != nil {
			item.User = *raw.User
		}
		if raw.Folder != nil {
			item.Folder = *raw.Folder
		}
		items = append(items, item)
	}
	return items, nil
}

type detailRaw struct {
	ID     string          `json:"id"`
	Name   string          `json:"name"`
	Folder *string         `json:"folder"`
	Notes  *string         `json:"notes"`
	Data   json.RawMessage `json:"data"`
}

type uriEntry struct {
	URI       string  `json:"uri"`
	MatchType *string `json:"match_type"`
}

type loginData struct {
	Username string     `json:"username"`
	Password string     `json:"password"`
	TOTP     *string    `json:"totp"`
	URIs     []uriEntry `json:"uris"`
}

type cardData struct {
	CardholderName string `json:"cardholder_name"`
	Number         string `json:"number"`
	Brand          string `json:"brand"`
	ExpMonth       string `json:"exp_month"`
	ExpYear        string `json:"exp_year"`
	Code           string `json:"code"`
}

type sshData struct {
	PublicKey   string `json:"public_key"`
	Fingerprint string `json:"fingerprint"`
	PrivateKey  string `json:"private_key"`
}

func (c *Client) GetDetail(id string, itemType model.ItemType) (*model.Item, error) {
	out, err := exec.Command("rbw", "get", id, "--raw").Output()
	if err != nil {
		return nil, fmt.Errorf("rbw get %s --raw: %w", id, err)
	}

	var raw detailRaw
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("parse detail: %w", err)
	}

	item := &model.Item{
		ListItem: model.ListItem{ID: raw.ID, Name: raw.Name, Type: itemType},
	}
	if raw.Folder != nil {
		item.Folder = *raw.Folder
	}
	if raw.Notes != nil {
		item.Notes = *raw.Notes
	}

	switch itemType {
	case model.TypeLogin:
		var d loginData
		if raw.Data != nil {
			if err := json.Unmarshal(raw.Data, &d); err != nil {
				return nil, fmt.Errorf("parse login data: %w", err)
			}
		}
		uris := make([]string, 0, len(d.URIs))
		for _, u := range d.URIs {
			if u.URI != "" {
				uris = append(uris, u.URI)
			}
		}
		detail := &model.LoginDetail{
			Username: d.Username,
			Password: d.Password,
			URIs:     uris,
		}
		if d.TOTP != nil {
			totpOut, err := exec.Command("rbw", "code", id).Output()
			if err == nil {
				detail.TOTP = strings.TrimSpace(string(totpOut))
			}
		}
		item.Detail = detail

	case model.TypeCard:
		var d cardData
		if raw.Data != nil {
			if err := json.Unmarshal(raw.Data, &d); err != nil {
				return nil, fmt.Errorf("parse card data: %w", err)
			}
		}
		item.Detail = &model.CardDetail{
			CardholderName: d.CardholderName,
			Number:         d.Number,
			Brand:          d.Brand,
			ExpMonth:       d.ExpMonth,
			ExpYear:        d.ExpYear,
			Code:           d.Code,
		}

	case model.TypeSSH:
		var d sshData
		if raw.Data != nil {
			if err := json.Unmarshal(raw.Data, &d); err != nil {
				return nil, fmt.Errorf("parse ssh data: %w", err)
			}
		}
		item.Detail = &model.SSHKeyDetail{
			PublicKey:   d.PublicKey,
			Fingerprint: d.Fingerprint,
			PrivateKey:  d.PrivateKey,
		}

	case model.TypeNote:
		item.Detail = &model.NoteDetail{}
	}

	return item, nil
}

func (c *Client) Sync() error {
	if out, err := exec.Command("rbw", "sync").CombinedOutput(); err != nil {
		return fmt.Errorf("rbw sync: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}
