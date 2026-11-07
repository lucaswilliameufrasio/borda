package main

import "github.com/lucaswilliameufrasio/borda/internal/routeros"

// seedMemory installs example hosts for the UI. They are not real routers.
func seedMemory(reader *routeros.Memory) {
	reader.Set("connected.example", ""+
		"identity\tExample\t7.0\n"+
		"route\t192.0.2.1\ttrue\t1\t192.0.2.1%ether1\tborda-primary\n"+
		"forward\taccept\tlan\tbridge\tether1\n"+
		"forward\taccept\treserve\tbridge\tlte1\n"+
		"address\tbridge\t192.0.2.10/24\n"+
		"address\tlte1\t192.0.2.50/32\n"+
		"lte\tlte1\ttrue\n"+
		"lease\t192.0.2.20\tAA:BB:CC:DD:EE:FF\tphone\n"+
		"check\t1\n")
	reader.Set("other.example", ""+
		"identity\tOther\t7.0\n"+
		"route\t192.0.2.2\ttrue\t1\t192.0.2.2%ether2\n"+
		"lease\t192.0.2.30\tAA:BB:CC:DD:EE:01\tlaptop\n"+
		"check\t1\n")
	reader.Set("empty.example", "identity\tEmpty\t7.0\n")
	reader.Fail("offline.example", routeros.ErrUnreachable)
}
