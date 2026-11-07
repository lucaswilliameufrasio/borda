package secret

import "github.com/zalando/go-keyring"

// OSKeyring uses the operating system keychain.
type OSKeyring struct{}

func (OSKeyring) Set(service string, account string, value string) error {
	return keyring.Set(service, account, value)
}

func (OSKeyring) Get(service string, account string) (string, error) {
	value, err := keyring.Get(service, account)
	if err != nil {
		return "", err
	}
	return value, nil
}
