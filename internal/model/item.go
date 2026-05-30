package model

type ItemType string

const (
	TypeLogin    ItemType = "Login"
	TypeNote     ItemType = "Note"
	TypeCard     ItemType = "Card"
	TypeSSH      ItemType = "SSH Key"
	TypeIdentity ItemType = "Identity"
)

type ListItem struct {
	ID     string
	Name   string
	Folder string
	User   string
	Type   ItemType
}

type LoginDetail struct {
	Username string
	Password string
	TOTP     string
	URIs     []string
}

type CardDetail struct {
	CardholderName string
	Number         string
	Brand          string
	ExpMonth       string
	ExpYear        string
	Code           string
}

type NoteDetail struct{}

type IdentityDetail struct {
	FullName string
	Email    string
	Phone    string
	Address  string
}

type SSHKeyDetail struct {
	PrivateKey  string
	PublicKey   string
	Fingerprint string
}

type Item struct {
	ListItem
	Notes  string
	Detail any
}
