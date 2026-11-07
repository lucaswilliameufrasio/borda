package secret

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const keychainService = "borda"

// Keeper stores router passwords in the vault the operator chose.
type Keeper struct {
	dir    string
	keys   Keyring
	client *http.Client

	mu     sync.Mutex
	unlock map[string]Material
}

func NewKeeper(dir string, keys Keyring, client *http.Client) *Keeper {
	if client == nil {
		client = http.DefaultClient
	}
	return &Keeper{
		dir:    dir,
		keys:   keys,
		client: client,
		unlock: map[string]Material{},
	}
}

// Put stores routerPassword and keeps unlock material only in memory.
func (k *Keeper) Put(ctx context.Context, routerID string, record Record, material Material, routerPassword string) error {
	switch record.Kind {
	case KindFile:
		blob, err := Seal(material.Passphrase, routerPassword)
		if err != nil {
			return err
		}
		if err := writeSecretFile(k.dir, routerID, blob); err != nil {
			return err
		}
	case KindKeychain:
		if k.keys == nil {
			return ErrUnavailable
		}
		if err := k.keys.Set(keychainService, routerID, routerPassword); err != nil {
			return fmt.Errorf("%w: %s", ErrUnavailable, "keychain")
		}
	case KindCiphera:
		if _, err := k.fetchCiphera(ctx, record, material.AccessToken); err != nil {
			return err
		}
	case KindInfisical:
		if _, err := k.fetchInfisical(ctx, record, material.AccessToken); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown vault")
	}
	k.Remember(routerID, material)
	return nil
}

// Remember keeps unlock material in memory. It does not store the router password.
func (keeper *Keeper) Remember(routerID string, material Material) {
	keeper.mu.Lock()
	keeper.unlock[routerID] = material
	keeper.mu.Unlock()
}

// Forget drops unlock material for one router.
func (keeper *Keeper) Forget(routerID string) {
	keeper.mu.Lock()
	delete(keeper.unlock, routerID)
	keeper.mu.Unlock()
}

// Get returns the router password. Remote vaults are read at call time.
func (k *Keeper) Get(ctx context.Context, routerID string, record Record) (string, error) {
	k.mu.Lock()
	material := k.unlock[routerID]
	k.mu.Unlock()
	switch record.Kind {
	case KindFile:
		if material.Passphrase == "" {
			return "", ErrLocked
		}
		blob, err := readSecretFile(k.dir, routerID)
		if err != nil {
			return "", err
		}
		return Open(material.Passphrase, blob)
	case KindKeychain:
		if k.keys == nil {
			return "", ErrUnavailable
		}
		value, err := k.keys.Get(keychainService, routerID)
		if err != nil {
			return "", ErrNotFound
		}
		return value, nil
	case KindCiphera:
		if material.AccessToken == "" {
			return "", ErrLocked
		}
		return k.fetchCiphera(ctx, record, material.AccessToken)
	case KindInfisical:
		if material.AccessToken == "" {
			return "", ErrLocked
		}
		return k.fetchInfisical(ctx, record, material.AccessToken)
	default:
		return "", fmt.Errorf("unknown vault")
	}
}

func (k *Keeper) fetchCiphera(ctx context.Context, record Record, token string) (string, error) {
	endpoint := strings.TrimRight(record.APIURL, "/") + "/v1/projects/" + url.PathEscape(record.ProjectID) + "/secrets/" + url.PathEscape(record.Environment)
	body, err := k.getJSON(ctx, endpoint, token)
	if err != nil {
		return "", err
	}
	var secrets []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal(body, &secrets); err != nil {
		return "", ErrUnavailable
	}
	for _, item := range secrets {
		if item.Key == record.SecretKey {
			if item.Value == "" {
				return "", ErrNotFound
			}
			return item.Value, nil
		}
	}
	return "", ErrNotFound
}

func (k *Keeper) fetchInfisical(ctx context.Context, record Record, token string) (string, error) {
	endpoint, err := url.Parse(strings.TrimRight(record.APIURL, "/") + "/api/v3/secrets/raw/" + url.PathEscape(record.SecretName))
	if err != nil {
		return "", ErrUnavailable
	}
	query := endpoint.Query()
	query.Set("workspaceId", record.ProjectID)
	query.Set("environment", record.Environment)
	query.Set("secretPath", "/")
	endpoint.RawQuery = query.Encode()
	body, err := k.getJSON(ctx, endpoint.String(), token)
	if err != nil {
		return "", err
	}
	var payload struct {
		Secret struct {
			SecretValue string `json:"secretValue"`
		} `json:"secret"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", ErrUnavailable
	}
	if payload.Secret.SecretValue == "" {
		return "", ErrNotFound
	}
	return payload.Secret.SecretValue, nil
}

func (k *Keeper) getJSON(ctx context.Context, endpoint string, token string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, ErrUnavailable
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "borda")
	response, err := k.client.Do(request)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, ErrUnavailable
	}
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return nil, ErrLocked
	}
	if response.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, ErrUnavailable
	}
	return body, nil
}
