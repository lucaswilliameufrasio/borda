package routeros

import (
	"sort"
	"strings"
)

// ARPEntry is one address seen on an interface.
type ARPEntry struct {
	Address       string
	MAC           string
	InterfaceName string
}

// Neighbor is one discovered neighbor.
type Neighbor struct {
	Address       string
	MAC           string
	Identity      string
	InterfaceName string
}

// Device is one LAN host joined by MAC.
// The exit path is the router's active route, not a per-device policy.
type Device struct {
	ID      string
	MAC     string
	Address string
	Name    string
	SeenOn  string
}

// DevicesFrom joins DHCP leases, ARP and neighbors that share a MAC.
// A later source replaces an earlier address, name or interface only when its value is not empty.
// Lease is last, so the announced host name wins over the neighbor identity.
func DevicesFrom(reading Reading) []Device {
	byID := map[string]*Device{}
	for _, neighbor := range reading.Neighbors {
		device, ok := deviceFor(byID, neighbor.MAC)
		if !ok {
			continue
		}
		fillDevice(device, neighbor.Address, neighbor.Identity, neighbor.InterfaceName)
	}
	for _, entry := range reading.ARP {
		device, ok := deviceFor(byID, entry.MAC)
		if !ok {
			continue
		}
		fillDevice(device, entry.Address, "", entry.InterfaceName)
	}
	for _, lease := range reading.Leases {
		device, ok := deviceFor(byID, lease.MAC)
		if !ok {
			continue
		}
		fillDevice(device, lease.Address, lease.HostName, "")
	}
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	devices := make([]Device, 0, len(ids))
	for _, id := range ids {
		devices = append(devices, *byID[id])
	}
	return devices
}

func deviceFor(byID map[string]*Device, mac string) (*Device, bool) {
	display, id, ok := canonicalMAC(mac)
	if !ok {
		return nil, false
	}
	device, found := byID[id]
	if !found {
		device = &Device{ID: id, MAC: display}
		byID[id] = device
	}
	return device, true
}

func fillDevice(device *Device, address string, name string, seenOn string) {
	address = strings.TrimSpace(address)
	name = strings.TrimSpace(name)
	seenOn = strings.TrimSpace(seenOn)
	if address != "" {
		device.Address = address
	}
	if name != "" {
		device.Name = name
	}
	if seenOn != "" {
		device.SeenOn = seenOn
	}
}

func canonicalMAC(value string) (string, string, bool) {
	var raw strings.Builder
	for _, char := range value {
		if char == ':' || char == '-' {
			continue
		}
		if !isHex(char) {
			return "", "", false
		}
		raw.WriteRune(lowerHex(char))
	}
	id := raw.String()
	if len(id) != 12 {
		return "", "", false
	}
	display := strings.ToUpper(id[0:2] + ":" + id[2:4] + ":" + id[4:6] + ":" + id[6:8] + ":" + id[8:10] + ":" + id[10:12])
	return display, id, true
}

func isHex(char rune) bool {
	if char >= '0' && char <= '9' {
		return true
	}
	if char >= 'a' && char <= 'f' {
		return true
	}
	if char >= 'A' && char <= 'F' {
		return true
	}
	return false
}

func lowerHex(char rune) rune {
	if char >= 'A' && char <= 'F' {
		return char + ('a' - 'A')
	}
	return char
}
