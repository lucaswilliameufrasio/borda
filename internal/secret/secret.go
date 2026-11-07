package secret

import "errors"

const (
	KindFile      = "file"
	KindKeychain  = "keychain"
	KindCiphera   = "ciphera"
	KindInfisical = "infisical"
)

var (
	ErrLocked      = errors.New("vault locked")
	ErrNotFound    = errors.New("secret not found")
	ErrUnavailable = errors.New("vault unavailable")
)

// Record is the persisted vault reference. It never contains a password or token.
type Record struct {
	Kind        string `json:"kind"`
	APIURL      string `json:"api_url,omitempty"`
	ProjectID   string `json:"project_id,omitempty"`
	Environment string `json:"environment,omitempty"`
	SecretKey   string `json:"secret_key,omitempty"`
	SecretName  string `json:"secret_name,omitempty"`
}

// Material is unlock data kept only in memory.
type Material struct {
	Passphrase  string
	AccessToken string
}

// Keyring stores one secret per account.
type Keyring interface {
	Set(service string, account string, secret string) error
	Get(service string, account string) (string, error)
}
