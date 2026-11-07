package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/scrypt"
)

const fileMagic = "BORDA1"
const scryptN = 16384
const scryptR = 8
const scryptP = 1
const saltLen = 16
const nonceLen = 12

// Seal encrypts plaintext with a passphrase. The passphrase is not stored.
func Seal(passphrase string, plaintext string) ([]byte, error) {
	if passphrase == "" {
		return nil, errors.New("missing passphrase")
	}
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("salt: %w", err)
	}
	key, err := deriveKey(passphrase, salt)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	nonce := make([]byte, nonceLen)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}
	body := aead.Seal(nil, nonce, []byte(plaintext), nil)
	out := make([]byte, 0, len(fileMagic)+1+saltLen+nonceLen+len(body))
	out = append(out, []byte(fileMagic)...)
	out = append(out, '\n')
	out = append(out, salt...)
	out = append(out, nonce...)
	out = append(out, body...)
	return out, nil
}

// Open decrypts a blob produced by Seal.
func Open(passphrase string, blob []byte) (string, error) {
	prefix := len(fileMagic) + 1
	if len(blob) < prefix+saltLen+nonceLen || string(blob[:len(fileMagic)]) != fileMagic || blob[len(fileMagic)] != '\n' {
		return "", errors.New("unknown secret file")
	}
	salt := blob[prefix : prefix+saltLen]
	nonce := blob[prefix+saltLen : prefix+saltLen+nonceLen]
	body := blob[prefix+saltLen+nonceLen:]
	key, err := deriveKey(passphrase, salt)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("gcm: %w", err)
	}
	plain, err := aead.Open(nil, nonce, body, nil)
	if err != nil {
		return "", ErrLocked
	}
	return string(plain), nil
}

func deriveKey(passphrase string, salt []byte) ([]byte, error) {
	key, err := scrypt.Key([]byte(passphrase), salt, scryptN, scryptR, scryptP, 32)
	if err != nil {
		return nil, fmt.Errorf("scrypt: %w", err)
	}
	return key, nil
}

func writeSecretFile(dir string, routerID string, blob []byte) error {
	folder := filepath.Join(dir, "secrets")
	if err := os.MkdirAll(folder, 0o700); err != nil {
		return err
	}
	path := filepath.Join(folder, routerID+".bin")
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, blob, 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

func readSecretFile(dir string, routerID string) ([]byte, error) {
	blob, err := os.ReadFile(filepath.Join(dir, "secrets", routerID+".bin"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return blob, nil
}
