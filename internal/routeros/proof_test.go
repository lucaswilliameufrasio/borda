package routeros

import (
	"context"
	"strings"
	"testing"
)

func readyOutput() string {
	return "identity\tExample\t7.0\n" +
		"route\t192.0.2.1\ttrue\t1\t192.0.2.1%ether1\tborda-primary\n" +
		"forward\taccept\treserve\tbridge\tlte1\n" +
		"address\tbridge\t192.0.2.10/24\n" +
		"address\tlte1\t192.0.2.50/32\n" +
		"lte\tlte1\ttrue\n"
}

func TestShouldRefuseTheProofUntilTheReserveIsReady(t *testing.T) {
	reading := ParseReading(readyOutput())
	if _, err := PlanProof(reading, ""); err == nil {
		t.Fatal("accepted a proof without a check address")
	}
	bare := ParseReading("route\t192.0.2.1\ttrue\t1\t192.0.2.1%ether1\n")
	if _, err := PlanProof(bare, "192.0.2.1"); err == nil {
		t.Fatal("accepted a route without the primary comment")
	}
	memory := NewMemory()
	memory.Set("plain.example", "identity\tExample\t7.0\nroute\t192.0.2.1\ttrue\t1\t192.0.2.1%ether1\n")
	if _, err := memory.Prove(context.Background(), Target{Host: "plain.example", Password: "secret", CheckAddress: "192.0.2.1"}); err == nil {
		t.Fatal("proof started")
	}
	if len(memory.ProofCalls("plain.example")) != 0 {
		t.Fatal("proof wrote without a plan")
	}
	memory.Set("other.example", readyOutput())
	if _, err := memory.Prove(context.Background(), Target{Host: "ready-unused.example", Password: "secret", CheckAddress: "192.0.2.1"}); err == nil {
		t.Fatal("missing host started a proof")
	}
	if len(memory.ProofCalls("other.example")) != 0 {
		t.Fatal("another router was touched")
	}
}

func TestShouldRestoreThePrimaryRouteAfterTheMeasure(t *testing.T) {
	memory := NewMemory()
	memory.Set("ready.example", readyOutput())
	target := Target{Host: "ready.example", Password: "secret", CheckAddress: "192.0.2.1"}
	proof, err := memory.Prove(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	if proof.Outcome != OutcomePassed || !proof.PrimaryActive || proof.USBReceived != 1 || proof.LANReceived != 1 {
		t.Fatalf("proof %#v", proof)
	}
	if strings.Join(memory.ProofCalls("ready.example"), ",") != "disable,enable" {
		t.Fatalf("calls %v", memory.ProofCalls("ready.example"))
	}
	reading, err := memory.Read(context.Background(), target)
	if err != nil || !PrimaryIsActive(reading) {
		t.Fatal("primary route was left inactive")
	}

	memory.FailProofMeasure("ready.example")
	failed, err := memory.Prove(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	if failed.Outcome != OutcomeFailed || !failed.PrimaryActive || failed.USBReceived != 0 {
		t.Fatalf("failed proof %#v", failed)
	}
	if !PrimaryIsActive(mustRead(t, memory, target)) {
		t.Fatal("primary route was left inactive after a failed measure")
	}
}

func TestShouldReportWhenThePrimaryRouteStaysDown(t *testing.T) {
	memory := NewMemory()
	memory.Set("ready.example", readyOutput())
	memory.FailProofRestore("ready.example")
	target := Target{Host: "ready.example", Password: "secret", CheckAddress: "192.0.2.1"}
	proof, err := memory.Prove(context.Background(), target)
	if err != ErrProofNotRestored {
		t.Fatalf("err %v", err)
	}
	if proof.PrimaryActive {
		t.Fatal("reported the primary route as active")
	}
	if PrimaryIsActive(mustRead(t, memory, target)) {
		t.Fatal("reading still shows the primary route active")
	}
}

func TestShouldKeepProofScriptsOnTheCommentedRoute(t *testing.T) {
	if !strings.Contains(DisableScript(), PrimaryComment) || !strings.Contains(EnableScript(), PrimaryComment) {
		t.Fatal("script does not name the primary comment")
	}
	for _, script := range []string{DisableScript(), EnableScript()} {
		if strings.Contains(script, " remove ") || strings.Contains(script, " add ") {
			t.Fatal("proof script removes or adds configuration")
		}
	}
	script, err := PingScript("192.0.2.1", "lte1", "")
	if err != nil || !strings.Contains(script, `interface="lte1"`) || !strings.Contains(script, "proplist=received") {
		t.Fatal("usb measure")
	}
	script, err = PingScript("192.0.2.1", "", "192.0.2.10")
	if err != nil || !strings.Contains(script, `src-address="192.0.2.10"`) {
		t.Fatal("lan measure")
	}
	if _, err := PingScript("192.0.2.1", "lte1; /ip route remove [find]", ""); err == nil {
		t.Fatal("accepted an unsafe interface")
	}
}

func mustRead(t *testing.T, memory *Memory, target Target) Reading {
	t.Helper()
	reading, err := memory.Read(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	return reading
}
