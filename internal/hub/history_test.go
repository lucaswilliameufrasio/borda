package hub

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lucaswilliameufrasio/borda/internal/routeros"
	"github.com/lucaswilliameufrasio/borda/internal/secret"
)

type countReader struct {
	routeros.Reader
	mu    sync.Mutex
	reads int
}

func (reader *countReader) Read(ctx context.Context, target routeros.Target) (routeros.Reading, error) {
	reader.mu.Lock()
	reader.reads++
	reader.mu.Unlock()
	return reader.Reader.Read(ctx, target)
}

func (reader *countReader) readCount() int {
	reader.mu.Lock()
	defer reader.mu.Unlock()
	return reader.reads
}

func TestShouldRecordPathChangesInTheHub(t *testing.T) {
	dir := t.TempDir()
	memory := routeros.NewMemory()
	memory.Set("edge.example", "identity\tExample\t7.0\nroute\t192.0.2.1\ttrue\t1\t192.0.2.1%ether1\ncheck\t1\n")
	memory.Set("plain.example", "identity\tExample\t7.0\nroute\t192.0.2.1\ttrue\t1\t192.0.2.1%ether1\ncheck\t1\n")
	memory.Set("ready.example", "identity\tExample\t7.0\n"+
		"route\t192.0.2.1\ttrue\t1\t192.0.2.1%ether1\tborda-primary\n"+
		"forward\taccept\treserve\tbridge\tlte1\n"+
		"address\tbridge\t192.0.2.10/24\n"+
		"address\tlte1\t192.0.2.50/32\n"+
		"lte\tlte1\ttrue\n"+
		"check\t1\n")
	reader := &countReader{Reader: memory}
	store, err := Open(dir, secret.NewKeeper(dir, secret.NewMemoryKeyring(), nil), reader)
	if err != nil {
		t.Fatal(err)
	}
	tick := 0
	store.now = func() time.Time {
		tick++
		return time.Unix(1_700_000_000+int64(tick), 0).UTC()
	}

	edge, err := store.Register(context.Background(), fileInput("Borda", "edge.example", "192.0.2.1"))
	if err != nil {
		t.Fatal(err)
	}
	plain, err := store.Register(context.Background(), fileInput("Sem prova", "plain.example", "192.0.2.1"))
	if err != nil {
		t.Fatal(err)
	}
	ready, err := store.Register(context.Background(), fileInput("Pronta", "ready.example", "192.0.2.1"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.Snapshot(context.Background(), edge.ID); err != nil {
		t.Fatal(err)
	}
	events, err := store.Events(edge.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("baseline created an event: %#v", events)
	}
	readsAfterEvents := reader.readCount()
	if _, err := store.Events(edge.ID); err != nil {
		t.Fatal(err)
	}
	if reader.readCount() != readsAfterEvents {
		t.Fatal("history read the router")
	}

	memory.Set("edge.example", "identity\tExample\t7.0\nroute\t192.0.2.9\ttrue\t2\t192.0.2.9%lte1\ncheck\t0\n")
	if _, err := store.Snapshot(context.Background(), edge.ID); err != nil {
		t.Fatal(err)
	}
	memory.Set("edge.example", "identity\tExample\t7.0\nroute\t192.0.2.8\ttrue\t1\t192.0.2.8%ether1\ncheck\t1\n")
	if _, err := store.Snapshot(context.Background(), edge.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Prove(context.Background(), plain.ID, true); err == nil {
		t.Fatal("proof started without a reserve")
	}
	if _, err := store.Prove(context.Background(), ready.ID, true); err != nil {
		t.Fatal(err)
	}

	events, err = store.Events(edge.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Reason != ReasonCheckFailed {
		t.Fatalf("edge events %#v", events)
	}
	plainEvents, err := store.Events(plain.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(plainEvents) != 0 {
		t.Fatalf("refused proof recorded %#v", plainEvents)
	}
	readyEvents, err := store.Events(ready.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(readyEvents) != 2 || readyEvents[0].Reason != ReasonRouteRestored || readyEvents[1].Reason != ReasonProofRequested {
		t.Fatalf("proof events %#v", readyEvents)
	}
	if readyEvents[0].At == readyEvents[1].At {
		t.Fatal("proof events share one timestamp")
	}

	body, err := os.ReadFile(store.historyPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), routerPassword) || strings.Contains(string(body), vaultPassphrase) {
		t.Fatal("history file contains a secret")
	}

	reopened, err := Open(dir, secret.NewKeeper(dir, secret.NewMemoryKeyring(), nil), routeros.NewMemory())
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := reopened.Events(edge.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded) != 1 || reloaded[0].Reason != ReasonCheckFailed || reloaded[0].ID != events[0].ID {
		t.Fatalf("reloaded %#v", reloaded)
	}
	memory.Fail("edge.example", routeros.ErrUnreachable)
	if _, err := store.Events(edge.ID); err != nil {
		t.Fatal(err)
	}
}

func TestShouldLeaveTheRouteRestoredEventOutWhenRestoreFails(t *testing.T) {
	dir := t.TempDir()
	memory := routeros.NewMemory()
	memory.Set("ready.example", "identity\tExample\t7.0\n"+
		"route\t192.0.2.1\ttrue\t1\t192.0.2.1%ether1\tborda-primary\n"+
		"forward\taccept\treserve\tbridge\tlte1\n"+
		"address\tbridge\t192.0.2.10/24\n"+
		"address\tlte1\t192.0.2.50/32\n"+
		"lte\tlte1\ttrue\n")
	memory.FailProofRestore("ready.example")
	store, err := Open(dir, secret.NewKeeper(dir, secret.NewMemoryKeyring(), nil), memory)
	if err != nil {
		t.Fatal(err)
	}
	ready, err := store.Register(context.Background(), fileInput("Pronta", "ready.example", "192.0.2.1"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Prove(context.Background(), ready.ID, true); err != routeros.ErrProofNotRestored {
		t.Fatal(err)
	}
	events, err := store.Events(ready.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Reason != ReasonProofRequested {
		t.Fatalf("events %#v", events)
	}
}
