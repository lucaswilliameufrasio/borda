package hub

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/lucaswilliameufrasio/borda/internal/routeros"
	"github.com/lucaswilliameufrasio/borda/internal/secret"
)

func TestShouldPublishThePortraitWithoutThePassword(t *testing.T) {
	dir := t.TempDir()
	memory := routeros.NewMemory()
	memory.Set("edge.example", "identity\tExample\t7.0\n"+
		"route\t192.0.2.1\ttrue\t1\t192.0.2.1%ether1\n"+
		"lease\t192.0.2.20\tAA:BB:CC:DD:EE:FF\tlan-host-secret\n"+
		"check\t1\n")
	reader := &countReader{Reader: memory}
	store, err := Open(dir, secret.NewKeeper(dir, secret.NewMemoryKeyring(), nil), reader)
	if err != nil {
		t.Fatal(err)
	}
	edge, err := store.Register(context.Background(), fileInput("Borda", "edge.example", "192.0.2.1"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Link(context.Background(), edge.ID, false); err == nil {
		t.Fatal("unconfirmed link was stored")
	}
	before, err := store.Hub(context.Background(), edge.ID)
	if err != nil {
		t.Fatal(err)
	}
	if before.Linked || before.Portrait != nil || before.Events != nil {
		t.Fatalf("published before approval %#v", before)
	}
	view, err := store.Link(context.Background(), edge.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if !view.Linked || view.Portrait == nil || view.Events == nil || view.Name != "Borda" {
		t.Fatalf("view %#v", view)
	}
	if view.Portrait.ActiveGateway != "192.0.2.1" || view.Portrait.LeaseCount != 1 {
		t.Fatalf("portrait %#v", view.Portrait)
	}
	encoded, err := os.ReadFile(store.linksPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	if strings.Contains(text, routerPassword) || strings.Contains(text, vaultPassphrase) || strings.Contains(text, "reader") || strings.Contains(text, "lan-host-secret") {
		t.Fatal("link file contains a secret or user content")
	}
	if err := store.RefuseHubWrite(edge.ID); err == nil {
		t.Fatal("hub accepted a write")
	}
	if len(memory.ProofCalls("edge.example")) != 0 {
		t.Fatal("hub write called the proof")
	}

	reopened, err := Open(dir, secret.NewKeeper(dir, secret.NewMemoryKeyring(), nil), reader)
	if err != nil {
		t.Fatal(err)
	}
	reads := reader.readCount()
	locked, err := reopened.Hub(context.Background(), edge.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !locked.Linked || locked.Portrait == nil || locked.Portrait.Session != "locked" || reader.readCount() != reads {
		t.Fatalf("locked hub %#v reads %d", locked.Portrait, reader.readCount())
	}
	if strings.Contains(locked.Name, routerPassword) {
		t.Fatal("locked hub view contains a secret")
	}
}
