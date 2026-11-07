package hub

import (
	"context"
	"strings"
	"testing"

	"github.com/lucaswilliameufrasio/borda/internal/assistant"
	"github.com/lucaswilliameufrasio/borda/internal/routeros"
	"github.com/lucaswilliameufrasio/borda/internal/secret"
)

func TestShouldKeepTheAssistantOffTheRouterUntilTheProofIsConfirmed(t *testing.T) {
	dir := t.TempDir()
	memory := routeros.NewMemory()
	memory.Set("edge.example", "identity\tExample\t7.0\n"+
		"route\t192.0.2.1\ttrue\t1\t192.0.2.1%ether1\tborda-primary\n"+
		"forward\taccept\treserve\tbridge\tlte1\n"+
		"address\tbridge\t192.0.2.10/24\n"+
		"address\tlte1\t192.0.2.50/32\n"+
		"lte\tlte1\ttrue\n"+
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

	answer, err := store.Ask(context.Background(), edge.ID, "como está o caminho?")
	if err != nil {
		t.Fatal(err)
	}
	if answer.Kind != assistant.KindAnswer || strings.Contains(answer.Text, "lan-host-secret") || strings.Contains(answer.Text, routerPassword) {
		t.Fatalf("answer %#v", answer)
	}
	refused, err := store.Ask(context.Background(), edge.ID, "faz um export")
	if err != nil {
		t.Fatal(err)
	}
	if refused.Kind != assistant.KindRefused {
		t.Fatalf("refused %#v", refused)
	}
	proposal, err := store.Ask(context.Background(), edge.ID, "roda a prova usb")
	if err != nil {
		t.Fatal(err)
	}
	if proposal.Kind != assistant.KindProposal || proposal.Proposal == nil || proposal.Proposal.Action != assistant.ActionProof {
		t.Fatalf("proposal %#v", proposal)
	}
	if len(memory.ProofCalls("edge.example")) != 0 {
		t.Fatal("asking called the proof")
	}

	if _, err := store.ApplyProposal(context.Background(), edge.ID, "export", true); err == nil {
		t.Fatal("export was accepted")
	}
	if _, err := store.ApplyProposal(context.Background(), edge.ID, assistant.ActionProof, false); err == nil {
		t.Fatal("unconfirmed proof was accepted")
	}
	if len(memory.ProofCalls("edge.example")) != 0 {
		t.Fatal("refused proposal wrote to the router")
	}
	if _, err := store.ApplyProposal(context.Background(), edge.ID, assistant.ActionProof, true); err != nil {
		t.Fatal(err)
	}
	calls := memory.ProofCalls("edge.example")
	if len(calls) != 2 || calls[0] != "disable" || calls[1] != "enable" {
		t.Fatalf("confirmed proof calls %#v", calls)
	}

	reopened, err := Open(dir, secret.NewKeeper(dir, secret.NewMemoryKeyring(), nil), reader)
	if err != nil {
		t.Fatal(err)
	}
	reads := reader.readCount()
	locked, err := reopened.Ask(context.Background(), edge.ID, "como está o caminho?")
	if err != nil {
		t.Fatal(err)
	}
	if locked.Kind != assistant.KindMissing || reader.readCount() != reads {
		t.Fatalf("locked %#v reads %d", locked, reader.readCount())
	}
	if strings.Contains(locked.Text, routerPassword) || strings.Contains(locked.Text, vaultPassphrase) {
		t.Fatal("locked reply contains a secret")
	}
}
