package routeros

import "testing"

func TestShouldParseActiveRouteAndRejectWritesInScript(t *testing.T) {
	script, err := ReadScript("192.0.2.1")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(script, `/ping address="192.0.2.1"`) {
		t.Fatal("script missing check")
	}
	for _, word := range []string{" add ", " set ", " remove "} {
		if contains(script, word) {
			t.Fatalf("script contains %q", word)
		}
	}
	if _, err := ReadScript("192.0.2.1; /system reboot"); err == nil {
		t.Fatal("accepted an unsafe check address")
	}

	reading := ParseReading("identity\tEdge\t7.23.7\n" +
		"route\t192.0.2.1\ttrue\t1\t192.0.2.1%ether1\n" +
		"route\t10.0.0.1\tfalse\t2\t10.0.0.1%lte1\n" +
		"forward\taccept\tlan\tbridge\tether1\n" +
		"address\tlte1\t10.40.0.2/24\n" +
		"lte\tlte1\ttrue\n" +
		"lease\t192.0.2.20\tAA:BB:CC:DD:EE:FF\tphone\n" +
		"check\t1\n")
	active := ActiveRoute(reading.Routes)
	if active == nil || active.InterfaceName != "ether1" || active.Distance != 1 {
		t.Fatalf("active %#v", active)
	}
	if len(reading.Leases) != 1 || reading.Leases[0].HostName != "phone" {
		t.Fatalf("leases %#v", reading.Leases)
	}
	if reading.CheckReceived == nil || *reading.CheckReceived != 1 {
		t.Fatal("check")
	}
}

func contains(value string, needle string) bool {
	return len(value) >= len(needle) && (value == needle || len(value) > 0 && indexOf(value, needle) >= 0)
}

func indexOf(value string, needle string) int {
	for start := 0; start+len(needle) <= len(value); start++ {
		if value[start:start+len(needle)] == needle {
			return start
		}
	}
	return -1
}
