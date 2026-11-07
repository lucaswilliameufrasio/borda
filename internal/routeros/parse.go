package routeros

import (
	"strconv"
	"strings"
)

// ParseReading reads the tab-separated snapshot produced by ReadScript.
func ParseReading(output string) Reading {
	reading := Reading{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		switch fields[0] {
		case "identity":
			if len(fields) >= 3 {
				reading.IdentityName = fields[1]
				reading.IdentityVersion = fields[2]
			}
		case "route":
			if len(fields) >= 5 {
				distance, _ := strconv.Atoi(fields[3])
				reading.Routes = append(reading.Routes, Route{
					Gateway:       fields[1],
					Active:        fields[2] == "true",
					Distance:      distance,
					InterfaceName: interfaceFromImmediate(fields[4], fields[1]),
				})
			}
		case "forward":
			if len(fields) >= 5 {
				reading.Forward = append(reading.Forward, ForwardRule{
					Action:       fields[1],
					Comment:      fields[2],
					InInterface:  fields[3],
					OutInterface: fields[4],
				})
			}
		case "address":
			if len(fields) >= 3 {
				reading.Addresses = append(reading.Addresses, Address{
					InterfaceName: fields[1],
					Value:         fields[2],
				})
			}
		case "lte":
			if len(fields) >= 3 {
				reading.LTE = append(reading.LTE, LTE{
					Name:    fields[1],
					Running: fields[2] == "true",
				})
			}
		case "lease":
			if len(fields) >= 3 {
				hostName := ""
				if len(fields) >= 4 {
					hostName = fields[3]
				}
				reading.Leases = append(reading.Leases, Lease{
					Address:  fields[1],
					MAC:      fields[2],
					HostName: hostName,
				})
			}
		case "check":
			if len(fields) >= 2 {
				received, err := strconv.Atoi(fields[1])
				if err == nil {
					reading.CheckReceived = &received
				}
			}
		}
	}
	return reading
}

func interfaceFromImmediate(immediate string, gateway string) string {
	if strings.Contains(immediate, "%") {
		parts := strings.SplitN(immediate, "%", 2)
		if len(parts) == 2 {
			return parts[1]
		}
	}
	if gateway != "" && !strings.Contains(gateway, ".") && !strings.Contains(gateway, ":") {
		return gateway
	}
	return ""
}

// ActiveRoute returns the active default with the lowest distance.
func ActiveRoute(routes []Route) *Route {
	var chosen *Route
	for index := range routes {
		route := routes[index]
		if !route.Active {
			continue
		}
		if chosen == nil || route.Distance < chosen.Distance {
			copy := route
			chosen = &copy
		}
	}
	return chosen
}
