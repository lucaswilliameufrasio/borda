package secret

import "sync"

// MemoryKeyring is an in-process keychain for tests and unsupported systems.
type MemoryKeyring struct {
	mu    sync.Mutex
	items map[string]string
}

func NewMemoryKeyring() *MemoryKeyring {
	return &MemoryKeyring{items: map[string]string{}}
}

func (ring *MemoryKeyring) Set(service string, account string, secretValue string) error {
	ring.mu.Lock()
	defer ring.mu.Unlock()
	ring.items[service+"\x00"+account] = secretValue
	return nil
}

func (ring *MemoryKeyring) Get(service string, account string) (string, error) {
	ring.mu.Lock()
	defer ring.mu.Unlock()
	value, ok := ring.items[service+"\x00"+account]
	if !ok {
		return "", ErrNotFound
	}
	return value, nil
}
