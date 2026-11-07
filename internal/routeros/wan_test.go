package routeros

import (
	"strings"
	"testing"
)

func TestShouldRegisterOnlyTheCommentedWan(t *testing.T) {
	reading := ParseReading("identity\tExample\t7.0\n" +
		"route\t192.0.2.1\ttrue\t1\t192.0.2.1%ether1\tborda-primary\n" +
		"address\tbridge\t192.0.2.10/24\n" +
		"address\tether1\t192.0.2.40/24\n" +
		"bridge\tether2\tbridge\n")
	if len(reading.BridgePorts) != 1 || reading.BridgePorts[0].InterfaceName != "ether2" {
		t.Fatalf("bridge %#v", reading.BridgePorts)
	}
	plan, err := PlanWan(reading, WanRequest{Name: "Cabo", InterfaceName: "ether2", Distance: 2})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Comment != "borda-wan-cabo" || plan.AlreadyPresent {
		t.Fatalf("plan %#v", plan)
	}
	scripts, err := WanScripts(plan)
	if err != nil {
		t.Fatal(err)
	}
	if len(scripts) != 3 {
		t.Fatalf("scripts %d", len(scripts))
	}
	joined := strings.Join(scripts, "\n")
	if strings.Contains(joined, PrimaryComment) {
		t.Fatal("wan script names the primary route")
	}
	if !strings.Contains(joined, `comment="borda-wan-cabo"`) || !strings.Contains(joined, `interface="ether2"`) {
		t.Fatalf("scripts %s", joined)
	}
	for _, script := range scripts {
		if strings.Contains(script, "{") && !strings.Contains(script, "}") {
			t.Fatalf("split block %s", script)
		}
	}

	occupied, err := PlanWan(reading, WanRequest{Name: "Cabo", InterfaceName: "bridge", Distance: 2})
	if err == nil || occupied.Comment != "" {
		t.Fatal("accepted the lan port")
	}
	var refusal *WanError
	if !asWanError(err, &refusal) || refusal.Reason != ReasonNotSpare {
		t.Fatalf("lan %#v", err)
	}
	taken, err := PlanWan(reading, WanRequest{Name: "Cabo", InterfaceName: "ether2", Distance: 1})
	if err == nil || taken.Comment != "" {
		t.Fatal("accepted a used distance")
	}
	if !asWanError(err, &refusal) || refusal.Reason != ReasonDistanceTaken {
		t.Fatalf("distance %#v", err)
	}

	reading.DHCPClients = []DHCPClient{{InterfaceName: "ether2", Distance: 2, Comment: "borda-wan-cabo"}}
	reading.NAT = []NATRule{{Chain: "srcnat", Action: "masquerade", InterfaceName: "ether2", Comment: "borda-wan-cabo"}}
	again, err := PlanWan(reading, WanRequest{Name: "cabo", InterfaceName: "ether2", Distance: 2})
	if err != nil || !again.AlreadyPresent {
		t.Fatalf("repeat %#v %v", again, err)
	}
	repeatScripts, err := WanScripts(again)
	if err != nil || len(repeatScripts) != 0 {
		t.Fatalf("repeat scripts %#v %v", repeatScripts, err)
	}
}

func asWanError(err error, target **WanError) bool {
	found, ok := err.(*WanError)
	if ok {
		*target = found
	}
	return ok
}
