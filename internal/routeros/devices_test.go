package routeros

import "testing"

func TestShouldJoinLeaseARPAndNeighborByMAC(t *testing.T) {
	reading := Reading{
		Leases: []Lease{{
			Address:  "192.0.2.20",
			MAC:      "AA:BB:CC:DD:EE:FF",
			HostName: "phone",
		}},
		ARP: []ARPEntry{
			{Address: "192.0.2.20", MAC: "aa-bb-cc-dd-ee-ff", InterfaceName: "bridge"},
			{Address: "192.0.2.21", MAC: "11:22:33:44:55:66", InterfaceName: "bridge"},
			{Address: "192.0.2.22", MAC: ""},
		},
		Neighbors: []Neighbor{{
			Address:       "192.0.2.21",
			MAC:           "11:22:33:44:55:66",
			Identity:      "camera",
			InterfaceName: "ether2",
		}},
	}
	devices := DevicesFrom(reading)
	if len(devices) != 2 {
		t.Fatalf("devices %#v", devices)
	}
	if devices[0].ID != "112233445566" || devices[0].Name != "camera" || devices[0].Address != "192.0.2.21" || devices[0].SeenOn != "bridge" {
		t.Fatalf("neighbor device %#v", devices[0])
	}
	if devices[1].ID != "aabbccddeeff" || devices[1].Name != "phone" || devices[1].MAC != "AA:BB:CC:DD:EE:FF" || devices[1].SeenOn != "bridge" {
		t.Fatalf("lease device %#v", devices[1])
	}
}

func TestShouldSkipAnUnusableMAC(t *testing.T) {
	devices := DevicesFrom(Reading{Leases: []Lease{{Address: "192.0.2.20", MAC: "not-a-mac", HostName: "phone"}}})
	if len(devices) != 0 {
		t.Fatalf("devices %#v", devices)
	}
}
