package secret

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestShouldReadCipheraSecretWithoutStoringIt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/projects/proj/secrets/read-only" {
			t.Errorf("path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer token-1" {
			t.Errorf("missing bearer")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"key":"OTHER","value":"nope"},{"key":"ROUTER_PASSWORD","value":"from-ciphera"}]`))
	}))
	defer server.Close()

	dir := t.TempDir()
	keeper := NewKeeper(dir, NewMemoryKeyring(), server.Client())
	record := Record{Kind: KindCiphera, APIURL: server.URL, ProjectID: "proj", Environment: "read-only", SecretKey: "ROUTER_PASSWORD"}
	if err := keeper.Put(context.Background(), "r1", record, Material{AccessToken: "token-1"}, ""); err != nil {
		t.Fatal(err)
	}
	got, err := keeper.Get(context.Background(), "r1", record)
	if err != nil {
		t.Fatal(err)
	}
	if got != "from-ciphera" {
		t.Fatalf("got %q", got)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		body, readErr := os.ReadFile(filepath.Join(dir, entry.Name()))
		if readErr == nil && stringIndex(body, []byte("from-ciphera")) >= 0 {
			t.Fatal("ciphera secret was written to disk")
		}
	}
}

func TestShouldReadInfisicalSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/secrets/raw/ROUTER_PASSWORD" {
			t.Errorf("path %s", r.URL.Path)
		}
		if r.URL.Query().Get("workspaceId") != "workspace" || r.URL.Query().Get("environment") != "prod" {
			t.Errorf("query %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"secret":{"secretValue":"from-infisical"}}`))
	}))
	defer server.Close()

	keeper := NewKeeper(t.TempDir(), NewMemoryKeyring(), server.Client())
	record := Record{Kind: KindInfisical, APIURL: server.URL, ProjectID: "workspace", Environment: "prod", SecretName: "ROUTER_PASSWORD"}
	if err := keeper.Put(context.Background(), "r1", record, Material{AccessToken: "token-2"}, ""); err != nil {
		t.Fatal(err)
	}
	got, err := keeper.Get(context.Background(), "r1", record)
	if err != nil {
		t.Fatal(err)
	}
	if got != "from-infisical" {
		t.Fatalf("got %q", got)
	}
}

func TestShouldLockFileVaultWithoutPassphrase(t *testing.T) {
	dir := t.TempDir()
	keeper := NewKeeper(dir, NewMemoryKeyring(), nil)
	record := Record{Kind: KindFile}
	if err := keeper.Put(context.Background(), "r1", record, Material{Passphrase: "phrase"}, "router-secret"); err != nil {
		t.Fatal(err)
	}
	keeper.mu.Lock()
	delete(keeper.unlock, "r1")
	keeper.mu.Unlock()
	if _, err := keeper.Get(context.Background(), "r1", record); err != ErrLocked {
		t.Fatalf("err %v", err)
	}
}

func TestShouldRoundTripKeychain(t *testing.T) {
	keeper := NewKeeper(t.TempDir(), NewMemoryKeyring(), nil)
	record := Record{Kind: KindKeychain}
	if err := keeper.Put(context.Background(), "r1", record, Material{}, "router-secret"); err != nil {
		t.Fatal(err)
	}
	got, err := keeper.Get(context.Background(), "r1", record)
	if err != nil {
		t.Fatal(err)
	}
	if got != "router-secret" {
		t.Fatalf("got %q", got)
	}
}
