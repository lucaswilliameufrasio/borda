package assistant

import "testing"

func TestShouldProposeOnlyTheClosedProofAction(t *testing.T) {
	connected := Portrait{
		Session:         "connected",
		HasActiveRoute:  true,
		ActiveGateway:   "192.0.2.1",
		ActiveInterface: "ether1",
		ActiveDistance:  1,
		HasCheck:        true,
		CheckAddress:    "192.0.2.1",
		CheckResponding: true,
		LeaseCount:      1,
		DeviceCount:     2,
	}
	answer := Propose(connected, "como está o caminho?")
	if answer.Kind != KindAnswer || answer.Proposal != nil {
		t.Fatalf("answer %#v", answer)
	}
	if answer.Text == "" {
		t.Fatal("empty answer")
	}

	proposal := Propose(connected, "pode rodar a prova usb?")
	if proposal.Kind != KindProposal || proposal.Proposal == nil || proposal.Proposal.Action != ActionProof {
		t.Fatalf("proposal %#v", proposal)
	}

	refused := Propose(connected, "faz um export e um backup")
	if refused.Kind != KindRefused || refused.Proposal != nil {
		t.Fatalf("refused %#v", refused)
	}
	user := Propose(connected, "cria um usuário novo")
	if user.Kind != KindRefused {
		t.Fatalf("user %#v", user)
	}

	locked := Propose(Portrait{Session: "locked"}, "como está o caminho?")
	if locked.Kind != KindMissing || len(locked.Missing) != 1 || locked.Missing[0] != "session" {
		t.Fatalf("locked %#v", locked)
	}
	empty := Propose(Portrait{Session: "connected"}, "como está o caminho?")
	if empty.Kind != KindMissing || len(empty.Missing) != 1 || empty.Missing[0] != "active_route" {
		t.Fatalf("empty %#v", empty)
	}
}
