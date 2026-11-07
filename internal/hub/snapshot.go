package hub

import "github.com/lucaswilliameufrasio/borda/internal/routeros"

// Snapshot is the read-only portrait of one router.
type Snapshot struct {
	RouterID     string           `json:"router_id"`
	Name         string           `json:"name"`
	Session      string           `json:"session"`
	Identity     *IdentityJSON    `json:"identity,omitempty"`
	ActiveRoute  *ActiveRouteJSON `json:"active_route,omitempty"`
	Check        *CheckJSON       `json:"check,omitempty"`
	Routes       *[]RouteJSON     `json:"routes,omitempty"`
	ForwardRules *[]ForwardJSON   `json:"forward_rules,omitempty"`
	Leases       *[]LeaseJSON     `json:"leases,omitempty"`
	Devices      *[]DeviceJSON    `json:"devices,omitempty"`
	Reserve      *ReserveJSON     `json:"reserve,omitempty"`
}

type IdentityJSON struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

type ActiveRouteJSON struct {
	Gateway       string `json:"gateway"`
	InterfaceName string `json:"interface_name,omitempty"`
	Distance      int    `json:"distance"`
}

type CheckJSON struct {
	Address    string `json:"address"`
	Responding bool   `json:"responding"`
}

type RouteJSON struct {
	Gateway       string `json:"gateway"`
	Active        bool   `json:"active"`
	Distance      int    `json:"distance"`
	InterfaceName string `json:"interface_name,omitempty"`
}

type ForwardJSON struct {
	Action       string `json:"action"`
	Comment      string `json:"comment,omitempty"`
	InInterface  string `json:"in_interface,omitempty"`
	OutInterface string `json:"out_interface,omitempty"`
}

type LeaseJSON struct {
	Address  string `json:"address"`
	MAC      string `json:"mac"`
	HostName string `json:"host_name,omitempty"`
}

type DeviceJSON struct {
	ID      string `json:"id"`
	MAC     string `json:"mac"`
	Address string `json:"address,omitempty"`
	Name    string `json:"name,omitempty"`
	SeenOn  string `json:"seen_on,omitempty"`
}

type ReserveJSON struct {
	InterfaceName string `json:"interface_name"`
	Running       bool   `json:"running"`
	HasAddress    bool   `json:"has_address"`
}

func fillConnected(snapshot Snapshot, router storedRouter, reading routeros.Reading) Snapshot {
	if reading.IdentityName != "" || reading.IdentityVersion != "" {
		snapshot.Identity = &IdentityJSON{Name: reading.IdentityName, Version: reading.IdentityVersion}
	}
	routes := make([]RouteJSON, 0, len(reading.Routes))
	for _, route := range reading.Routes {
		routes = append(routes, RouteJSON{
			Gateway:       route.Gateway,
			Active:        route.Active,
			Distance:      route.Distance,
			InterfaceName: route.InterfaceName,
		})
	}
	snapshot.Routes = &routes
	active := routeros.ActiveRoute(reading.Routes)
	if active != nil {
		snapshot.ActiveRoute = &ActiveRouteJSON{
			Gateway:       active.Gateway,
			InterfaceName: active.InterfaceName,
			Distance:      active.Distance,
		}
	}
	rules := make([]ForwardJSON, 0, len(reading.Forward))
	for _, rule := range reading.Forward {
		rules = append(rules, ForwardJSON{
			Action:       rule.Action,
			Comment:      rule.Comment,
			InInterface:  rule.InInterface,
			OutInterface: rule.OutInterface,
		})
	}
	snapshot.ForwardRules = &rules
	leases := make([]LeaseJSON, 0, len(reading.Leases))
	for _, lease := range reading.Leases {
		leases = append(leases, LeaseJSON{
			Address:  lease.Address,
			MAC:      lease.MAC,
			HostName: lease.HostName,
		})
	}
	snapshot.Leases = &leases
	joined := routeros.DevicesFrom(reading)
	devices := make([]DeviceJSON, 0, len(joined))
	for _, device := range joined {
		devices = append(devices, DeviceJSON{
			ID:      device.ID,
			MAC:     device.MAC,
			Address: device.Address,
			Name:    device.Name,
			SeenOn:  device.SeenOn,
		})
	}
	snapshot.Devices = &devices
	if router.CheckAddress != "" {
		responding := false
		if reading.CheckReceived != nil && *reading.CheckReceived > 0 {
			responding = true
		}
		snapshot.Check = &CheckJSON{Address: router.CheckAddress, Responding: responding}
	}
	for _, modem := range reading.LTE {
		if !modem.Running {
			continue
		}
		hasAddress := false
		for _, address := range reading.Addresses {
			if address.InterfaceName == modem.Name && address.Value != "" {
				hasAddress = true
			}
		}
		if hasAddress {
			snapshot.Reserve = &ReserveJSON{
				InterfaceName: modem.Name,
				Running:       true,
				HasAddress:    true,
			}
			break
		}
	}
	return snapshot
}
