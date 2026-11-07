package hub

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lucaswilliameufrasio/borda/internal/routeros"
	"github.com/lucaswilliameufrasio/borda/internal/secret"
)

func TestShouldRememberOneRowPerDevice(t *testing.T) {
	dir := t.TempDir()
	memory := routeros.NewMemory()
	withDevice := "identity\tExample\t7.0\n" +
		"route\t192.0.2.1\ttrue\t1\t192.0.2.1%ether1\n" +
		"lease\t192.0.2.20\tAA:BB:CC:DD:EE:FF\tlan-host-secret\n"
	withoutDevice := "identity\tExample\t7.0\nroute\t192.0.2.1\ttrue\t1\t192.0.2.1%ether1\n"
	memory.Set("edge.example", withDevice)
	store, err := Open(dir, secret.NewKeeper(dir, secret.NewMemoryKeyring(), nil), memory)
	if err != nil {
		t.Fatal(err)
	}
	moment := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return moment }
	router, err := store.Register(context.Background(), fileInput("Borda", "edge.example", "192.0.2.1"))
	if err != nil {
		t.Fatal(err)
	}

	first, err := store.Snapshot(context.Background(), router.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(*first.Devices) != 1 || (*first.Devices)[0].Presence != PresenceNow {
		t.Fatalf("first %#v", first.Devices)
	}
	moment = moment.Add(5 * time.Minute)
	second, err := store.Snapshot(context.Background(), router.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(*second.Devices) != 1 || (*second.Devices)[0].Presence != PresenceNow {
		t.Fatalf("second %#v", second.Devices)
	}
	rows := presenceRows(t, store.presencePath)
	if len(rows) != 1 || rows[0].LastSeen != moment.Format(time.RFC3339) {
		t.Fatalf("rows %#v", rows)
	}
	if strings.Contains(presenceText(t, store.presencePath), routerPassword) || strings.Contains(presenceText(t, store.presencePath), vaultPassphrase) {
		t.Fatal("presence file includes a secret")
	}

	memory.Set("edge.example", withoutDevice)
	memory.Fail("edge.example", routeros.ErrUnreachable)
	if _, err := store.Snapshot(context.Background(), router.ID); err != nil {
		t.Fatal(err)
	}
	memory.Fail("edge.example", nil)
	if presenceRows(t, store.presencePath)[0].LastSeen != moment.Format(time.RFC3339) {
		t.Fatal("a lost session moved the last sighting")
	}

	moment = moment.Add(30 * time.Minute)
	recent, err := store.Snapshot(context.Background(), router.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(*recent.Devices) != 1 || (*recent.Devices)[0].Presence != PresenceRecent || (*recent.Devices)[0].Name != "lan-host-secret" {
		t.Fatalf("recent %#v", recent.Devices)
	}
	if len(presenceRows(t, store.presencePath)) != 1 {
		t.Fatal("a disappearance appended another row")
	}

	moment = moment.Add(2 * time.Hour)
	absent, err := store.Snapshot(context.Background(), router.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(*absent.Devices) != 1 || (*absent.Devices)[0].Presence != PresenceAbsent {
		t.Fatalf("absent %#v", absent.Devices)
	}

	moment = moment.Add(8 * 24 * time.Hour)
	gone, err := store.Snapshot(context.Background(), router.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(*gone.Devices) != 0 {
		t.Fatalf("kept an old device %#v", gone.Devices)
	}
	if strings.Contains(presenceText(t, store.presencePath), "lan-host-secret") {
		t.Fatal("an expired device stayed in the file")
	}
	events, err := os.ReadFile(store.historyPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(events), "lan-host-secret") {
		t.Fatal("device name entered the path history")
	}
}

func presenceRows(t *testing.T, path string) []storedPresence {
	t.Helper()
	var state presenceFile
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, &state); err != nil {
		t.Fatal(err)
	}
	return state.Devices
}

func presenceText(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}
