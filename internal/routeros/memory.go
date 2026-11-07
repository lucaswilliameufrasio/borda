package routeros

import (
	"context"
	"sync"
)

// Memory is a fake router keyed by host.
type Memory struct {
	mu    sync.Mutex
	hosts map[string]string
	fail  map[string]error
}

func NewMemory() *Memory {
	return &Memory{
		hosts: map[string]string{},
		fail:  map[string]error{},
	}
}

func (reader *Memory) Set(host string, output string) {
	reader.mu.Lock()
	defer reader.mu.Unlock()
	reader.hosts[host] = output
}

func (reader *Memory) Fail(host string, err error) {
	reader.mu.Lock()
	defer reader.mu.Unlock()
	reader.fail[host] = err
}

func (reader *Memory) Read(ctx context.Context, target Target) (Reading, error) {
	if err := ctx.Err(); err != nil {
		return Reading{}, err
	}
	if target.Password == "" {
		return Reading{}, ErrAuthenticationRejected
	}
	reader.mu.Lock()
	failure := reader.fail[target.Host]
	output := reader.hosts[target.Host]
	reader.mu.Unlock()
	if failure != nil {
		return Reading{}, failure
	}
	return ParseReading(output), nil
}
