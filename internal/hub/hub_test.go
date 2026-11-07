package hub

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucaswilliameufrasio/borda/internal/routeros"
	"github.com/lucaswilliameufrasio/borda/internal/secret"
)

const routerPassword = "plane-router-secret"
const vaultPassphrase = "plane-vault-secret"

func TestShouldSeparateRoutersAndOmitSecrets(t *testing.T) {
	dir := t.TempDir()
	memory := routeros.NewMemory()
	memory.Set("connected.example", ""+
		"identity\tExample\t7.0\n"+
		"route\t192.0.2.1\ttrue\t1\t192.0.2.1%ether1\n"+
		"forward\taccept\tlan\tbridge\tether1\n"+
		"lease\t192.0.2.20\tAA:BB:CC:DD:EE:FF\tphone\n"+
		"check\t1\n")
	memory.Set("empty.example", "identity\tEmpty\t7.0\n")
	memory.Fail("offline.example", routeros.ErrUnreachable)
	keeper := secret.NewKeeper(dir, secret.NewMemoryKeyring(), nil)
	store, err := Open(dir, keeper, memory)
	if err != nil {
		t.Fatal(err)
	}

	connected, err := store.Register(context.Background(), fileInput("Casa", "connected.example", "192.0.2.1"))
	if err != nil {
		t.Fatal(err)
	}
	empty, err := store.Register(context.Background(), fileInput("Vazio", "empty.example", ""))
	if err != nil {
		t.Fatal(err)
	}
	offline, err := store.Register(context.Background(), fileInput("Fora", "offline.example", ""))
	if err != nil {
		t.Fatal(err)
	}
	if connected.ID == empty.ID || empty.ID == offline.ID {
		t.Fatal("router ids collided")
	}

	first, err := store.Snapshot(context.Background(), connected.ID)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Snapshot(context.Background(), empty.ID)
	if err != nil {
		t.Fatal(err)
	}
	third, err := store.Snapshot(context.Background(), offline.ID)
	if err != nil {
		t.Fatal(err)
	}
	if first.Session != "connected" || first.ActiveRoute == nil || first.ActiveRoute.InterfaceName != "ether1" {
		t.Fatal("active route missing")
	}
	if first.Check == nil || !first.Check.Responding || first.Check.Address != "192.0.2.1" {
		t.Fatal("check missing")
	}
	if first.Leases == nil || len(*first.Leases) != 1 || (*first.Leases)[0].HostName != "phone" {
		t.Fatal("lease missing")
	}
	if second.Session != "connected" || second.ActiveRoute != nil {
		t.Fatal("empty router invented a route")
	}
	if second.Leases == nil || len(*second.Leases) != 0 {
		t.Fatal("empty lease list missing")
	}
	if third.Session != "unreachable" || third.Routes != nil || third.ActiveRoute != nil {
		t.Fatal("offline snapshot invented a path")
	}
	encoded, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	other, err := json.Marshal(second)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(other), "192.0.2.20") || strings.Contains(string(encoded), "Empty") {
		t.Fatal("snapshots mixed")
	}
	if strings.Contains(string(encoded), routerPassword) || strings.Contains(string(other), vaultPassphrase) {
		t.Fatal("snapshot contains a secret")
	}
	emptyJSON, err := json.Marshal(second)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(emptyJSON), `"leases":[]`) || strings.Contains(string(emptyJSON), "active_route") {
		t.Fatalf("empty portrait %s", emptyJSON)
	}

	registry, err := os.ReadFile(filepath.Join(dir, "routers.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(registry), routerPassword) || strings.Contains(string(registry), vaultPassphrase) {
		t.Fatal("registry contains a secret")
	}
	if !strings.Contains(string(registry), `"username"`) {
		t.Fatal("registry dropped the username")
	}

	restartedKeeper := secret.NewKeeper(dir, secret.NewMemoryKeyring(), nil)
	restarted, err := Open(dir, restartedKeeper, memory)
	if err != nil {
		t.Fatal(err)
	}
	locked, err := restarted.Snapshot(context.Background(), connected.ID)
	if err != nil {
		t.Fatal(err)
	}
	if locked.Session != "locked" || locked.Routes != nil {
		t.Fatal("restart did not lock the file vault")
	}
	if err := restarted.Unlock(context.Background(), connected.ID, "wrong-passphrase", ""); err == nil {
		t.Fatal("accepted a wrong passphrase")
	}
	stillLocked, err := restarted.Snapshot(context.Background(), connected.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stillLocked.Session != "locked" {
		t.Fatal("failed unlock stayed open")
	}
	if err := restarted.Unlock(context.Background(), connected.ID, vaultPassphrase, ""); err != nil {
		t.Fatal(err)
	}
	opened, err := restarted.Snapshot(context.Background(), connected.ID)
	if err != nil {
		t.Fatal(err)
	}
	if opened.Session != "connected" || opened.ActiveRoute == nil || opened.ActiveRoute.InterfaceName != "ether1" {
		t.Fatal("unlock did not restore the portrait")
	}
	views := restarted.List()
	if len(views) != 3 {
		t.Fatalf("list %d", len(views))
	}
}

func TestShouldRejectMissingSetupFields(t *testing.T) {
	store := newStore(t)
	_, err := store.Register(context.Background(), RegisterInput{VaultKind: secret.KindFile})
	validation, ok := err.(*ValidationError)
	if !ok || len(validation.Fields) == 0 {
		t.Fatal("expected validation errors")
	}
}

func fileInput(name string, host string, check string) RegisterInput {
	return RegisterInput{
		Name:            name,
		Host:            host,
		Username:        "reader",
		CheckAddress:    check,
		VaultKind:       secret.KindFile,
		RouterPassword:  routerPassword,
		VaultPassphrase: vaultPassphrase,
	}
}

func newStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	store, err := Open(dir, secret.NewKeeper(dir, secret.NewMemoryKeyring(), nil), routeros.NewMemory())
	if err != nil {
		t.Fatal(err)
	}
	return store
}
